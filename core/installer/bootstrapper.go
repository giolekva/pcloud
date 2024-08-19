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

	"github.com/giolekva/pcloud/core/installer/io"
	"github.com/giolekva/pcloud/core/installer/soft"
)

const IPAddressPoolLocal = "local"
const IPAddressPoolConfigRepo = "config-repo"
const IPAddressPoolIngressPublic = "ingress-public"

const dnsAPIConfigMapName = "api-config"

type Bootstrapper struct {
	cl         ChartLoader
	ns         NamespaceCreator
	ha         HelmActionConfigFactory
	appRepo    AppRepository
	repoClient soft.ClientGetter
}

func NewBootstrapper(cl ChartLoader, ns NamespaceCreator, ha HelmActionConfigFactory, appRepo AppRepository, repoClient soft.ClientGetter) Bootstrapper {
	return Bootstrapper{cl, ns, ha, appRepo, repoClient}
}

func (b Bootstrapper) findApp(name string) (InfraApp, error) {
	app, err := b.appRepo.Find(name)
	if err != nil {
		return nil, err
	}
	if a, ok := app.(InfraApp); ok {
		return a, nil
	} else {
		return nil, fmt.Errorf("not found")
	}
}

func (b Bootstrapper) Run(env BootstrapConfig) error {
	if err := b.ns.Create(env.InfraName); err != nil {
		return err
	}
	if err := b.installMetallb(env); err != nil {
		return err
	}
	if err := b.installLonghorn(env.InfraName, env.StorageDir, env.VolumeDefaultReplicaCount); err != nil {
		return err
	}
	bootstrapJobKeys, err := NewSSHKeyPair("bootstrapper")
	if err != nil {
		return err
	}
	if err := b.installSoftServe(bootstrapJobKeys.AuthorizedKey(), env.InfraName, env.ServiceIPs.ConfigRepo); err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	ss, err := b.repoClient.Get(
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
	if err := b.installFluxcd(ss, env.InfraName); err != nil {
		return err
	}
	fmt.Println("Fluxcd installed")
	repoIO, err := ss.GetRepo("config")
	if err != nil {
		fmt.Println("Failed to get config repo")
		return err
	}
	hf := NewGitHelmFetcher()
	lg := NewInfraLocalChartGenerator()
	mgr, err := NewInfraAppManager(repoIO, b.ns, hf, lg)
	if err != nil {
		return err
	}
	fmt.Println("Configuring main repo")
	if err := configureMainRepo(repoIO, env); err != nil {
		return err
	}
	fmt.Println("Installing infrastructure services")
	if err := b.installInfrastructureServices(mgr, env); err != nil {
		return err
	}
	fmt.Println("Installing public ingress")
	if err := b.installIngressPublic(mgr, ss, env); err != nil {
		return err
	}
	fmt.Println("Installing DNS Zone Manager")
	if err := b.installDNSZoneManager(mgr, env); err != nil {
		return err
	}
	fmt.Println("Installing Fluxcd Reconciler")
	if err := b.installFluxcdReconciler(mgr, ss, env); err != nil {
		return err
	}
	fmt.Println("Installing env manager")
	if err := b.installEnvManager(mgr, ss, env); err != nil {
		return err
	}
	fmt.Println("Installing Ory Hydra Maester")
	if err := b.installOryHydraMaester(mgr, env); err != nil {
		return err
	}
	fmt.Println("Environment ready to use")
	return nil
}

func (b Bootstrapper) installMetallb(env BootstrapConfig) error {
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

func (b Bootstrapper) installMetallbNamespace(env BootstrapConfig) error {
	fmt.Println("Installing metallb namespace")
	config, err := b.ha.New(env.InfraName)
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
	installer.Namespace = env.InfraName
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

func (b Bootstrapper) installFluxcd(ss soft.Client, envName string) error {
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
	repoIO, err := ss.GetRepo("config")
	if err != nil {
		return err
	}
	if _, err := repoIO.Do(func(r soft.RepoFS) (string, error) {
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
	host := strings.Split(ss.Address(), ":")[0]
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

func (b Bootstrapper) installInfrastructureServices(mgr *InfraAppManager, env BootstrapConfig) error {
	install := func(name string) error {
		fmt.Printf("Installing infrastructure service %s\n", name)
		app, err := b.findApp(name)
		if err != nil {
			return err
		}
		namespace := fmt.Sprintf("%s-%s", env.InfraName, app.Namespace())
		appDir := filepath.Join("/infrastructure", app.Slug())
		_, err = mgr.Install(app, appDir, namespace, map[string]any{})
		return err
	}
	appsToInstall := []string{
		"resource-renderer-controller",
		"headscale-controller",
		"csi-driver-smb",
		"cert-manager",
	}
	for _, name := range appsToInstall {
		if err := install(name); err != nil {
			return err
		}
	}
	return nil
}

func configureMainRepo(repo soft.RepoIO, bootstrap BootstrapConfig) error {
	_, err := repo.Do(func(r soft.RepoFS) (string, error) {
		if err := soft.WriteYaml(r, "bootstrap-config.yaml", bootstrap); err != nil {
			return "", err
		}
		infra := InfraConfig{
			Name:                 bootstrap.InfraName,
			PublicIP:             bootstrap.PublicIP,
			InfraNamespacePrefix: bootstrap.NamespacePrefix,
			InfraAdminPublicKey:  bootstrap.AdminPublicKey,
		}
		if err := soft.WriteYaml(r, "config.yaml", infra); err != nil {
			return "", err
		}
		if err := soft.WriteYaml(r, "env-cidrs.yaml", EnvCIDRs{}); err != nil {
			return "", err
		}
		kust := io.NewKustomization()
		kust.AddResources(
			fmt.Sprintf("%s-flux", bootstrap.InfraName),
			"infrastructure",
			"environments",
		)
		if err := soft.WriteYaml(r, "kustomization.yaml", kust); err != nil {
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
`, bootstrap.InfraName)))
			if err != nil {
				return "", err
			}
		}
		infraKust := io.NewKustomization()
		infraKust.AddResources("pcloud-charts.yaml")
		if err := soft.WriteYaml(r, "infrastructure/kustomization.yaml", infraKust); err != nil {
			return "", err
		}
		if err := soft.WriteYaml(r, "environments/kustomization.yaml", io.NewKustomization()); err != nil {
			return "", err
		}
		return "initialize pcloud directory structure", nil
	})
	return err
}

func (b Bootstrapper) installEnvManager(mgr *InfraAppManager, ss soft.Client, env BootstrapConfig) error {
	keys, err := NewSSHKeyPair("env-manager")
	if err != nil {
		return err
	}
	user := fmt.Sprintf("%s-env-manager", env.InfraName)
	if err := ss.AddUser(user, keys.AuthorizedKey()); err != nil {
		return err
	}
	if err := ss.MakeUserAdmin(user); err != nil {
		return err
	}
	app, err := b.findApp("env-manager")
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s-%s", env.InfraName, app.Namespace())
	appDir := filepath.Join("/infrastructure", app.Slug())
	_, err = mgr.Install(app, appDir, namespace, map[string]any{
		"repoIP":        env.ServiceIPs.ConfigRepo,
		"repoPort":      22,
		"repoName":      "config",
		"sshPrivateKey": string(keys.RawPrivateKey()),
	})
	return err
}

func (b Bootstrapper) installIngressPublic(mgr *InfraAppManager, ss soft.Client, env BootstrapConfig) error {
	keys, err := NewSSHKeyPair("port-allocator")
	if err != nil {
		return err
	}
	user := fmt.Sprintf("%s-port-allocator", env.InfraName)
	if err := ss.AddUser(user, keys.AuthorizedKey()); err != nil {
		return err
	}
	if err := ss.AddReadWriteCollaborator("config", user); err != nil {
		return err
	}
	app, err := b.findApp("ingress-public")
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s-%s", env.InfraName, app.Namespace())
	appDir := filepath.Join("/infrastructure", app.Slug())
	_, err = mgr.Install(app, appDir, namespace, map[string]any{
		"sshPrivateKey": string(keys.RawPrivateKey()),
	})
	return err
}

func (b Bootstrapper) installOryHydraMaester(mgr *InfraAppManager, env BootstrapConfig) error {
	app, err := b.findApp("hydra-maester")
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s-%s", env.InfraName, app.Namespace())
	appDir := filepath.Join("/infrastructure", app.Slug())
	_, err = mgr.Install(app, appDir, namespace, map[string]any{})
	return err
}

func (b Bootstrapper) installDNSZoneManager(mgr *InfraAppManager, env BootstrapConfig) error {
	app, err := b.findApp("dns-gateway")
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s-%s", env.InfraName, app.Namespace())
	appDir := filepath.Join("/infrastructure", app.Slug())
	_, err = mgr.Install(app, appDir, namespace, map[string]any{
		"servers": []EnvDNS{},
	})
	return err
}

func (b Bootstrapper) installFluxcdReconciler(mgr *InfraAppManager, ss soft.Client, env BootstrapConfig) error {
	app, err := b.findApp("fluxcd-reconciler")
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s-%s", env.InfraName, app.Namespace())
	appDir := filepath.Join("/infrastructure", app.Slug())
	_, err = mgr.Install(app, appDir, namespace, map[string]any{})
	return err
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
