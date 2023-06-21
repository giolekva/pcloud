package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"text/template"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

var envManagerFlags struct {
	repoIP   string
	repoPort int
	sshKey   string
	port     int
}

func envManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "envmanager",
		RunE: envManagerCmdRun,
	}
	cmd.Flags().StringVar(
		&envManagerFlags.repoIP,
		"repo-ip",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&envManagerFlags.repoPort,
		"repo-port",
		22,
		"",
	)
	cmd.Flags().StringVar(
		&envManagerFlags.sshKey,
		"ssh-key",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&envManagerFlags.port,
		"port",
		8080,
		"",
	)
	return cmd
}

func envManagerCmdRun(cmd *cobra.Command, args []string) error {
	sshKey, err := os.ReadFile(envManagerFlags.sshKey)
	if err != nil {
		return err
	}
	fmt.Println(string(sshKey))
	ss, err := soft.NewClient(envManagerFlags.repoIP, envManagerFlags.repoPort, sshKey, log.Default())
	if err != nil {
		return err
	}
	b, err := ss.GetPublicKey()
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	fmt.Println(111)
	repo, err := ss.GetRepo("pcloud")
	fmt.Println(222)
	if err != nil {
		return err
	}
	fmt.Println(333)
	repoIO := installer.NewRepoIO(repo, ss.Signer)
	s := &envServer{
		port: envManagerFlags.port,
		ss:   ss,
		repo: repoIO,
	}
	s.start()
	return nil
}

type envServer struct {
	port int
	ss   *soft.Client
	repo installer.RepoIO
}

func (s *envServer) start() {
	e := echo.New()
	e.POST("/env", s.createEnv)
	log.Fatal(e.Start(fmt.Sprintf(":%d", s.port)))
}

type createEnvReq struct {
	Name          string `json:"name"`
	ContactEmail  string `json:"contactEmail"`
	Domain        string `json:"domain"`
	GandiAPIToken string `json:"gandiAPIToken"`
	AdminUsername string `json:"adminUsername"`
	// TODO(giolekva): take admin password as well
}

func (s *envServer) createEnv(c echo.Context) error {
	var req createEnvReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return err
	}
	keys, err := installer.NewSSHKeyPair()
	if err != nil {
		return err
	}
	{
		readme := fmt.Sprintf("# %s PCloud environment", req.Name)
		if err := s.ss.AddRepository(req.Name, readme); err != nil {
			return err
		}
		fluxUserName := fmt.Sprintf("flux-%s", req.Name)
		if err := s.ss.AddUser(fluxUserName, keys.Public); err != nil {
			return err
		}
		if err := s.ss.AddCollaborator(req.Name, fluxUserName); err != nil {
			return err
		}
	}
	{
		repo, err := s.ss.GetRepo(req.Name)
		if repo == nil {
			return err
		}
		if err := initNewEnv(installer.NewRepoIO(repo, s.ss.Signer), req); err != nil {
			return err
		}
	}
	{
		repo, err := s.ss.GetRepo("pcloud")
		if err != nil {
			return err
		}
		ssPubKey, err := s.ss.GetPublicKey()
		if err != nil {
			return err
		}
		if err := addNewEnv(
			installer.NewRepoIO(repo, s.ss.Signer),
			req,
			keys,
			ssPubKey,
		); err != nil {
			return err
		}
	}
	return nil
}

func initNewEnv(r installer.RepoIO, req createEnvReq) error {
	appManager, err := installer.NewAppManager(r)
	if err != nil {
		return err
	}
	appsRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	// TODO(giolekva): env name and ip should come from pcloud repo config.yaml
	// TODO(giolekva): private domain can be configurable as well
	config := installer.Config{
		Values: installer.Values{
			PCloudEnvName:   "pcloud",
			Id:              req.Name,
			ContactEmail:    req.ContactEmail,
			Domain:          req.Domain,
			PrivateDomain:   fmt.Sprintf("p.%s", req.Domain),
			PublicIP:        "46.49.35.44",
			NamespacePrefix: fmt.Sprintf("%s-", req.Name),
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
			"GandiAPIToken": req.GandiAPIToken,
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
	{
		app, err := appsRepo.Find("tailscale-proxy")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, map[string]any{
			"Username": req.AdminUsername,
			"IPSubnet": "10.1.0.0/24",
		}); err != nil {
			return err
		}
		// TODO(giolekva): headscale accept routes
	}

	return nil
}

func addNewEnv(
	repoIO installer.RepoIO,
	req createEnvReq,
	keys installer.KeyPair,
	pcloudRepoPublicKey []byte,
) error {
	kust, err := repoIO.ReadKustomization("environments/kustomization.yaml")
	if err != nil {
		return err
	}
	kust.AddResources(req.Name)
	tmpls, err := template.ParseFS(filesTmpls, "env-tmpl/*.yaml")
	if err != nil {
		return err
	}
	for _, tmpl := range tmpls.Templates() {
		dstPath := path.Join("environments", req.Name, tmpl.Name())
		dst, err := repoIO.Writer(dstPath)
		if err != nil {
			return err
		}
		defer dst.Close()
		if err := tmpl.Execute(dst, map[string]string{
			"Name":       req.Name,
			"PrivateKey": base64.StdEncoding.EncodeToString([]byte(keys.Private)),
			"PublicKey":  base64.StdEncoding.EncodeToString([]byte(keys.Public)),
			"GitHost":    envManagerFlags.repoIP,
			"KnownHosts": base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s %s", envManagerFlags.repoIP, pcloudRepoPublicKey))),
		}); err != nil {
			return err
		}
	}
	if err := repoIO.WriteKustomization("environments/kustomization.yaml", *kust); err != nil {
		return err
	}
	if err := repoIO.CommitAndPush(fmt.Sprintf("%s: initialize environment", req.Name)); err != nil {
		return err
	}
	return nil
}
