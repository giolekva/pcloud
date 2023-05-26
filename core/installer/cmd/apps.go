package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"
)

const appDirName = "apps"

var installFlags struct {
	sshKey   string
	config   string
	appName  string
	repoAddr string
}

func installCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "install",
		RunE: installCmdRun,
	}
	cmd.Flags().StringVar(
		&installFlags.sshKey,
		"ssh-key",
		"",
		"",
	)
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
		&installFlags.repoAddr,
		"repo-addr",
		"",
		"",
	)
	return cmd
}

type inMemoryAppRepository struct {
	apps []installer.App
}

func NewInMemoryAppRepository(apps []installer.App) installer.AppRepository {
	return &inMemoryAppRepository{
		apps,
	}
}

func (r inMemoryAppRepository) Find(name string) (*installer.App, error) {
	for _, a := range r.apps {
		if a.Name == name {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("Application not found: %s", name)
}

func installCmdRun(cmd *cobra.Command, args []string) error {
	cfg, err := readConfig(installFlags.config)
	if err != nil {
		return err
	}
	sshKey, err := os.ReadFile(installFlags.sshKey)
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(sshKey)
	if err != nil {
		return err
	}
	repo, err := cloneRepo(installFlags.repoAddr, signer)
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	appRoot, err := wt.Filesystem.Chroot(appDirName)
	if err != nil {
		return err
	}
	m, err := installer.NewAppManager(
		appRoot,
		cfg,
		NewInMemoryAppRepository(installer.CreateAllApps()),
	)
	if err != nil {
		return err
	}
	if err := m.Install(installFlags.appName); err != nil {
		return err
	}
	if st, err := wt.Status(); err != nil {
		return err
	} else {
		fmt.Printf("%+v\n", st)
	}
	wt.AddGlob("*")
	if st, err := wt.Status(); err != nil {
		return err
	} else {
		fmt.Printf("%+v\n", st)
	}
	if _, err := wt.Commit(fmt.Sprintf("install: %s", installFlags.appName), &git.CommitOptions{
		Author: &object.Signature{
			Name: "pcloud-appmanager",
			When: time.Now(),
		},
	}); err != nil {
		return err
	}
	return repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth(signer),
	})
}

func readConfig(config string) (installer.Config, error) {
	var cfg installer.Config
	inp, err := ioutil.ReadFile(config)
	if err != nil {
		return cfg, err
	}
	err = yaml.UnmarshalStrict(inp, &cfg)
	return cfg, err
}

func cloneRepo(address string, signer ssh.Signer) (*git.Repository, error) {
	return git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:             address,
		Auth:            auth(signer),
		RemoteName:      "origin",
		InsecureSkipTLS: true,
	})
}

func auth(signer ssh.Signer) *gitssh.PublicKeys {
	return &gitssh.PublicKeys{
		Signer: signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// TODO(giolekva): verify server public key
				// fmt.Printf("## %s || %s -- \n", serverPubKey, ssh.MarshalAuthorizedKey(key))
				return nil
			},
		},
	}
}
