package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/welcome"
)

var envManagerFlags struct {
	repoIP   string
	repoPort int
	sshKey   string
	port     int
}

func envManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "envmanager",
		RunE: envManagerCmdRun,
	}
	cmd.Flags().StringVar(
		&envManagerFlags.repoIP,
		"repo-ip",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&envManagerFlags.repoPort,
		"repo-port",
		22,
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
	sshKey, err := os.ReadFile(envManagerFlags.sshKey)
	if err != nil {
		return err
	}
	ss, err := soft.NewClient(envManagerFlags.repoIP, envManagerFlags.repoPort, sshKey, log.Default())
	if err != nil {
		return err
	}
	repo, err := ss.GetRepo("pcloud")
	if err != nil {
		return err
	}
	repoIO := installer.NewRepoIO(repo, ss.Signer)
	s := welcome.NewEnvServer(
		envManagerFlags.port,
		ss,
		repoIO,
	)
	s.Start()
	return nil
}
