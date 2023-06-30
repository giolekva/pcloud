package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/giolekva/pcloud/core/installer"
)

var appManagerFlags struct {
	sshKey   string
	repoAddr string
	port     int
}

func appManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "appmanager",
		RunE: appManagerCmdRun,
	}
	cmd.Flags().StringVar(
		&appManagerFlags.sshKey,
		"ssh-key",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&appManagerFlags.repoAddr,
		"repo-addr",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&appManagerFlags.port,
		"port",
		8080,
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
	repo, err := cloneRepo(appManagerFlags.repoAddr, signer)
	if err != nil {
		return err
	}
	kube, err := newNSCreator()
	if err != nil {
		return err
	}
	m, err := installer.NewAppManager(
		installer.NewRepoIO(repo, signer),
		kube,
	)
	if err != nil {
		return err
	}
	r := installer.NewInMemoryAppRepository[installer.StoreApp](installer.CreateStoreApps())
	s := &server{
		port: appManagerFlags.port,
		m:    m,
		r:    r,
	}
	s.start()
	return nil
}

type server struct {
	port int
	m    *installer.AppManager
	r    installer.AppRepository[installer.StoreApp]
}

func (s *server) start() {
	e := echo.New()
	e.GET("/api/app-repo", s.handleAppRepo)
	e.POST("/api/app/:slug/render", s.handleAppRender)
	e.POST("/api/app/:slug/install", s.handleAppInstall)
	e.GET("/api/app/:slug", s.handleApp)
	e.GET("/api/instance/:slug", s.handleInstance)
	e.POST("/api/instance/:slug/update", s.handleAppUpdate)
	webapp, err := url.Parse("http://localhost:5173")
	if err != nil {
		panic(err)
	}
	// var f ff
	e.Any("/*", echo.WrapHandler(httputil.NewSingleHostReverseProxy(webapp)))
	// e.Any("/*", echo.WrapHandler(&f))
	fmt.Printf("Starting HTTP server on port: %d\n", s.port)
	log.Fatal(e.Start(fmt.Sprintf(":%d", s.port)))
}

type app struct {
	Name             string                `json:"name"`
	Icon             string                `json:"icon"`
	ShortDescription string                `json:"shortDescription"`
	Slug             string                `json:"slug"`
	Schema           string                `json:"schema"`
	Instances        []installer.AppConfig `json:"instances,omitempty"`
}

func (s *server) handleAppRepo(c echo.Context) error {
	all, err := s.r.GetAll()
	if err != nil {
		return err
	}
	resp := make([]app, len(all))
	for i, a := range all {
		resp[i] = app{a.Name, a.Icon, a.ShortDescription, a.Name, a.Schema, nil}
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *server) handleApp(c echo.Context) error {
	slug := c.Param("slug")
	a, err := s.r.Find(slug)
	if err != nil {
		return err
	}
	instances, err := s.m.FindAllInstances(slug)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, app{a.Name, a.Icon, a.ShortDescription, a.Name, a.Schema, instances})
}

func (s *server) handleInstance(c echo.Context) error {
	slug := c.Param("slug")
	instance, err := s.m.FindInstance(slug)
	if err != nil {
		return err
	}
	values, ok := instance.Config["Values"].(map[string]any)
	if !ok {
		return fmt.Errorf("Expected map")
	}
	for k, v := range values {
		if k == "Network" {
			n, ok := v.(map[string]any)
			if !ok {
				return fmt.Errorf("Expected map")
			}
			values["Network"], ok = n["Name"]
			if !ok {
				return fmt.Errorf("Missing Name")
			}
			break
		}

	}
	a, err := s.r.Find(instance.Id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, app{a.Name, a.Icon, a.ShortDescription, a.Name, a.Schema, []installer.AppConfig{instance}})
}

type file struct {
	Name     string `json:"name"`
	Contents string `json:"contents"`
}

type rendered struct {
	Readme string `json:"readme"`
	Files  []file `json:"files"`
}

type network struct {
	Name              string
	IngressClass      string
	CertificateIssuer string
	Domain            string
}

func createNetworks(global installer.Config) []network {
	return []network{
		{
			Name:              "Public",
			IngressClass:      fmt.Sprintf("%s-ingress-public", global.Values.PCloudEnvName),
			CertificateIssuer: fmt.Sprintf("%s-public", global.Values.Id),
			Domain:            global.Values.Domain,
		},
		{
			Name:         "Private",
			IngressClass: fmt.Sprintf("%s-ingress-private", global.Values.Id),
			Domain:       global.Values.PrivateDomain,
		},
	}
}

func (s *server) handleAppRender(c echo.Context) error {
	slug := c.Param("slug")
	contents, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	global, err := s.m.Config()
	if err != nil {
		return err
	}
	var values map[string]any
	if err := json.Unmarshal(contents, &values); err != nil {
		return err
	}
	if network, ok := values["Network"]; ok {
		for _, n := range createNetworks(global) {
			if n.Name == network { // TODO(giolekva): handle not found
				values["Network"] = n
			}
		}
	}
	all := map[string]any{
		"Global": global.Values,
		"Values": values,
	}
	a, err := s.r.Find(slug)
	if err != nil {
		return err
	}
	var readme bytes.Buffer
	if err := a.Readme.Execute(&readme, all); err != nil {
		return err
	}
	var resp rendered
	resp.Readme = readme.String()
	for _, tmpl := range a.Templates { // TODO(giolekva): deduplicate with Install
		var f bytes.Buffer
		if err := tmpl.Execute(&f, all); err != nil {
			fmt.Printf("%+v\n", all)
			fmt.Println(err.Error())
			return err
		} else {
			resp.Files = append(resp.Files, file{tmpl.Name(), f.String()})
		}
	}
	out, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := c.Response().Writer.Write(out); err != nil {
		return err
	}
	return nil
}

func (s *server) handleAppInstall(c echo.Context) error {
	slug := c.Param("slug")
	contents, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	var values map[string]any
	if err := json.Unmarshal(contents, &values); err != nil {
		return err
	}
	a, err := s.r.Find(slug)
	if err != nil {
		return err
	}
	config, err := s.m.Config()
	if err != nil {
		return err
	}
	if network, ok := values["Network"]; ok {
		for _, n := range createNetworks(config) {
			if n.Name == network { // TODO(giolekva): handle not found
				values["Network"] = n
			}
		}
	}
	nsGen := installer.NewPrefixGenerator(config.Values.NamespacePrefix)
	suffixGen := installer.NewFixedLengthRandomSuffixGenerator(3)
	if err := s.m.Install(a.App, nsGen, suffixGen, values); err != nil {
		return err
	}
	return c.String(http.StatusOK, "Installed")
}

func (s *server) handleAppUpdate(c echo.Context) error {
	slug := c.Param("slug")
	appConfig, err := s.m.AppConfig(slug)
	if err != nil {
		return err
	}
	contents, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	var values map[string]any
	if err := json.Unmarshal(contents, &values); err != nil {
		return err
	}
	a, err := s.r.Find(appConfig.Id)
	if err != nil {
		return err
	}
	config, err := s.m.Config()
	if err != nil {
		return err
	}
	if network, ok := values["Network"]; ok {
		for _, n := range createNetworks(config) {
			if n.Name == network { // TODO(giolekva): handle not found
				values["Network"] = n
			}
		}
	}
	if err := s.m.Update(a.App, slug, values); err != nil {
		return err
	}
	return c.String(http.StatusOK, "Installed")
}

func cloneRepo(address string, signer ssh.Signer) (*git.Repository, error) {
	return git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:             address,
		Auth:            auth(signer),
		RemoteName:      "origin",
		InsecureSkipTLS: true,
	})
}

func auth(signer ssh.Signer) *gitssh.PublicKeys {
	return &gitssh.PublicKeys{
		Signer: signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// TODO(giolekva): verify server public key
				// fmt.Printf("## %s || %s -- \n", serverPubKey, ssh.MarshalAuthorizedKey(key))
				return nil
			},
		},
	}
}
