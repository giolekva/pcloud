package main

import (
	"os"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/welcome"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var welcomeFlags struct {
	repo              string
	sshKey            string
	port              int
	createAccountAddr string
	loginAddr         string
}

func welcomeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "welcome",
		RunE: welcomeCmdRun,
	}
	cmd.Flags().StringVar(
		&welcomeFlags.repo,
		"repo-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&welcomeFlags.sshKey,
		"ssh-key",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&welcomeFlags.port,
		"port",
		8080,
		"",
	)
	cmd.Flags().StringVar(
		&welcomeFlags.createAccountAddr,
		"create-account-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&welcomeFlags.loginAddr,
		"login-addr",
		"",
		"",
	)
	return cmd
}

func welcomeCmdRun(cmd *cobra.Command, args []string) error {
	sshKey, err := os.ReadFile(welcomeFlags.sshKey)
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(sshKey)
	if err != nil {
		return err
	}
	addr, err := soft.ParseRepositoryAddress(welcomeFlags.repo)
	if err != nil {
		return err
	}
	repo, err := soft.CloneRepository(addr, signer)
	if err != nil {
		return err
	}
	nsCreator, err := newNSCreator()
	if err != nil {
		return err
	}
	s := welcome.NewServer(
		welcomeFlags.port,
		installer.NewRepoIO(repo, signer),
		nsCreator,
		welcomeFlags.createAccountAddr,
		welcomeFlags.loginAddr,
	)
	s.Start()
	return nil
}
