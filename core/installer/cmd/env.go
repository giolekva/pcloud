// TODO
// * flux -n lekva create source git pcloud --url=ssh://192.168.0.211/pcloud-apps --branch=main --private-key-file=/Users/lekva/.ssh/id_rsa

package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"text/template"

	"golang.org/x/exp/slices"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

//go:embed env-tmpl
var filesTmpls embed.FS

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
	ss, err := soft.NewClient("192.168.0.211", 22, adminPrivKey, log.Default())
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
	repo, err := ss.CloneRepository("pcloud")
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	envKust := "environments/kustomization.yaml"
	envKustFile, err := wt.Filesystem.Open(envKust)
	if err != nil {
		return err
	}
	kust, err := installer.ReadKustomization(envKustFile)
	if err != nil {
		return err
	}
	if slices.Contains(kust.Resources, createEnvFlags.name) {
		return fmt.Errorf("Environment already exists: %s", createEnvFlags.name)
	}
	tmpls, err := template.ParseFS(filesTmpls, "env-tmpl/*.yaml")
	if err != nil {
		return err
	}
	for _, tmpl := range tmpls.Templates() {
		dstPath := path.Join("environments", createEnvFlags.name, tmpl.Name())
		fmt.Println(dstPath)
		dst, err := wt.Filesystem.Create(dstPath)
		if err != nil {
			return err
		}
		if err := tmpl.Execute(dst, map[string]string{
			"Name":       createEnvFlags.name,
			"PrivateKey": base64.StdEncoding.EncodeToString([]byte(priv)),
			"PublicKey":  base64.StdEncoding.EncodeToString([]byte(pub)),
			"GitHost":    "192.168.0.211",
			"KnownHosts": base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("192.168.0.211 %s", ssPubKey))),
		}); err != nil {
			return err
		}
		if _, err := wt.Add(dstPath); err != nil {
			return err
		}
	}
	kust.Resources = append(kust.Resources, createEnvFlags.name)
	ff, err := wt.Filesystem.Create(envKust)
	if err != nil {
		return err
	}
	contents, err := yaml.Marshal(kust)
	if err != nil {
		return err
	}
	if _, err := io.Copy(ff, bytes.NewReader(contents)); err != nil {
		return err
	}
	if _, err := wt.Add(envKust); err != nil {
		return err
	}
	if err := ss.Commit(wt, fmt.Sprintf("%s: new environment", createEnvFlags.name)); err != nil {
		return err
	}
	if err := ss.Push(repo); err != nil {
		return err
	}
	return nil
}
