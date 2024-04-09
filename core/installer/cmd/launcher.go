package main

import (
	"fmt"

	"github.com/giolekva/pcloud/core/installer/welcome"
	"github.com/spf13/cobra"
)

var launcherFlags struct {
	logoutUrl string
	port      int
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
	return cmd
}

func launcherCmdRun(cmd *cobra.Command, args []string) error {
	s, err := welcome.NewLauncherServer(
		launcherFlags.port,
		launcherFlags.logoutUrl,
		&welcome.FakeAppDirectory{},
	)
	if err != nil {
		return fmt.Errorf("failed to create LauncherServer: %v", err)
	}
	s.Start()
	return nil
}
