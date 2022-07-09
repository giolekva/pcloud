package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/kube"
)

var bootstrapFlags struct {
	chartsDir    string
	adminPubKey  string
	adminPrivKey string
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
	return cmd
}

func bootstrapCmdRun(cmd *cobra.Command, args []string) error {
	adminPubKey, adminPrivKey, err := readAdminKeys()
	if err != nil {
		return err
	}
	fluxPub, fluxPriv, err := generateSSHKeys()
	if err != nil {
		return err
	}
	softServePub, softServePriv, err := generateSSHKeys()
	if err != nil {
		return err
	}
	fmt.Println("Installing SoftServe")
	if err := installSoftServe(softServePub, softServePriv, string(adminPubKey)); err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	ss, err := soft.NewClient("192.168.0.208", 22, adminPrivKey, log.Default())
	if err != nil {
		return err
	}
	if err := ss.UpdateConfig(
		soft.DefaultConfig([]string{string(adminPubKey), fluxPub}),
		"set admin keys"); err != nil {
		return err
	}
	if err := ss.ReloadConfig(); err != nil {
		return err
	}
	fmt.Println("Creating /pcloud repo")
	if err := ss.AddRepository("pcloud", "# PCloud Systems\n"); err != nil {
		return err
	}
	fmt.Println("Installing Flux")
	if err := installFlux("ssh://soft-serve.pcloud.svc.cluster.local:22/pcloud", "soft-serve.pcloud.svc.cluster.local", softServePub, fluxPriv); err != nil {
		return err
	}
	return nil
}

func installSoftServe(pubKey, privKey, adminKey string) error {
	config, err := createActionConfig()
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
	}
	installer := action.NewInstall(config)
	installer.Namespace = "pcloud"
	installer.CreateNamespace = true
	installer.ReleaseName = "soft-serve"
	installer.Wait = true
	installer.Timeout = 5 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func installFlux(repoAddr, repoHost, repoHostPubKey, privateKey string) error {
	config, err := createActionConfig()
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
	installer.Timeout = 5 * time.Minute
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func createActionConfig() (*action.Configuration, error) {
	config := new(action.Configuration)
	if err := config.Init(
		kube.GetConfig(rootFlags.kubeConfig, "", ""),
		"pcloud",
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

func generateSSHKeys() (string, string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	privEnc, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", err
	}
	privPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privEnc,
		},
	)
	pubKey, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", "", err
	}
	return string(ssh.MarshalAuthorizedKey(pubKey)), string(privPem), nil
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
