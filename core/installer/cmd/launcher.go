package main

import (
	"fmt"
	"log"
	"os"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/welcome"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var launcherFlags struct {
	logoutURL string
	port      int
	repoAddr  string
	sshKey    string
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
		&launcherFlags.logoutURL,
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
		return err
	}
	signer, err := ssh.ParsePrivateKey(sshKey)
	if err != nil {
		return err
	}
	addr, err := soft.ParseRepositoryAddress(launcherFlags.repoAddr)
	if err != nil {
		return err
	}
	repo, err := soft.CloneRepository(addr, signer)
	if err != nil {
		return err
	}
	log.Println("Cloned repository")
	repoIO, err := soft.NewRepoIO(repo, signer)
	if err != nil {
		return err
	}
	appManager, err := installer.NewAppManager(repoIO, nil, nil, nil, nil, nil, "/apps")
	if err != nil {
		return err
	}
	s, err := welcome.NewLauncherServer(
		launcherFlags.port,
		launcherFlags.logoutURL,
		&welcome.AppManagerDirectory{AppManager: appManager},
	)
	if err != nil {
		return fmt.Errorf("failed to create LauncherServer: %v", err)
	}
	s.Start()
	return nil
}
