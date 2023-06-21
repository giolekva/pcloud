package main

import (
	"log"

	"github.com/spf13/cobra"
)

var rootCmd *cobra.Command

var rootFlags struct {
	kubeConfig string
}

func init() {
	rootCmd = &cobra.Command{
		Use: "pcloud",
	}
	rootCmd.PersistentFlags().StringVar(
		&rootFlags.kubeConfig,
		"kubeconfig",
		"",
		"",
	)
	rootCmd.AddCommand(bootstrapCmd())
	rootCmd.AddCommand(createEnvCmd())
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(appManagerCmd())
	rootCmd.AddCommand(envManagerCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
