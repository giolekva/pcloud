package main

import (
	"fmt"
	"log"
	"os"

	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/spf13/cobra"
)

var createEnvFlags struct {
	name         string
	adminPrivKey string
}

func createEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "create-env",
		RunE: createEnvCmdRun,
	}
	cmd.Flags().StringVar(
		&createEnvFlags.name,
		"name",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&createEnvFlags.adminPrivKey,
		"admin-priv-key",
		"",
		"",
	)
	return cmd
}

func createEnvCmdRun(cmd *cobra.Command, args []string) error {
	adminPrivKey, err := os.ReadFile(createEnvFlags.adminPrivKey)
	if err != nil {
		return err
	}
	ss, err := soft.NewClient("192.168.0.208", 22, adminPrivKey, log.Default())
	if err != nil {
		return err
	}
	readme := fmt.Sprintf("# %s PCloud environment", createEnvFlags.name)
	if err := ss.AddRepository(createEnvFlags.name, readme); err != nil {
		return err
	}
	return nil
}
