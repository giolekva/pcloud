package installer

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/netip"
	"path/filepath"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/giolekva/pcloud/core/installer/soft"
)

const IPAddressPoolLocal = "local"
const IPAddressPoolConfigRepo = "config-repo"
const IPAddressPoolIngressPublic = "ingress-public"

const dnsAPIConfigMapName = "api-config"

type Bootstrapper struct {
	cl      ChartLoader
	ns      NamespaceCreator
	ha      HelmActionConfigFactory
	appRepo AppRepository
}

func NewBootstrapper(cl ChartLoader, ns NamespaceCreator, ha HelmActionConfigFactory, appRepo AppRepository) Bootstrapper {
	return Bootstrapper{cl, ns, ha, appRepo}
}

func (b Bootstrapper) Run(env EnvConfig) error {
	if err := b.ns.Create(env.Name); err != nil {
		return err
	}
	if err := b.installMetallb(env); err != nil {
		return err
	}
	if err := b.installLonghorn(env.Name, env.StorageDir, env.VolumeDefaultReplicaCount); err != nil {
		return err
	}
	bootstrapJobKeys, err := NewSSHKeyPair("bootstrapper")
	if err != nil {
		return err
	}
	if err := b.installSoftServe(bootstrapJobKeys.AuthorizedKey(), env.Name, env.ServiceIPs.ConfigRepo); err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	ss, err := soft.WaitForClient(
		netip.AddrPortFrom(env.ServiceIPs.ConfigRepo, 22).String(),
		bootstrapJobKeys.RawPrivateKey(),
		log.Default())
	if err != nil {
		return err
	}
	defer func() {
		if ss.RemovePublicKey("admin", bootstrapJobKeys.AuthorizedKey()); err != nil {
			fmt.Printf("Failed to remove admin public key: %s\n", err.Error())
		}
	}()
	if ss.AddPublicKey("admin", string(env.AdminPublicKey)); err != nil {
		return err
	}
	if err := b.installFluxcd(ss, env.Name); err != nil {
		return err
	}
	fmt.Println("Fluxcd installed")
	repo, err := ss.GetRepo("config")
	if err != nil {
		fmt.Println("Failed to get config repo")
		return err
	}
	repoIO, err := NewRepoIO(repo, ss.Signer)
	if err != nil {
		return err
	}
	fmt.Println("Configuring main repo")
	if err := configureMainRepo(repoIO, env); err != nil {
		return err
	}
	fmt.Println("Installing infrastructure services")
	if err := b.installInfrastructureServices(repoIO, env); err != nil {
		return err
	}
	fmt.Println("Installing DNS Zone Manager")
	if err := b.installDNSZoneManager(repoIO, env); err != nil {
		return err
	}
	fmt.Println("Installing Fluxcd Reconciler")
	if err := b.installFluxcdReconciler(repoIO, ss, env); err != nil {
		return err
	}
	fmt.Println("Installing env manager")
	if err := b.installEnvManager(repoIO, ss, env); err != nil {
		return err
	}
	fmt.Println("Installing Ory Hydra Maester")
	if err := b.installOryHydraMaester(repoIO, env); err != nil {
		return err
	}
	fmt.Println("Environment ready to use")
	return nil
}

func (b Bootstrapper) installMetallb(env EnvConfig) error {
	if err := b.installMetallbNamespace(env); err != nil {
		return err
	}
	if err := b.installMetallbService(); err != nil {
		return err
	}
	if err := b.installMetallbIPAddressPool(IPAddressPoolLocal, true, env.ServiceIPs.From, env.ServiceIPs.To); err != nil {
		return err
	}
	if err := b.installMetallbIPAddressPool(IPAddressPoolConfigRepo, false, env.ServiceIPs.ConfigRepo, env.ServiceIPs.ConfigRepo); err != nil {
		return err
	}
	if err := b.installMetallbIPAddressPool(IPAddressPoolIngressPublic, false, env.ServiceIPs.IngressPublic, env.ServiceIPs.IngressPublic); err != nil {
		return err
	}
	return nil
}

func (b Bootstrapper) installMetallbNamespace(env EnvConfig) error {
	fmt.Println("Installing metallb namespace")
	config, err := b.ha.New(env.Name)
	if err != nil {
		return err
	}
	chart, err := b.cl.Load("namespace")
	if err != nil {
		return err
	}
	values := map[string]any{
		"namespace": "metallb-system",
		"labels": []string{
			"pod-security.kubernetes.io/audit: privileged",
			"pod-security.kubernetes.io/enforce: privileged",
			"pod-security.kubernetes.io/warn: privileged",
		},
	}
	installer := action.NewInstall(config)
	installer.Namespace = env.Name
	installer.ReleaseName = "metallb-ns"
	installer.Wait = true
	installer.WaitForJobs = true
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func (b Bootstrapper) installMetallbService() error {
	fmt.Println("Installing metallb")
	config, err := b.ha.New("metallb-system")
	if err != nil {
		return err
	}
	chart, err := b.cl.Load("metallb")
	if err != nil {
		return err
	}
	values := map[string]any{ // TODO(giolekva): add loadBalancerClass?
		"controller": map[string]any{
			"image": map[string]any{
				"repository": "quay.io/metallb/controller",
				"tag":        "v0.13.12",
				"pullPolicy": "IfNotPresent",
			},
			"logLevel": "info",
		},
		"speaker": map[string]any{
			"image": map[string]any{
				"repository": "quay.io/metallb/speaker",
				"tag":        "v0.13.12",
				"pullPolicy": "IfNotPresent",
			},
			"logLevel": "info",
		},
	}
	installer := action.NewInstall(config)
	installer.Namespace = "metallb-system"
	installer.CreateNamespace = true
	installer.ReleaseName = "metallb"
	installer.IncludeCRDs = true
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 20 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func (b Bootstrapper) installMetallbIPAddressPool(name string, autoAssign bool, from, to netip.Addr) error {
	fmt.Printf("Installing metallb-ipaddresspool: %s\n", name)
	config, err := b.ha.New("metallb-system")
	if err != nil {
		return err
	}
	chart, err := b.cl.Load("metallb-ipaddresspool")
	if err != nil {
		return err
	}
	values := map[string]any{
		"name":       name,
		"autoAssign": autoAssign,
		"from":       from.String(),
		"to":         to.String(),
	}
	installer := action.NewInstall(config)
	installer.Namespace = "metallb-system"
	installer.CreateNamespace = true
	installer.ReleaseName = name
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 20 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func (b Bootstrapper) installLonghorn(envName string, storageDir string, volumeDefaultReplicaCount int) error {
	fmt.Println("Installing Longhorn")
	config, err := b.ha.New(envName)
	if err != nil {
		return err
	}
	chart, err := b.cl.Load("longhorn")
	if err != nil {
		return err
	}
	values := map[string]any{
		"defaultSettings": map[string]any{
			"defaultDataPath": storageDir,
		},
		"persistence": map[string]any{
			"defaultClassReplicaCount": volumeDefaultReplicaCount,
		},
		"service": map[string]any{
			"ui": map[string]any{
				"type": "LoadBalancer",
			},
		},
		"ingress": map[string]any{
			"enabled": false,
		},
	}
	installer := action.NewInstall(config)
	installer.Namespace = "longhorn-system"
	installer.CreateNamespace = true
	installer.ReleaseName = "longhorn"
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 20 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func (b Bootstrapper) installSoftServe(adminPublicKey string, namespace string, repoIP netip.Addr) error {
	fmt.Println("Installing SoftServe")
	keys, err := NewSSHKeyPair("soft-serve")
	if err != nil {
		return err
	}
	config, err := b.ha.New(namespace)
	if err != nil {
		return err
	}
	chart, err := b.cl.Load("soft-serve")
	if err != nil {
		return err
	}
	values := map[string]any{
		"image": map[string]any{
			"repository": "charmcli/soft-serve",
			"tag":        "v0.7.1",
			"pullPolicy": "IfNotPresent",
		},
		"privateKey":  string(keys.RawPrivateKey()),
		"publicKey":   string(keys.RawAuthorizedKey()),
		"adminKey":    adminPublicKey,
		"reservedIP":  repoIP.String(),
		"serviceType": "LoadBalancer",
	}
	installer := action.NewInstall(config)
	installer.Namespace = namespace
	installer.CreateNamespace = true
	installer.ReleaseName = "soft-serve"
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 20 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func (b Bootstrapper) installFluxcd(ss *soft.Client, envName string) error {
	keys, err := NewSSHKeyPair("fluxcd")
	if err != nil {
		return err
	}
	if err := ss.AddUser("flux", keys.AuthorizedKey()); err != nil {
		return err
	}
	if err := ss.MakeUserAdmin("flux"); err != nil {
		return err
	}
	if err := ss.AddRepository("config"); err != nil {
		return err
	}
	repo, err := ss.GetRepo("config")
	if err != nil {
		return err
	}
	repoIO, err := NewRepoIO(repo, ss.Signer)
	if err != nil {
		return err
	}
	if err := repoIO.Atomic(func(r RepoFS) (string, error) {
		w, err := r.Writer("README.md")
		if err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(w, "# %s systems", envName); err != nil {
			return "", err
		}
		return "readme", nil
	}); err != nil {
		return err
	}
	fmt.Println("Installing Flux")
	ssPublicKeys, err := ss.GetPublicKeys()
	if err != nil {
		return err
	}
	host := strings.Split(ss.Addr, ":")[0]
	if err := b.installFluxBootstrap(
		ss.GetRepoAddress("config"),
		host,
		ssPublicKeys,
		string(keys.RawPrivateKey()),
		envName,
	); err != nil {
		return err
	}
	return nil
}

func (b Bootstrapper) installFluxBootstrap(repoAddr, repoHost string, repoHostPubKeys []string, privateKey, envName string) error {
	config, err := b.ha.New(envName)
	if err != nil {
		return err
	}
	chart, err := b.cl.Load("flux-bootstrap")
	if err != nil {
		return err
	}
	var lines []string
	for _, k := range repoHostPubKeys {
		lines = append(lines, fmt.Sprintf("%s %s", repoHost, k))
	}
	values := map[string]any{
		"image": map[string]any{
			"repository": "fluxcd/flux-cli",
			"tag":        "v2.1.2",
			"pullPolicy": "IfNotPresent",
		},
		"repositoryAddress":        repoAddr,
		"repositoryHost":           repoHost,
		"repositoryHostPublicKeys": strings.Join(lines, "\n"),
		"privateKey":               privateKey,
		"installationNamespace":    fmt.Sprintf("%s-flux", envName),
	}
	installer := action.NewInstall(config)
	installer.Namespace = envName
	installer.CreateNamespace = true
	installer.ReleaseName = "flux"
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 20 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func (b Bootstrapper) installInfrastructureServices(repo RepoIO, env EnvConfig) error {
	install := func(name string) error {
		fmt.Printf("Installing infrastructure service %s\n", name)
		app, err := b.appRepo.Find(name)
		if err != nil {
			return err
		}
		namespace := fmt.Sprintf("%s-%s", env.Name, app.Namespace())
		derived := Derived{
			Global: Values{
				PCloudEnvName: env.Name,
			},
		}
		return InstallApp(repo, b.ns, app, filepath.Join("/infrastructure", app.Name()), namespace, nil, derived)
	}
	appsToInstall := []string{
		"resource-renderer-controller",
		"headscale-controller",
		"csi-driver-smb",
		"ingress-public",
		"cert-manager",
	}
	for _, name := range appsToInstall {
		if err := install(name); err != nil {
			return err
		}
	}
	return nil
}

func configureMainRepo(repo RepoIO, env EnvConfig) error {
	return repo.Atomic(func(r RepoFS) (string, error) {
		if err := WriteYaml(r, "config.yaml", env); err != nil {
			return "", err
		}
		if err := WriteYaml(r, "env-cidrs.yaml", EnvCIDRs{}); err != nil {
			return "", err
		}
		kust := NewKustomization()
		kust.AddResources(
			fmt.Sprintf("%s-flux", env.Name),
			"infrastructure",
			"environments",
		)
		if err := WriteYaml(r, "kustomization.yaml", kust); err != nil {
			return "", err
		}
		{
			out, err := r.Writer("infrastructure/pcloud-charts.yaml")
			if err != nil {
				return "", err
			}
			defer out.Close()
			_, err = out.Write([]byte(fmt.Sprintf(`
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: pcloud # TODO(giolekva): use more generic name
  namespace: %s
spec:
  interval: 1m0s
  url: https://github.com/giolekva/pcloud
  ref:
    branch: main
`, env.Name)))
			if err != nil {
				return "", err
			}
		}
		infraKust := NewKustomization()
		infraKust.AddResources("pcloud-charts.yaml")
		if err := WriteYaml(r, "infrastructure/kustomization.yaml", infraKust); err != nil {
			return "", err
		}
		if err := WriteYaml(r, "environments/kustomization.yaml", NewKustomization()); err != nil {
			return "", err
		}
		return "initialize pcloud directory structure", nil
	})
}

func (b Bootstrapper) installEnvManager(repo RepoIO, ss *soft.Client, env EnvConfig) error {
	keys, err := NewSSHKeyPair("env-manager")
	if err != nil {
		return err
	}
	user := fmt.Sprintf("%s-env-manager", env.Name)
	if err := ss.AddUser(user, keys.AuthorizedKey()); err != nil {
		return err
	}
	if err := ss.MakeUserAdmin(user); err != nil {
		return err
	}
	app, err := b.appRepo.Find("env-manager")
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s-%s", env.Name, app.Namespace())
	derived := Derived{
		Global: Values{
			PCloudEnvName: env.Name,
		},
		Values: map[string]any{
			"repoIP":        env.ServiceIPs.ConfigRepo,
			"repoPort":      22,
			"repoName":      "config",
			"sshPrivateKey": string(keys.RawPrivateKey()),
		},
	}
	return InstallApp(repo, b.ns, app, filepath.Join("/infrastructure", app.Name()), namespace, derived.Values, derived)
}

func (b Bootstrapper) installOryHydraMaester(repo RepoIO, env EnvConfig) error {
	app, err := b.appRepo.Find("hydra-maester")
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s-%s", env.Name, app.Namespace())
	derived := Derived{
		Global: Values{
			PCloudEnvName: env.Name,
		},
	}
	return InstallApp(repo, b.ns, app, filepath.Join("/infrastructure", app.Name()), namespace, nil, derived)
}

func (b Bootstrapper) installDNSZoneManager(repo RepoIO, env EnvConfig) error {
	const (
		volumeClaimName = "dns-zone-configs"
		volumeMountPath = "/etc/pcloud/dns-zone-configs"
	)
	app, err := b.appRepo.Find("dns-zone-manager")
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s-%s", env.Name, app.Namespace())
	derived := Derived{
		Global: Values{
			PCloudEnvName: env.Name,
		},
		Values: map[string]any{
			"volume": map[string]any{
				"claimName": volumeClaimName,
				"mountPath": volumeMountPath,
				"size":      "1Gi",
			},
			"apiConfigMapName": dnsAPIConfigMapName,
		},
	}
	return InstallApp(repo, b.ns, app, filepath.Join("/infrastructure", app.Name()), namespace, derived.Values, derived)
}

func (b Bootstrapper) installFluxcdReconciler(repo RepoIO, ss *soft.Client, env EnvConfig) error {
	app, err := b.appRepo.Find("fluxcd-reconciler")
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s-%s", env.Name, app.Namespace())
	derived := Derived{
		Global: Values{
			PCloudEnvName: env.Name,
		},
	}
	return InstallApp(repo, b.ns, app, filepath.Join("/infrastructure", app.Name()), namespace, nil, derived)
}

type HelmActionConfigFactory interface {
	New(namespace string) (*action.Configuration, error)
}

type ChartLoader interface {
	Load(name string) (*chart.Chart, error)
}

type fsChartLoader struct {
	baseDir string
}

func NewFSChartLoader(baseDir string) ChartLoader {
	return &fsChartLoader{baseDir}
}

func (l *fsChartLoader) Load(name string) (*chart.Chart, error) {
	return loader.Load(filepath.Join(l.baseDir, name))
}
