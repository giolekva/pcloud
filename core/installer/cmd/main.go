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
	rootCmd.AddCommand(appManagerCmd())
	rootCmd.AddCommand(envManagerCmd())
	rootCmd.AddCommand(welcomeCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
