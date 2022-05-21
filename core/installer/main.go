package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"embed"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/kube"
	"sigs.k8s.io/yaml"
)

//go:embed values-tmpl
var valuesTmpls embed.FS

//go:embed config.yaml
var configTmpl string

var rootCmd *cobra.Command

var rootFlags struct {
	kubeConfig string
}

var installFlags struct {
	config    string
	appName   string
	outputDir string
}

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
	// installer.DryRun = true
	// installer.OutputDir = "/tmp/rr"
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
	installer.ReleaseName = "flux4"
	installer.Wait = true
	installer.WaitForJobs = true
	installer.Timeout = 5 * time.Minute
	// installer.DryRun = true
	// installer.OutputDir = "/tmp/ee"
	if _, err := installer.RunWithContext(context.TODO(), chart, values); err != nil {
		return err
	}
	return nil
}

func overwriteConfigRepo(address string, auth transport.AuthMethod, cfg string) error {
	repo, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:             address,
		Auth:            auth,
		RemoteName:      "soft",
		InsecureSkipTLS: true,
	})
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if err := func() error {
		f, err := wt.Filesystem.Create("config.yaml")
		if err != nil {
			return nil
		}
		defer f.Close()
		if _, err := io.WriteString(f, cfg); err != nil {
			return err
		}
		return nil

	}(); err != nil {
		return err
	}
	if _, err := wt.Add("config.yaml"); err != nil {
		return err
	}
	if _, err := wt.Commit("initial overwrite to give access to fluxcd", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "pcloud",
			Email: "pcloud@installer",
			When:  time.Now(),
		},
	}); err != nil {
		return err
	}
	if err = repo.Push(&git.PushOptions{
		RemoteName: "soft",
		Auth:       auth,
	}); err != nil {
		return err
	}
	return nil
}

func createRepo(address string, readme string, auth transport.AuthMethod) error {
	repo, err := git.Init(memory.NewStorage(), memfs.New())
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if err := func() error {
		f, err := wt.Filesystem.Create("README.md")
		if err != nil {
			return nil
		}
		defer f.Close()
		if _, err := io.WriteString(f, readme); err != nil {
			return err
		}
		return nil

	}(); err != nil {
		return err
	}
	if _, err := wt.Add("README.md"); err != nil {
		return err
	}
	if _, err := wt.Commit("init", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "pcloud",
			Email: "pcloud@installer",
			When:  time.Now(),
		},
	}); err != nil {
		return err
	}
	if _, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "soft",
		URLs: []string{address},
	}); err != nil {
		return err
	}
	if err = repo.Push(&git.PushOptions{
		RemoteName: "soft",
		Auth:       auth,
	}); err != nil {
		return err
	}
	return nil
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

func generateConfig(adminKeys []string) (string, error) {
	keys := make([]string, len(adminKeys))
	for i, key := range adminKeys {
		keys[i] = strings.Trim(key, " \n")
	}
	configT, err := template.New("config").Parse(configTmpl)
	if err != nil {
		return "", err
	}
	var configB strings.Builder
	if err := configT.Execute(&configB, keys); err != nil {
		return "", err
	}
	return configB.String(), nil
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

func createSSHAuthMethod(key []byte) (*gitssh.PublicKeys, error) {
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return &gitssh.PublicKeys{
		User:   "pcloud",
		Signer: signer,
	}, nil
}

func reloadConfig(addr string, clientPrivKey []byte, serverPubKey string) error {
	signer, err := ssh.ParsePrivateKey(clientPrivKey)
	if err != nil {
		return err
	}
	config := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			fmt.Printf("## %s || %s -- \n", serverPubKey, ssh.MarshalAuthorizedKey(key))
			return nil
		},
	}
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	return session.Run("reload")
}

func bootstrapCmdRun(cmd *cobra.Command, args []string) error {
	adminPubKey, adminPrivKey, err := readAdminKeys()
	if err != nil {
		return err
	}
	auth, err := createSSHAuthMethod(adminPrivKey)
	if err != nil {
		return err
	}
	fluxPub, fluxPriv, err := generateSSHKeys()
	if err != nil {
		return err
	}
	config, err := generateConfig([]string{string(adminPubKey), fluxPub})
	if err != nil {
		return err
	}
	softServePub, softServePriv, err := generateSSHKeys()
	if err != nil {
		return err
	}
	auth.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fmt.Printf("-- %s || %s -- \n", softServePub, ssh.MarshalAuthorizedKey(key))
		return nil
	}
	fmt.Println("Installing SoftServe")
	if err := installSoftServe(softServePub, softServePriv, string(adminPubKey)); err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	fmt.Println("Overwriting config")
	if err := overwriteConfigRepo("ssh://192.168.0.208:22/config", auth, config); err != nil {
		return err
	}
	fmt.Println("Reloading config")
	if err := reloadConfig("192.168.0.208:22", adminPrivKey, softServePub); err != nil {
		return err
	}
	fmt.Println("Creating /pcloud repo")
	if err := createRepo("ssh://192.168.0.208:22/pcloud", "PCloud System\n", auth); err != nil {
		return err
	}
	fmt.Println("Installing Flux")
	if err := installFlux("ssh://soft-serve.pcloud.svc.cluster.local:22/pcloud", "soft-serve.pcloud.svc.cluster.local", softServePub, fluxPriv); err != nil {
		return err
	}
	return nil
}

func installCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "install",
		RunE: installCmdRun,
	}
	cmd.Flags().StringVar(
		&installFlags.config,
		"config",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&installFlags.appName,
		"app",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&installFlags.outputDir,
		"output-dir",
		"",
		"",
	)
	return cmd
}

func init() {
	rootCmd = &cobra.Command{
		Use: "pcloud",
	}
	rootCmd.PersistentFlags().StringVar(
		&rootFlags.kubeConfig,
		"kubeconfig",
		"",
		"",
	)
	rootCmd.AddCommand(bootstrapCmd())
	rootCmd.AddCommand(installCmd())
}

func readConfig(config string) (Config, error) {
	var cfg Config
	inp, err := ioutil.ReadFile(config)
	if err != nil {
		return cfg, err
	}
	err = yaml.UnmarshalStrict(inp, &cfg)
	return cfg, err
}

func installCmdRun(cmd *cobra.Command, args []string) error {
	cfg, err := readConfig(installFlags.config)
	if err != nil {
		return err
	}
	tmpls, err := template.ParseFS(valuesTmpls, "values-tmpl/*.yaml")
	if err != nil {
		log.Fatal(err)
	}
	apps := CreateAllApps(tmpls)
	for _, a := range apps {
		if a.Name == installFlags.appName {
			for _, t := range a.Templates {
				out, err := os.Create(filepath.Join(installFlags.outputDir, t.Name()))
				if err != nil {
					return err
				}
				defer out.Close()
				if err := t.Execute(out, cfg); err != nil {
					return err
				}
			}
			break
		}
	}
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
