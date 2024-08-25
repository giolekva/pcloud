package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ssh"

	"github.com/spf13/cobra"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

var rewriteFlags struct {
	path string
}

func rewriteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "rewrite",
		RunE: rewriteCmdRun,
	}
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
	return cmd
}

func rewriteCmdRun(cmd *cobra.Command, args []string) error {
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
	log.Println("Creating repository")
	r := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	hf := installer.NewGitHelmFetcher()
	mgr, err := installer.NewAppManager(repoIO, nil, nil, hf, nil, "/apps")
	if err != nil {
		return err
	}
	env, err := mgr.Config()
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", env)
	log.Println("Read config")
	if err != nil {
		return err
	}
	all, err := mgr.FindAllInstances()
	if err != nil {
		return err
	}
	for _, inst := range all {
		app, err := installer.FindEnvApp(r, inst.AppId)
		if err != nil {
			return err
		}
		v := inst.InputToValues(app.Schema())
		if _, err := mgr.Install(
			app,
			inst.Id,
			inst.Release.AppDir,
			inst.Release.Namespace,
			v,
			installer.WithNoPublish(),
		); err != nil {
			return err
		}
	}
	repoIO.CommitAndPush("upgrade: persist cue files")
	return nil
}
