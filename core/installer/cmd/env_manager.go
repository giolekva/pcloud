package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/welcome"
)

var envManagerFlags struct {
	repoAddr string
	repoName string
	sshKey   string
	port     int
}

func envManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "envmanager",
		RunE: envManagerCmdRun,
	}
	cmd.Flags().StringVar(
		&envManagerFlags.repoAddr,
		"repo-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&envManagerFlags.repoName,
		"repo-name",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&envManagerFlags.sshKey,
		"ssh-key",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&envManagerFlags.port,
		"port",
		8080,
		"",
	)
	return cmd
}

func envManagerCmdRun(cmd *cobra.Command, args []string) error {
	sshKey, err := installer.NewSSHKeyPair(envManagerFlags.sshKey)
	if err != nil {
		return err
	}
	ss, err := soft.WaitForClient(envManagerFlags.repoAddr, sshKey.RawPrivateKey(), log.Default())
	if err != nil {
		return err
	}
	log.Printf("Created Soft Serve client\n")
	repo, err := ss.GetRepo(envManagerFlags.repoName)
	if err != nil {
		return err
	}
	log.Printf("Cloned repo: %s\n", envManagerFlags.repoName)
	repoIO := installer.NewRepoIO(repo, sshKey.Signer())
	nsCreator, err := newNSCreator()
	if err != nil {
		return err
	}
	dnsFetcher, err := newZoneFetcher()
	if err != nil {
		return err
	}
	s := welcome.NewEnvServer(
		envManagerFlags.port,
		ss,
		repoIO,
		nsCreator,
		dnsFetcher,
		installer.NewFixedLengthRandomNameGenerator(4),
	)
	log.Printf("Starting server\n")
	s.Start()
	return nil
}
