package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/welcome"

	_ "github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/spf13/cobra"
)

var dodoAppFlags struct {
	port              int
	apiPort           int
	sshKey            string
	repoAddr          string
	self              string
	namespace         string
	envAppManagerAddr string
	envConfig         string
	appAdminKey       string
	gitRepoPublicKey  string
	db                string
	networks          []string
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
	cmd.Flags().IntVar(
		&dodoAppFlags.apiPort,
		"api-port",
		8081,
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.db,
		"db",
		"",
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
		&dodoAppFlags.envAppManagerAddr,
		"env-app-manager-addr",
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
	cmd.Flags().StringSliceVar(
		&dodoAppFlags.networks,
		"networks",
		[]string{},
		"",
	)
	return cmd
}

func dodoAppCmdRun(cmd *cobra.Command, args []string) error {
	sshKey, err := os.ReadFile(dodoAppFlags.sshKey)
	if err != nil {
		return err
	}
	envConfig, err := os.Open(dodoAppFlags.envConfig)
	if err != nil {
		return err
	}
	defer envConfig.Close()
	var env installer.EnvConfig
	if err := json.NewDecoder(envConfig).Decode(&env); err != nil {
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
	if ok, err := softClient.RepoExists(welcome.ConfigRepoName); err != nil {
		return err
	} else if !ok {
		if err := softClient.AddRepository(welcome.ConfigRepoName); err != nil {
			return err
		}
	}
	configRepo, err := softClient.GetRepo(welcome.ConfigRepoName)
	if err != nil {
		return err
	}
	db, err := sql.Open("sqlite3", dodoAppFlags.db)
	if err != nil {
		return err
	}
	st, err := welcome.NewStore(configRepo, db)
	if err != nil {
		return err
	}
	s, err := welcome.NewDodoAppServer(
		st,
		dodoAppFlags.port,
		dodoAppFlags.apiPort,
		dodoAppFlags.self,
		string(sshKey),
		dodoAppFlags.gitRepoPublicKey,
		softClient,
		dodoAppFlags.namespace,
		dodoAppFlags.envAppManagerAddr,
		dodoAppFlags.networks,
		nsc,
		jc,
		env,
	)
	if err != nil {
		return err
	}
	if dodoAppFlags.appAdminKey != "" {
		if _, err := s.CreateApp("app", dodoAppFlags.appAdminKey, "Private"); err != nil {
			return err
		}
	}
	return s.Start()
}
