// TODO
// * flux -n lekva create source git pcloud --url=ssh://192.168.0.211/pcloud-apps --branch=main --private-key-file=/Users/lekva/.ssh/id_rsa

package main

import (
	"embed"
	"encoding/base64"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path"
	"text/template"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

//go:embed env-tmpl
var filesTmpls embed.FS

var createEnvFlags struct {
	name          string
	ip            string
	port          int
	adminPrivKey  string
	adminUsername string
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
	cmd.Flags().StringVar(
		&createEnvFlags.adminUsername,
		"admin-username",
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
	keys, err := installer.NewSSHKeyPair()
	if err != nil {
		return err
	}
	if 1 == 2 {
		readme := fmt.Sprintf("# %s PCloud environment", createEnvFlags.name)
		if err := ss.AddRepository(createEnvFlags.name, readme); err != nil {
			return err
		}
		fluxUserName := fmt.Sprintf("flux-%s", createEnvFlags.name)
		if err := ss.AddUser(fluxUserName, keys.Public); err != nil {
			return err
		}
		if err := ss.AddCollaborator(createEnvFlags.name, fluxUserName); err != nil {
			return err
		}
	}
	envRepo, err := ss.GetRepo(createEnvFlags.name)
	if envRepo == nil {
		return err
	}
	if err := initEnvRepo(installer.NewRepoIO(envRepo, ss.Signer)); err != nil {
		return err
	}
	if 1 == 2 {
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
				"PrivateKey": base64.StdEncoding.EncodeToString([]byte(keys.Private)),
				"PublicKey":  base64.StdEncoding.EncodeToString([]byte(keys.Public)),
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
	}
	return nil
}

func initEnvRepo(r installer.RepoIO) error {
	appManager, err := installer.NewAppManager(r)
	if err != nil {
		return err
	}
	appsRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	if 1 == 2 {
		config := installer.Config{ // TODO(gioleka): configurable
			Values: installer.Values{
				PCloudEnvName:   "pcloud",
				Id:              "lekva",
				ContactEmail:    "giolekva@gmail.com",
				Domain:          "lekva.me",
				PrivateDomain:   "p.lekva.me",
				PublicIP:        "46.49.35.44",
				NamespacePrefix: "lekva-",
			},
		}
		if err := r.WriteYaml("config.yaml", config); err != nil {
			return err
		}
		{
			out, err := r.Writer("pcloud-charts.yaml")
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = out.Write([]byte(`
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: GitRepository
metadata:
  name: pcloud
  namespace: lekva
spec:
  interval: 1m0s
  url: https://github.com/giolekva/pcloud
  ref:
    branch: main
`))
			if err != nil {
				return err
			}
		}
		rootKust := installer.NewKustomization()
		rootKust.AddResources("pcloud-charts.yaml", "apps")
		if err := r.WriteKustomization("kustomization.yaml", rootKust); err != nil {
			return err
		}
		appsKust := installer.NewKustomization()
		if err := r.WriteKustomization("apps/kustomization.yaml", appsKust); err != nil {
			return err
		}
		r.CommitAndPush("initialize config")
		{
			app, err := appsRepo.Find("metallb-config-env")
			if err != nil {
				return err
			}
			if err := appManager.Install(*app, map[string]any{
				"IngressPrivate": "10.1.0.1",
				"Headscale":      "10.1.0.2",
				"SoftServe":      "10.1.0.3",
				"Rest": map[string]any{
					"From": "10.1.0.100",
					"To":   "10.1.0.255",
				},
			}); err != nil {
				return err
			}
		}
		{
			app, err := appsRepo.Find("ingress-private")
			if err != nil {
				return err
			}
			if err := appManager.Install(*app, map[string]any{
				"GandiAPIToken": "", // TODO(gioleka): configurable
			}); err != nil {
				return err
			}
		}
		{
			app, err := appsRepo.Find("core-auth")
			if err != nil {
				return err
			}
			if err := appManager.Install(*app, map[string]any{
				"Subdomain": "test", // TODO(giolekva): make core-auth chart actually use this
			}); err != nil {
				return err
			}
		}
	}
	{
		app, err := appsRepo.Find("headscale")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, map[string]any{
			"Subdomain": "headscale",
		}); err != nil {
			return err
		}
	}
	if 1 == 2 {
		{
			app, err := appsRepo.Find("tailscale-proxy")
			if err != nil {
				return err
			}
			if err := appManager.Install(*app, map[string]any{
				"Username": createEnvFlags.adminUsername,
				"IPSubnet": "10.1.0.0/24",
			}); err != nil {
				return err
			}
			// TODO(giolekva): headscale accept routes
		}
	}
	return nil
}
