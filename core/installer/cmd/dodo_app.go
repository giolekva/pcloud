package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/welcome"

	"github.com/spf13/cobra"
)

var dodoAppFlags struct {
	port             int
	sshKey           string
	repoAddr         string
	self             string
	namespace        string
	envConfig        string
	appAdminKey      string
	gitRepoPublicKey string
}

func dodoAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "dodo-app",
		RunE: dodoAppCmdRun,
	}
	cmd.Flags().IntVar(
		&dodoAppFlags.port,
		"port",
		8080,
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.repoAddr,
		"repo-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.sshKey,
		"ssh-key",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.self,
		"self",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.namespace,
		"namespace",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.envConfig,
		"env-config",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.appAdminKey,
		"app-admin-key",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.gitRepoPublicKey,
		"git-repo-public-key",
		"",
		"",
	)
	return cmd
}

func dodoAppCmdRun(cmd *cobra.Command, args []string) error {
	envConfig, err := os.Open(dodoAppFlags.envConfig)
	if err != nil {
		return err
	}
	defer envConfig.Close()
	var env installer.EnvConfig
	if err := json.NewDecoder(envConfig).Decode(&env); err != nil {
		return err
	}
	sshKey, err := os.ReadFile(dodoAppFlags.sshKey)
	if err != nil {
		return err
	}
	cg := soft.RealClientGetter{}
	softClient, err := cg.Get(dodoAppFlags.repoAddr, sshKey, log.Default())
	if err != nil {
		return err
	}
	jc, err := newJobCreator()
	if err != nil {
		return err
	}
	nsc, err := newNSCreator()
	if err != nil {
		return err
	}
	s, err := welcome.NewDodoAppServer(
		dodoAppFlags.port,
		dodoAppFlags.self,
		string(sshKey),
		dodoAppFlags.gitRepoPublicKey,
		softClient,
		dodoAppFlags.namespace,
		nsc,
		jc,
		env,
	)
	if err != nil {
		return err
	}
	if dodoAppFlags.appAdminKey != "" {
		if err := s.CreateApp("app", dodoAppFlags.appAdminKey); err != nil {
			return err
		}
	}
	return s.Start()
}
