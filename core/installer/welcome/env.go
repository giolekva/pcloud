package welcome

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"text/template"

	"github.com/labstack/echo/v4"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

//go:embed env-tmpl
var filesTmpls embed.FS

//go:embed create-env.html
var createEnvFormHtml string

type EnvServer struct {
	port      int
	ss        *soft.Client
	repo      installer.RepoIO
	nsCreator installer.NamespaceCreator
}

func NewEnvServer(port int, ss *soft.Client, repo installer.RepoIO, nsCreator installer.NamespaceCreator) *EnvServer {
	return &EnvServer{
		port,
		ss,
		repo,
		nsCreator,
	}
}

func (s *EnvServer) Start() {
	e := echo.New()
	e.StaticFS("/static", echo.MustSubFS(staticAssets, "static"))
	e.GET("/env", s.createEnvForm)
	e.POST("/env", s.createEnv)
	log.Fatal(e.Start(fmt.Sprintf(":%d", s.port)))
}

func (s *EnvServer) createEnvForm(c echo.Context) error {
	return c.HTML(http.StatusOK, createEnvFormHtml)
}

type createEnvReq struct {
	Name         string `json:"name"`
	ContactEmail string `json:"contactEmail"`
	Domain       string `json:"domain"`
}

func (s *EnvServer) createEnv(c echo.Context) error {
	var req createEnvReq
	if err := func() error {
		var err error
		f, err := c.FormParams()
		if err != nil {
			return err
		}
		if req.Name, err = getFormValue(f, "name"); err != nil {
			return err
		}
		if req.Domain, err = getFormValue(f, "domain"); err != nil {
			return err
		}
		if req.ContactEmail, err = getFormValue(f, "contact-email"); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return err
		}
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
		if err != nil {
			return err
		}
		var env installer.EnvConfig
		r, err := s.repo.Reader("config.yaml")
		if err != nil {
			return err
		}
		defer r.Close()
		if err := installer.ReadYaml(r, &env); err != nil {
			return err
		}
		if err := initNewEnv(s.ss, installer.NewRepoIO(repo, s.ss.Signer), s.nsCreator, req, env); err != nil {
			return err
		}
	}
	{
		ssPubKey, err := s.ss.GetPublicKey()
		if err != nil {
			return err
		}
		if err := addNewEnv(
			s.repo,
			req,
			keys,
			ssPubKey,
		); err != nil {
			return err
		}
	}
	return c.String(http.StatusOK, "OK")
}

func initNewEnv(
	ss *soft.Client,
	r installer.RepoIO,
	nsCreator installer.NamespaceCreator,
	req createEnvReq,
	env installer.EnvConfig,
) error {
	appManager, err := installer.NewAppManager(r, nsCreator)
	if err != nil {
		return err
	}
	appsRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	// TODO(giolekva): private domain can be configurable as well
	config := installer.Config{
		Values: installer.Values{
			PCloudEnvName:   env.Name,
			Id:              req.Name,
			ContactEmail:    req.ContactEmail,
			Domain:          req.Domain,
			PrivateDomain:   fmt.Sprintf("p.%s", req.Domain),
			PublicIP:        env.PublicIP,
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
		_, err = out.Write([]byte(fmt.Sprintf(`
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: pcloud
  namespace: %s
spec:
  interval: 1m0s
  url: https://github.com/giolekva/pcloud
  ref:
    branch: main
`, req.Name)))
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
	nsGen := installer.NewPrefixGenerator(req.Name + "-")
	suffixGen := installer.NewEmptySuffixGenerator()
	{
		app, err := appsRepo.Find("metallb-ipaddresspool")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{
			"Name":       fmt.Sprintf("%s-ingress-private", req.Name),
			"From":       "10.1.0.1",
			"To":         "10.1.0.1",
			"AutoAssign": false,
		}); err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{
			"Name":       fmt.Sprintf("%s-headscale", req.Name),
			"From":       "10.1.0.2",
			"To":         "10.1.0.2",
			"AutoAssign": false,
		}); err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{
			"Name":       fmt.Sprintf("%s-soft-serve", req.Name), // TODO(giolekva): rename to config repo
			"From":       "10.1.0.3",
			"To":         "10.1.0.3",
			"AutoAssign": false,
		}); err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{
			"Name":       req.Name,
			"From":       "10.1.0.100",
			"To":         "10.1.0.254",
			"AutoAssign": false,
		}); err != nil {
			return err
		}
	}
	{
		app, err := appsRepo.Find("ingress-private")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{}); err != nil {
			return err
		}
	}
	{
		app, err := appsRepo.Find("certificate-issuer-public")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{}); err != nil {
			return err
		}
	}
	{
		app, err := appsRepo.Find("core-auth")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{
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
		if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{
			"Subdomain": "headscale",
		}); err != nil {
			return err
		}
	}
	{
		keys, err := installer.NewSSHKeyPair()
		if err != nil {
			return err
		}
		user := fmt.Sprintf("%s-welcome", req.Name)
		if err := ss.AddUser(user, keys.Public); err != nil {
			return err
		}
		if err := ss.AddCollaborator(req.Name, user); err != nil {
			return err
		}
		app, err := appsRepo.Find("welcome")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{
			"RepoAddr":      ss.GetRepoAddress(req.Name),
			"SSHPrivateKey": keys.Private,
		}); err != nil {
			return err
		}
	}
	{
		keys, err := installer.NewSSHKeyPair()
		if err != nil {
			return err
		}
		user := fmt.Sprintf("%s-appmanager", req.Name)
		if err := ss.AddUser(user, keys.Public); err != nil {
			return err
		}
		if err := ss.AddCollaborator(req.Name, user); err != nil {
			return err
		}
		app, err := appsRepo.Find("app-manager") // TODO(giolekva): configure
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{
			"RepoAddr":      ss.GetRepoAddress(req.Name),
			"SSHPrivateKey": keys.Private,
		}); err != nil {
			return err
		}
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
	repoIP := repoIO.Addr().Addr().String()
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
			"GitHost":    repoIP,
			"KnownHosts": base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s %s", repoIP, pcloudRepoPublicKey))),
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
