package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/kube"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

var bootstrapFlags struct {
	pcloudEnvName             string
	chartsDir                 string
	adminPubKey               string
	storageDir                string
	volumeDefaultReplicaCount int
	softServeIP               string // TODO(giolekva): reserve using metallb IPAddressPool
}

func bootstrapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "bootstrap",
		RunE: bootstrapCmdRun,
	}
	cmd.Flags().StringVar(
		&bootstrapFlags.pcloudEnvName,
		"pcloud-env-name",
		"pcloud",
		"",
	)
	cmd.Flags().StringVar(
		&bootstrapFlags.chartsDir,
		"charts-dir",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&bootstrapFlags.adminPubKey,
		"admin-pub-key",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&bootstrapFlags.storageDir,
		"storage-dir",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&bootstrapFlags.volumeDefaultReplicaCount,
		"volume-default-replica-count",
		3,
		"",
	)
	cmd.Flags().StringVar(
		&bootstrapFlags.softServeIP,
		"soft-serve-ip",
		"",
		"",
	)
	return cmd
}

func bootstrapCmdRun(cmd *cobra.Command, args []string) error {
	adminPubKey, err := os.ReadFile(bootstrapFlags.adminPubKey)
	if err != nil {
		return err
	}
	bootstrapJobKeys, err := installer.NewSSHKeyPair()
	if err != nil {
		return err
	}
	if err := installMetallb(); err != nil {
		return err
	}
	if err := installLonghorn(); err != nil {
		return err
	}
	time.Sleep(2 * time.Minute) // TODO(giolekva): implement proper wait
	if err := installSoftServe(bootstrapJobKeys.Public); err != nil {
		return err
	}
	time.Sleep(1 * time.Minute) // TODO(giolekva): implement proper wait
	ss, err := soft.NewClient(bootstrapFlags.softServeIP, 22, []byte(bootstrapJobKeys.Private), log.Default())
	if err != nil {
		return err
	}
	if ss.AddPublicKey("admin", string(adminPubKey)); err != nil {
		return err
	}
	if err := installFluxcd(ss, bootstrapFlags.pcloudEnvName); err != nil {
		return err
	}
	repo, err := ss.GetRepo(bootstrapFlags.pcloudEnvName)
	if err != nil {
		return err
	}
	repoIO := installer.NewRepoIO(repo, ss.Signer)
	if err := configurePCloudRepo(repoIO); err != nil {
		return err
	}
	// TODO(giolekva): commit this to the repo above
	global := installer.Values{
		PCloudEnvName: bootstrapFlags.pcloudEnvName,
	}
	nsCreator, err := newNSCreator()
	if err != nil {
		return err
	}
	nsGen := installer.NewPrefixGenerator("pcloud-")
	if err := installInfrastructureServices(repoIO, nsGen, nsCreator, global); err != nil {
		return err
	}
	if err := installEnvManager(ss, repoIO, nsGen, nsCreator, global); err != nil {
		return err
	}
	if ss.RemovePublicKey("admin", bootstrapJobKeys.Public); err != nil {
		return err
	}
	return nil
}

func installMetallb() error {
	if err := installMetallbNamespace(); err != nil {
		return err
	}
	if err := installMetallbService(); err != nil {
		return err
	}
	if err := installMetallbConfig(); err != nil {
		return err
	}
	return nil
}

func installMetallbNamespace() error {
	fmt.Println("Installing metallb namespace")
	// config, err := createActionConfig("default")
	config, err := createActionConfig(bootstrapFlags.pcloudEnvName)
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "namespace"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{
		// "namespace": "pcloud-metallb",
		"namespace": "metallb-system",
		"labels": []string{
			"pod-security.kubernetes.io/audit: privileged",
			"pod-security.kubernetes.io/enforce: privileged",
			"pod-security.kubernetes.io/warn: privileged",
		},
	}
	installer := action.NewInstall(config)
	installer.Namespace = bootstrapFlags.pcloudEnvName
	installer.ReleaseName = "metallb-ns"
	installer.Wait = true
	installer.WaitForJobs = true
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func installMetallbService() error {
	fmt.Println("Installing metallb")
	// config, err := createActionConfig("default")
	config, err := createActionConfig("metallb-system")
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "metallb"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{ // TODO(giolekva): add loadBalancerClass?
		"controller": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "quay.io/metallb/controller",
				"tag":        "v0.13.9",
				"pullPolicy": "IfNotPresent",
			},
			"logLevel": "info",
		},
		"speaker": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "quay.io/metallb/speaker",
				"tag":        "v0.13.9",
				"pullPolicy": "IfNotPresent",
			},
			"logLevel": "info",
		},
	}
	installer := action.NewInstall(config)
	installer.Namespace = "metallb-system" // "pcloud-metallb"
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

func installMetallbConfig() error {
	fmt.Println("Installing metallb-config")
	// config, err := createActionConfig("default")
	config, err := createActionConfig("metallb-system")
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "metallb-config"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{
		"from": "192.168.0.210",
		"to":   "192.168.0.240",
	}
	installer := action.NewInstall(config)
	installer.Namespace = "metallb-system" // "pcloud-metallb"
	installer.CreateNamespace = true
	installer.ReleaseName = "metallb-cfg"
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 20 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func installLonghorn() error {
	fmt.Println("Installing Longhorn")
	config, err := createActionConfig(bootstrapFlags.pcloudEnvName)
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "longhorn"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{
		"defaultSettings": map[string]interface{}{
			"defaultDataPath": bootstrapFlags.storageDir,
		},
		"persistence": map[string]interface{}{
			"defaultClassReplicaCount": bootstrapFlags.volumeDefaultReplicaCount,
		},
		"service": map[string]interface{}{
			"ui": map[string]interface{}{
				"type": "LoadBalancer",
			},
		},
		"ingress": map[string]interface{}{
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

func installSoftServe(adminPublicKey string) error {
	fmt.Println("Installing SoftServe")
	keys, err := installer.NewSSHKeyPair()
	if err != nil {
		return err
	}
	config, err := createActionConfig(bootstrapFlags.pcloudEnvName)
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "soft-serve"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{
		"privateKey": keys.Private,
		"publicKey":  keys.Public,
		"adminKey":   adminPublicKey,
		"reservedIP": bootstrapFlags.softServeIP,
	}
	installer := action.NewInstall(config)
	installer.Namespace = bootstrapFlags.pcloudEnvName
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

func installFluxcd(ss *soft.Client, pcloudEnvName string) error {
	keys, err := installer.NewSSHKeyPair()
	if err != nil {
		return err
	}
	if err := ss.AddUser("flux", keys.Public); err != nil {
		return err
	}
	if err := ss.MakeUserAdmin("flux"); err != nil {
		return err
	}
	fmt.Printf("Creating /%s repo", pcloudEnvName)
	if err := ss.AddRepository(pcloudEnvName, "# PCloud Systems"); err != nil {
		return err
	}
	fmt.Println("Installing Flux")
	ssPublic, err := ss.GetPublicKey()
	if err != nil {
		return err
	}
	if err := installFluxBootstrap(
		ss.GetRepoAddress(pcloudEnvName),
		ss.IP,
		string(ssPublic),
		keys.Private,
	); err != nil {
		return err
	}
	return nil
}

func installFluxBootstrap(repoAddr, repoHost, repoHostPubKey, privateKey string) error {
	config, err := createActionConfig(bootstrapFlags.pcloudEnvName)
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "flux-bootstrap"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{
		"repositoryAddress":       repoAddr,
		"repositoryHost":          repoHost,
		"repositoryHostPublicKey": repoHostPubKey,
		"privateKey":              privateKey,
	}
	installer := action.NewInstall(config)
	installer.Namespace = bootstrapFlags.pcloudEnvName
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

func installInfrastructureServices(repo installer.RepoIO, nsGen installer.NamespaceGenerator, nsCreator installer.NamespaceCreator, global installer.Values) error {
	appRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	install := func(name string) error {
		app, err := appRepo.Find(name)
		if err != nil {
			return err
		}
		namespaces := make([]string, len(app.Namespaces))
		for i, n := range app.Namespaces {
			namespaces[i], err = nsGen.Generate(n)
			if err != nil {
				return err
			}
		}
		for _, n := range namespaces {
			if err := nsCreator.Create(n); err != nil {
				return err
			}
		}
		derived := installer.Derived{
			Global: global,
		}
		if len(namespaces) > 0 {
			derived.Release.Namespace = namespaces[0]
		}
		return repo.InstallApp(*app, filepath.Join("/infrastructure", app.Name), map[string]any{}, derived)
	}
	appsToInstall := []string{
		"resource-renderer-controller",
		"headscale-controller",
		"csi-driver-smb",
		"ingress-public",
		"cert-manager",
		"cert-manager-webhook-gandi",
		"cert-manager-webhook-gandi-role",
	}
	for _, name := range appsToInstall {
		if err := install(name); err != nil {
			return err
		}
	}
	return nil
}

func configurePCloudRepo(repo installer.RepoIO) error {
	{
		kust := installer.NewKustomization()
		kust.AddResources("pcloud-flux", "infrastructure", "environments")
		if err := repo.WriteKustomization("kustomization.yaml", kust); err != nil {
			return err
		}
		{
			out, err := repo.Writer("infrastructure/pcloud-charts.yaml")
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = out.Write([]byte(`
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: GitRepository
metadata:
  name: pcloud # TODO(giolekva): use more generic name
  namespace: pcloud # TODO(giolekva): configurable
spec:
  interval: 1m0s
  url: https://github.com/giolekva/pcloud
  ref:
    branch: main
`))
			if err != nil {
				return err
			}
		}
		infraKust := installer.NewKustomization()
		infraKust.AddResources("pcloud-charts.yaml")
		if err := repo.WriteKustomization("infrastructure/kustomization.yaml", infraKust); err != nil {
			return err
		}
		if err := repo.WriteKustomization("environments/kustomization.yaml", installer.NewKustomization()); err != nil {
			return err
		}
		if err := repo.CommitAndPush("initialize pcloud directory structure"); err != nil {
			return err
		}
	}
	return nil
}

func installEnvManager(ss *soft.Client, repo installer.RepoIO, nsGen installer.NamespaceGenerator, nsCreator installer.NamespaceCreator, global installer.Values) error {
	keys, err := installer.NewSSHKeyPair()
	if err != nil {
		return err
	}
	user := fmt.Sprintf("%s-env-manager", bootstrapFlags.pcloudEnvName)
	if err := ss.AddUser(user, keys.Public); err != nil {
		return err
	}
	if err := ss.MakeUserAdmin(user); err != nil {
		return err
	}
	appRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	app, err := appRepo.Find("env-manager")
	if err != nil {
		return err
	}
	namespaces := make([]string, len(app.Namespaces))
	for i, n := range app.Namespaces {
		namespaces[i], err = nsGen.Generate(n)
		if err != nil {
			return err
		}
	}
	for _, n := range namespaces {
		if err := nsCreator.Create(n); err != nil {
			return err
		}
	}
	derived := installer.Derived{
		Global: global,
		Values: map[string]any{
			"RepoIP":        bootstrapFlags.softServeIP,
			"SSHPrivateKey": keys.Private,
		},
	}
	if len(namespaces) > 0 {
		derived.Release.Namespace = namespaces[0]
	}
	return repo.InstallApp(*app, filepath.Join("/infrastructure", app.Name), derived.Values, derived)
}

func createActionConfig(namespace string) (*action.Configuration, error) {
	config := new(action.Configuration)
	if err := config.Init(
		kube.GetConfig(rootFlags.kubeConfig, "", namespace),
		namespace,
		"",
		func(fmtString string, args ...interface{}) {
			fmt.Printf(fmtString, args...)
			fmt.Println()
		},
	); err != nil {
		return nil, err
	}
	return config, nil
}
