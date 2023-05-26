package main

import (
	"os"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var appManagerFlags struct {
	sshKey   string
	repoAddr string
}

func appManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "appmanager",
		RunE: installCmdRun,
	}
	cmd.Flags().StringVar(
		&installFlags.sshKey,
		"ssh-key",
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

func appManagerCmdRun(cmd *cobra.Command, args []string) error {
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
	_, err = installer.NewAppManager(repo, signer)
	if err != nil {
		return err
	}
	// TODO(gio): start server
	return nil
}
