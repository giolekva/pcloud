// TODO
// * flux -n lekva create source git pcloud --url=ssh://192.168.0.211/pcloud-apps --branch=main --private-key-file=/Users/lekva/.ssh/id_rsa

package main

import (
	"embed"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path"
	"text/template"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/spf13/cobra"
)

//go:embed env-tmpl
var filesTmpls embed.FS

var createEnvFlags struct {
	name         string
	ip           string
	port         int
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
		&createEnvFlags.ip,
		"ip",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&createEnvFlags.port,
		"port",
		22,
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
	ss, err := soft.NewClient(createEnvFlags.ip, createEnvFlags.port, adminPrivKey, log.Default())
	if err != nil {
		return err
	}
	ssPubKey, err := ss.GetPublicKey()
	if err != nil {
		return err
	}
	fmt.Println(string(ssPubKey))
	pub, priv, err := installer.GenerateSSHKeys()
	{
		_ = priv
	}
	if err != nil {
		return err
	}
	readme := fmt.Sprintf("# %s PCloud environment", createEnvFlags.name)
	if err := ss.AddRepository(createEnvFlags.name, readme); err != nil {
		return err
	}
	fluxUserName := fmt.Sprintf("flux-%s", createEnvFlags.name)
	if err := ss.AddUser(fluxUserName, pub); err != nil {
		return err
	}
	if err := ss.AddCollaborator(createEnvFlags.name, fluxUserName); err != nil {
		return err
	}
	repo, err := ss.GetRepo("pcloud")
	if err != nil {
		return err
	}
	repoIO := installer.NewRepoIO(repo, ss.Signer)
	kust, err := repoIO.ReadKustomization("environments/kustomization.yaml")
	if err != nil {
		return err
	}
	kust.AddResources(createEnvFlags.name)
	tmpls, err := template.ParseFS(filesTmpls, "env-tmpl/*.yaml")
	if err != nil {
		return err
	}
	for _, tmpl := range tmpls.Templates() {
		dstPath := path.Join("environments", createEnvFlags.name, tmpl.Name())
		dst, err := repoIO.Writer(dstPath)
		if err != nil {
			return err
		}
		defer dst.Close()
		if err := tmpl.Execute(dst, map[string]string{
			"Name":       createEnvFlags.name,
			"PrivateKey": base64.StdEncoding.EncodeToString([]byte(priv)),
			"PublicKey":  base64.StdEncoding.EncodeToString([]byte(pub)),
			"GitHost":    createEnvFlags.ip,
			"KnownHosts": base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s %s", createEnvFlags.ip, ssPubKey))),
		}); err != nil {
			return err
		}
	}
	if err := repoIO.WriteKustomization("environments/kustomization.yaml", *kust); err != nil {
		return err
	}
	if err := repoIO.CommitAndPush(fmt.Sprintf("%s: initialize environment", createEnvFlags.name)); err != nil {
		return err
	}
	return nil
}
