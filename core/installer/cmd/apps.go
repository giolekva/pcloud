package main

import (
	"net"
	"os"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

const appDirName = "apps"

var installFlags struct {
	sshKey   string
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

func installCmdRun(cmd *cobra.Command, args []string) error {
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
	m, err := installer.NewAppManager(
		repo,
		signer,
	)
	if err != nil {
		return err
	}
	appRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	app, err := appRepo.Find(installFlags.appName)
	if err != nil {
		return err
	}
	return m.Install(*app, nil)
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
