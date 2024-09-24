package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/tasks"
	"github.com/giolekva/pcloud/core/installer/welcome"

	_ "github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/spf13/cobra"
)

var dodoAppFlags struct {
	external          bool
	port              int
	apiPort           int
	sshKey            string
	repoAddr          string
	self              string
	selfPublic        string
	repoPublicAddr    string
	namespace         string
	envAppManagerAddr string
	envConfig         string
	gitRepoPublicKey  string
	db                string
	networks          []string
	fetchUsersAddr    string
	headscaleAPIAddr  string
}

func dodoAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "dodo-app",
		RunE: dodoAppCmdRun,
	}
	cmd.Flags().BoolVar(
		&dodoAppFlags.external,
		"external",
		false,
		"",
	)
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
		&dodoAppFlags.selfPublic,
		"self-public",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.fetchUsersAddr,
		"fetch-users-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.repoPublicAddr,
		"repo-public-addr",
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
	cmd.Flags().StringVar(
		&dodoAppFlags.headscaleAPIAddr,
		"headscale-api-addr",
		"",
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
	var nf welcome.NetworkFilter
	if len(dodoAppFlags.networks) == 0 {
		nf = welcome.NewNoNetworkFilter()
	} else {
		nf = welcome.NewAllowListFilter(dodoAppFlags.networks)
	}
	if dodoAppFlags.external {
		nf = welcome.NewCombinedFilter(welcome.NewNetworkFilterByOwner(st), nf)
	}
	var ug welcome.UserGetter
	if dodoAppFlags.external {
		ug = welcome.NewExternalUserGetter()
	} else {
		ug = welcome.NewInternalUserGetter()
	}
	reconciler := &tasks.SequentialReconciler{
		[]tasks.Reconciler{
			&tasks.SourceGitReconciler{},
			// &tasks.KustomizationReconciler{},
		},
	}
	vpnKeyGen := installer.NewHeadscaleAPIClient(dodoAppFlags.headscaleAPIAddr)
	cnc := &proxyConfigurator{dodoAppFlags.envAppManagerAddr}
	s, err := welcome.NewDodoAppServer(
		st,
		nf,
		ug,
		dodoAppFlags.port,
		dodoAppFlags.apiPort,
		dodoAppFlags.self,
		dodoAppFlags.selfPublic,
		dodoAppFlags.repoPublicAddr,
		string(sshKey),
		dodoAppFlags.gitRepoPublicKey,
		softClient,
		dodoAppFlags.namespace,
		dodoAppFlags.envAppManagerAddr,
		nsc,
		jc,
		vpnKeyGen,
		cnc,
		env,
		dodoAppFlags.external,
		dodoAppFlags.fetchUsersAddr,
		reconciler,
	)
	if err != nil {
		return err
	}
	return s.Start()
}

type proxyConfigurator struct {
	apiAddr string
}

func (pc *proxyConfigurator) AddCluster(name string, ingressIP net.IP) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}

func (pc *proxyConfigurator) RemoveCluster(name string, ingressIP net.IP) error {
	return fmt.Errorf("NOT IMPLEMENTED")
}

type proxyPair struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (pc *proxyConfigurator) AddProxy(src, dst string) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(proxyPair{src, dst}); err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("%s/api/proxy/add", pc.apiAddr), "application/json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		io.Copy(&buf, resp.Body)
		return fmt.Errorf(buf.String())
	}
	return nil
}

func (pc *proxyConfigurator) RemoveProxy(src, dst string) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(proxyPair{src, dst}); err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("%s/api/proxy/remove", pc.apiAddr), "application/json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		io.Copy(&buf, resp.Body)
		return fmt.Errorf(buf.String())
	}
	return nil
}
