package main

import (
	"log"
	"os"

	"golang.org/x/crypto/ssh"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/tasks"
	"github.com/giolekva/pcloud/core/installer/welcome"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
)

var appManagerFlags struct {
	sshKey                 string
	repoAddr               string
	port                   int
	appRepoAddr            string
	headscaleAPIAddr       string
	dnsAPIAddr             string
	clusterProxyConfigPath string
}

func appManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "appmanager",
		RunE: appManagerCmdRun,
	}
	cmd.Flags().IntVar(
		&appManagerFlags.port,
		"port",
		8080,
		"",
	)
	cmd.Flags().StringVar(
		&appManagerFlags.repoAddr,
		"repo-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&appManagerFlags.sshKey,
		"ssh-key",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&appManagerFlags.appRepoAddr,
		"app-repo-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&appManagerFlags.headscaleAPIAddr,
		"headscale-api-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&appManagerFlags.dnsAPIAddr,
		"dns-api-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&appManagerFlags.clusterProxyConfigPath,
		"cluster-proxy-config-path",
		"",
		"",
	)
	return cmd
}

func appManagerCmdRun(cmd *cobra.Command, args []string) error {
	sshKey, err := os.ReadFile(appManagerFlags.sshKey)
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(sshKey)
	if err != nil {
		return err
	}
	addr, err := soft.ParseRepositoryAddress(appManagerFlags.repoAddr)
	if err != nil {
		return err
	}
	repo, err := soft.CloneRepository(addr, signer)
	if err != nil {
		return err
	}
	log.Println("Cloned repository")
	repoIO, err := soft.NewRepoIO(repo, signer)
	if err != nil {
		return err
	}
	nsc, err := newNSCreator()
	if err != nil {
		return err
	}
	jc, err := newJobCreator()
	if err != nil {
		return err
	}
	hf := installer.NewGitHelmFetcher()
	vpnAPIClient := installer.NewHeadscaleAPIClient(appManagerFlags.headscaleAPIAddr)
	cnc := &installer.NginxProxyConfigurator{
		// TODO(gio): read from env config
		PrivateSubdomain: "p",
		DNSAPIAddr:       appManagerFlags.dnsAPIAddr,
		Repo:             repoIO,
		NginxConfigPath:  appManagerFlags.clusterProxyConfigPath,
	}
	m, err := installer.NewAppManager(repoIO, nsc, jc, hf, vpnAPIClient, cnc, "/apps")
	if err != nil {
		return err
	}
	env, err := m.Config()
	if err != nil {
		return err
	}
	log.Println("Read config")
	log.Println("Creating repository")
	var r installer.AppRepository
	if appManagerFlags.appRepoAddr != "" {
		fs := memfs.New()
		err = installer.FetchAppsFromHTTPRepository(appManagerFlags.appRepoAddr, fs)
		if err != nil {
			return err
		}
		r, err = installer.NewFSAppRepository(fs)
		if err != nil {
			return err
		}
	} else {
		r = installer.NewInMemoryAppRepository(installer.CreateStoreApps())
	}
	fr := installer.NewInMemoryAppRepository(installer.CreateAllEnvApps())
	helmMon, err := newHelmReleaseMonitor()
	if err != nil {
		return err
	}
	s, err := welcome.NewAppManagerServer(
		appManagerFlags.port,
		repoIO,
		m,
		r,
		fr,
		tasks.NewFixedReconciler(env.Id, env.Id),
		helmMon,
		cnc,
		vpnAPIClient,
	)
	if err != nil {
		return err
	}
	return s.Start()
}
