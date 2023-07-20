package main

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/welcome"
)

var appManagerFlags struct {
	sshKey     string
	repoAddr   string
	port       int
	webAppAddr string
}

func appManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "appmanager",
		RunE: appManagerCmdRun,
	}
	cmd.Flags().StringVar(
		&appManagerFlags.sshKey,
		"ssh-key",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&appManagerFlags.repoAddr,
		"repo-addr",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&appManagerFlags.port,
		"port",
		8080,
		"",
	)
	cmd.Flags().StringVar(
		&appManagerFlags.webAppAddr,
		"web-app-addr",
		"",
		"",
	)
	return cmd
}

func appManagerCmdRun(cmd *cobra.Command, args []string) error {
	sshKey, err := os.ReadFile(appManagerFlags.sshKey)
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(sshKey)
	if err != nil {
		return err
	}
	addr, err := soft.ParseRepositoryAddress(appManagerFlags.repoAddr)
	if err != nil {
		return err
	}
	repo, err := soft.CloneRepo(addr, signer)
	if err != nil {
		return err
	}
	kube, err := newNSCreator()
	if err != nil {
		return err
	}
	m, err := installer.NewAppManager(
		installer.NewRepoIO(repo, signer),
		kube,
	)
	if err != nil {
		return err
	}
	r := installer.NewInMemoryAppRepository[installer.StoreApp](installer.CreateStoreApps())
	s := welcome.NewAppManagerServer(
		appManagerFlags.port,
		appManagerFlags.webAppAddr,
		m,
		r,
	)
	s.Start()
	return nil
}
