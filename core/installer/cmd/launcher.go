package main

import (
	"fmt"
	"os"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/welcome"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var launcherFlags struct {
	logoutUrl  string
	port       int
	repoAddr   string
	sshKey     string
	appManager *installer.AppManager
}

func launcherCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "launcher",
		RunE: launcherCmdRun,
	}
	cmd.Flags().IntVar(
		&launcherFlags.port,
		"port",
		8080,
		"",
	)
	cmd.Flags().StringVar(
		&launcherFlags.logoutUrl,
		"logout-url",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&launcherFlags.repoAddr,
		"repo-addr",
		"",
		"The address of the repository",
	)
	cmd.Flags().StringVar(
		&launcherFlags.sshKey,
		"ssh-key",
		"",
		"The path to the SSH key file",
	)
	return cmd
}

func launcherCmdRun(cmd *cobra.Command, args []string) error {
	sshKey, err := os.ReadFile(launcherFlags.sshKey)
	if err != nil {
		return fmt.Errorf("failed reading ssh key: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(sshKey)
	if err != nil {
		return fmt.Errorf("failed parsing ssh private key: %v", err)
	}
	addr, err := soft.ParseRepositoryAddress(launcherFlags.repoAddr)
	if err != nil {
		return err
	}
	repo, err := soft.CloneRepository(addr, signer)
	if err != nil {
		return fmt.Errorf("failed cloning repository: %v", err)
	}
	repoIO, err := soft.NewRepoIO(repo, signer)
	if err != nil {
		return fmt.Errorf("failed initializing RepoIO: %v", err)
	}
	appManager, err := installer.NewAppManager(repoIO, nil, "/apps")
	if err != nil {
		return fmt.Errorf("failed to create AppManager: %v", err)
	}
	s, err := welcome.NewLauncherServer(
		launcherFlags.port,
		launcherFlags.logoutUrl,
		&welcome.AppManagerDirectory{AppManager: appManager},
	)
	if err != nil {
		return fmt.Errorf("failed to create LauncherServer: %v", err)
	}
	s.Start()
	return nil
}
