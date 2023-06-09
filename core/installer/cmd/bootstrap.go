// TODO
// * ns pcloud not found

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
	chartsDir                 string
	adminPubKey               string
	adminPrivKey              string
	storageDir                string
	volumeDefaultReplicaCount int
	softServeIP               string
}

func bootstrapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "bootstrap",
		RunE: bootstrapCmdRun,
	}
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
		&bootstrapFlags.adminPrivKey,
		"admin-priv-key",
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
	adminPubKey, adminPrivKey, err := readAdminKeys()
	if err != nil {
		return err
	}
	softServePub, softServePriv, err := installer.GenerateSSHKeys()
	if err != nil {
		return err
	}
	if err := installMetallbNamespace(); err != nil {
		return err
	}
	if err := installMetallb(); err != nil {
		return err
	}
	time.Sleep(1 * time.Minute)
	if err := installMetallbConfig(); err != nil {
		return err
	}
	if err := installLonghorn(); err != nil {
		return err
	}
	time.Sleep(2 * time.Minute)
	if err := installSoftServe(softServePub, softServePriv, string(adminPubKey)); err != nil {
		return err
	}
	time.Sleep(2 * time.Minute)
	ss, err := soft.NewClient(bootstrapFlags.softServeIP, 22, adminPrivKey, log.Default())
	if err != nil {
		return err
	}
	fluxPub, fluxPriv, err := installer.GenerateSSHKeys()
	if err != nil {
		return err
	}
	if err := ss.AddUser("flux", fluxPub); err != nil {
		return err
	}
	if err := ss.MakeUserAdmin("flux"); err != nil {
		return err
	}
	fmt.Println("Creating /pcloud repo")
	if err := ss.AddRepository("pcloud", "# PCloud Systems"); err != nil {
		return err
	}
	fmt.Println("Installing Flux")
	if err := installFlux("ssh://soft-serve.pcloud.svc.cluster.local:22/pcloud", "soft-serve.pcloud.svc.cluster.local", softServePub, fluxPriv); err != nil {
		return err
	}
	pcloudRepo, err := ss.GetRepo("pcloud") // TODO(giolekva): configurable
	if err != nil {
		return err
	}
	if err := configurePCloudRepo(installer.NewRepoIO(pcloudRepo, ss.Signer)); err != nil {
		return err
	}
	// TODO(giolekva): everything below must be installed using Flux
	if err := installIngressPublic(); err != nil {
		return err
	}
	if err := installCertManager(); err != nil {
		return err
	}
	if err := installCertManagerWebhookGandi(); err != nil {
		return err
	}
	// TODO(giolekva): ideally should be installed automatically if any of the user installed apps requires it
	if err := installSmbDriver(); err != nil {
		return err
	}
	return nil
}

func installMetallbNamespace() error {
	fmt.Println("Installing metallb namespace")
	// config, err := createActionConfig("default")
	config, err := createActionConfig("pcloud")
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
	installer.Namespace = "pcloud"
	installer.ReleaseName = "metallb-ns"
	installer.Wait = true
	installer.WaitForJobs = true
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func installMetallb() error {
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
	config, err := createActionConfig("pcloud")
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

func installSoftServe(pubKey, privKey, adminKey string) error {
	fmt.Println("Installing SoftServe")
	config, err := createActionConfig("pcloud")
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "soft-serve"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{
		"privateKey": privKey,
		"publicKey":  pubKey,
		"adminKey":   adminKey,
		"reservedIP": bootstrapFlags.softServeIP,
	}
	installer := action.NewInstall(config)
	installer.Namespace = "pcloud"
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

func installFlux(repoAddr, repoHost, repoHostPubKey, privateKey string) error {
	config, err := createActionConfig("pcloud")
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
	installer.Namespace = "pcloud"
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

func installIngressPublic() error {
	config, err := createActionConfig("pcloud")
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "ingress-nginx"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{
		"fullnameOverride": "pcloud-ingress-public",
		"controller": map[string]interface{}{
			"service": map[string]interface{}{
				"type": "LoadBalancer",
			},
			"ingressClassByName": true,
			"ingressClassResource": map[string]interface{}{
				"name":            "pcloud-ingress-public",
				"enabled":         true,
				"default":         false,
				"controllerValue": "k8s.io/pcloud-ingress-public",
			},
			"config": map[string]interface{}{
				"proxy-body-size": "100M",
			},
		},
		"udp": map[string]interface{}{
			"6881": "lekva-app-qbittorrent/torrent:6881",
		},
		"tcp": map[string]interface{}{
			"6881": "lekva-app-qbittorrent/torrent:6881",
		},
	}
	installer := action.NewInstall(config)
	installer.Namespace = "pcloud-ingress-public"
	installer.CreateNamespace = true
	installer.ReleaseName = "ingress-public"
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 20 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func installCertManager() error {
	config, err := createActionConfig("pcloud-cert-manager")
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "cert-manager"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{
		"fullnameOverride": "pcloud-cert-manager",
		"installCRDs":      true,
		"image": map[string]interface{}{
			"tag":        "v1.11.1",
			"pullPolicy": "IfNotPresent",
		},
	}
	installer := action.NewInstall(config)
	installer.Namespace = "pcloud-cert-manager"
	installer.CreateNamespace = true
	installer.ReleaseName = "cert-manager"
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 20 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func installCertManagerWebhookGandi() error {
	config, err := createActionConfig("pcloud-cert-manager")
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "cert-manager-webhook-gandi"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{
		"fullnameOverride": "pcloud-cert-manager-webhook-gandi",
		"certManager": map[string]interface{}{
			"namespace":          "pcloud-cert-manager",
			"serviceAccountName": "pcloud-cert-manager",
		},
		"image": map[string]interface{}{
			"repository": "giolekva/cert-manager-webhook-gandi",
			"tag":        "v0.2.0",
			"pullPolicy": "IfNotPresent",
		},
		"logLevel": 2,
	}
	installer := action.NewInstall(config)
	installer.Namespace = "pcloud-cert-manager"
	installer.CreateNamespace = false
	installer.ReleaseName = "cert-manager-webhook-gandi"
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 20 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func installSmbDriver() error {
	config, err := createActionConfig("pcloud-csi-driver-smb")
	if err != nil {
		return err
	}
	chart, err := loader.Load(filepath.Join(bootstrapFlags.chartsDir, "csi-driver-smb"))
	if err != nil {
		return err
	}
	values := map[string]interface{}{}
	installer := action.NewInstall(config)
	installer.Namespace = "pcloud-csi-driver-smb"
	installer.CreateNamespace = true
	installer.ReleaseName = "csi-driver-smb"
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 20 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func configurePCloudRepo(repo installer.RepoIO) error {
	kust := installer.NewKustomization()
	kust.AddResources("pcloud-flux", "environments")
	if err := repo.WriteKustomization("kustomization.yaml", kust); err != nil {
		return err
	}
	if err := repo.WriteKustomization("environments/kustomization.yaml", installer.NewKustomization()); err != nil {
		return err
	}
	return repo.CommitAndPush("initialize pcloud directory structure, environments with kustomization.yaml-s")
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

func readAdminKeys() ([]byte, []byte, error) {
	pubKey, err := os.ReadFile(bootstrapFlags.adminPubKey)
	if err != nil {
		return nil, nil, err
	}
	privKey, err := os.ReadFile(bootstrapFlags.adminPrivKey)
	if err != nil {
		return nil, nil, err
	}
	return pubKey, privKey, nil
}
