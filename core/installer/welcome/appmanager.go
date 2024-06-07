package welcome

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/labstack/echo/v4"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/tasks"
)

//go:embed appmanager-tmpl/*
var appTmpls embed.FS

type AppManagerServer struct {
	port       int
	m          *installer.AppManager
	r          installer.AppRepository
	reconciler tasks.Reconciler
	h          installer.HelmReleaseMonitor
	tasks      map[string]tasks.Task
	tmpl       tmplts
}

type tmplts struct {
	index *template.Template
	app   *template.Template
}

func parseTemplatesAppManager(fs embed.FS) (tmplts, error) {
	base, err := template.New("base.html").Funcs(template.FuncMap(sprig.FuncMap())).ParseFS(fs, "appmanager-tmpl/base.html")
	if err != nil {
		return tmplts{}, err
	}
	parse := func(path string) (*template.Template, error) {
		if b, err := base.Clone(); err != nil {
			return nil, err
		} else {
			return b.ParseFS(fs, path)
		}
	}
	index, err := parse("appmanager-tmpl/index.html")
	if err != nil {
		return tmplts{}, err
	}
	app, err := parse("appmanager-tmpl/app.html")
	if err != nil {
		return tmplts{}, err
	}
	return tmplts{index, app}, nil
}

func NewAppManagerServer(
	port int,
	m *installer.AppManager,
	r installer.AppRepository,
	reconciler tasks.Reconciler,
	h installer.HelmReleaseMonitor,
) (*AppManagerServer, error) {
	tmpl, err := parseTemplatesAppManager(appTmpls)
	if err != nil {
		return nil, err
	}
	return &AppManagerServer{
		port:       port,
		m:          m,
		r:          r,
		reconciler: reconciler,
		h:          h,
		tasks:      make(map[string]tasks.Task),
		tmpl:       tmpl,
	}, nil
}

func (s *AppManagerServer) Start() error {
	e := echo.New()
	e.StaticFS("/static", echo.MustSubFS(staticAssets, "static"))
	e.GET("/api/app-repo", s.handleAppRepo)
	e.POST("/api/app/:slug/install", s.handleAppInstall)
	e.GET("/api/app/:slug", s.handleApp)
	e.GET("/api/instance/:slug", s.handleInstance)
	e.POST("/api/instance/:slug/update", s.handleAppUpdate)
	e.POST("/api/instance/:slug/remove", s.handleAppRemove)
	e.GET("/", s.handleIndex)
	e.GET("/not-installed", s.handleNotInstalledApps)
	e.GET("/installed", s.handleInstalledApps)
	e.GET("/app/:slug", s.handleAppUI)
	e.GET("/instance/:slug", s.handleInstanceUI)
	fmt.Printf("Starting HTTP server on port: %d\n", s.port)
	return e.Start(fmt.Sprintf(":%d", s.port))
}

type app struct {
	Name             string                        `json:"name"`
	Icon             template.HTML                 `json:"icon"`
	ShortDescription string                        `json:"shortDescription"`
	Slug             string                        `json:"slug"`
	Instances        []installer.AppInstanceConfig `json:"instances,omitempty"`
}

func (s *AppManagerServer) handleAppRepo(c echo.Context) error {
	all, err := s.r.GetAll()
	if err != nil {
		return err
	}
	resp := make([]app, len(all))
	for i, a := range all {
		resp[i] = app{a.Name(), a.Icon(), a.Description(), a.Slug(), nil}
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *AppManagerServer) handleApp(c echo.Context) error {
	slug := c.Param("slug")
	a, err := s.r.Find(slug)
	if err != nil {
		return err
	}
	instances, err := s.m.FindAllAppInstances(slug)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, app{a.Name(), a.Icon(), a.Description(), a.Slug(), instances})
}

func (s *AppManagerServer) handleInstance(c echo.Context) error {
	slug := c.Param("slug")
	instance, err := s.m.FindInstance(slug)
	if err != nil {
		return err
	}
	a, err := s.r.Find(instance.AppId)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, app{a.Name(), a.Icon(), a.Description(), a.Slug(), []installer.AppInstanceConfig{*instance}})
}

func (s *AppManagerServer) handleAppInstall(c echo.Context) error {
	slug := c.Param("slug")
	contents, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	var values map[string]any
	if err := json.Unmarshal(contents, &values); err != nil {
		return err
	}
	log.Printf("Values: %+v\n", values)
	a, err := installer.FindEnvApp(s.r, slug)
	if err != nil {
		return err
	}
	log.Printf("Found application: %s\n", slug)
	env, err := s.m.Config()
	if err != nil {
		return err
	}
	log.Printf("Configuration: %+v\n", env)
	suffixGen := installer.NewFixedLengthRandomSuffixGenerator(3)
	suffix, err := suffixGen.Generate()
	if err != nil {
		return err
	}
	instanceId := a.Slug() + suffix
	appDir := fmt.Sprintf("/apps/%s", instanceId)
	namespace := fmt.Sprintf("%s%s%s", env.NamespacePrefix, a.Namespace(), suffix)
	rr, err := s.m.Install(a, instanceId, appDir, namespace, values)
	if err != nil {
		return err
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	go s.reconciler.Reconcile(ctx)
	if _, ok := s.tasks[instanceId]; ok {
		panic("MUST NOT REACH!")
	}
	t := tasks.NewMonitorRelease(s.h, rr)
	t.OnDone(func(err error) {
		delete(s.tasks, instanceId)
	})
	s.tasks[instanceId] = t
	go t.Start()
	return c.String(http.StatusOK, fmt.Sprintf("/instance/%s", instanceId))
}

func (s *AppManagerServer) handleAppUpdate(c echo.Context) error {
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
	a, err := installer.FindEnvApp(s.r, appConfig.AppId)
	if err != nil {
		return err
	}
	if _, ok := s.tasks[slug]; ok {
		return fmt.Errorf("Update already in progress")
	}
	rr, err := s.m.Update(a, slug, values)
	if err != nil {
		return err
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	go s.reconciler.Reconcile(ctx)
	t := tasks.NewMonitorRelease(s.h, rr)
	t.OnDone(func(err error) {
		delete(s.tasks, slug)
	})
	s.tasks[slug] = t
	go t.Start()
	return c.String(http.StatusOK, fmt.Sprintf("/instance/%s", slug))
}

func (s *AppManagerServer) handleAppRemove(c echo.Context) error {
	slug := c.Param("slug")
	if err := s.m.Remove(slug); err != nil {
		return err
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	go s.reconciler.Reconcile(ctx)
	return c.String(http.StatusOK, "/")
}

type PageData struct {
	Apps        []app
	CurrentPage string
}

func (s *AppManagerServer) handleIndex(c echo.Context) error {
	all, err := s.r.GetAll()
	if err != nil {
		log.Printf("all apps: %v", err)
		return err
	}
	resp := make([]app, 0)
	for _, a := range all {
		instances, err := s.m.FindAllAppInstances(a.Slug())
		if err != nil {
			return err
		}
		resp = append(resp, app{a.Name(), a.Icon(), a.Description(), a.Slug(), instances})
	}
	data := PageData{
		Apps:        resp,
		CurrentPage: "ALL",
	}
	if err := s.tmpl.index.Execute(c.Response(), data); err != nil {
		log.Printf("executing template: %v", err)
		return err
	}
	return nil
}

func (s *AppManagerServer) handleNotInstalledApps(c echo.Context) error {
	all, err := s.r.GetAll()
	if err != nil {
		return err
	}
	resp := make([]app, 0)
	for _, a := range all {
		instances, err := s.m.FindAllAppInstances(a.Slug())
		if err != nil {
			return err
		}
		if len(instances) == 0 {
			resp = append(resp, app{a.Name(), a.Icon(), a.Description(), a.Slug(), nil})
		}
	}
	data := PageData{
		Apps:        resp,
		CurrentPage: "NOT_INSTALLED",
	}
	if err := s.tmpl.index.Execute(c.Response(), data); err != nil {
		return err
	}
	return nil
}

func (s *AppManagerServer) handleInstalledApps(c echo.Context) error {
	all, err := s.r.GetAll()
	if err != nil {
		return err
	}
	resp := make([]app, 0)
	for _, a := range all {
		instances, err := s.m.FindAllAppInstances(a.Slug())
		if err != nil {
			return err
		}
		if len(instances) != 0 {
			resp = append(resp, app{a.Name(), a.Icon(), a.Description(), a.Slug(), instances})
		}
	}
	data := PageData{
		Apps:        resp,
		CurrentPage: "INSTALLED",
	}
	if err := s.tmpl.index.Execute(c.Response(), data); err != nil {
		return err
	}
	return nil
}

type appPageData struct {
	App               installer.EnvApp
	Instance          *installer.AppInstanceConfig
	Instances         []installer.AppInstanceConfig
	AvailableNetworks []installer.Network
	Task              tasks.Task
	CurrentPage       string
}

func (s *AppManagerServer) handleAppUI(c echo.Context) error {
	global, err := s.m.Config()
	if err != nil {
		return err
	}
	slug := c.Param("slug")
	a, err := installer.FindEnvApp(s.r, slug)
	if err != nil {
		return err
	}
	instances, err := s.m.FindAllAppInstances(slug)
	if err != nil {
		return err
	}
	data := appPageData{
		App:               a,
		Instances:         instances,
		AvailableNetworks: installer.CreateNetworks(global),
		CurrentPage:       a.Name(),
	}
	return s.tmpl.app.Execute(c.Response(), data)
}

func (s *AppManagerServer) handleInstanceUI(c echo.Context) error {
	global, err := s.m.Config()
	if err != nil {
		return err
	}
	slug := c.Param("slug")
	instance, err := s.m.FindInstance(slug)
	if err != nil {
		return err
	}
	a, err := installer.FindEnvApp(s.r, instance.AppId)
	if err != nil {
		return err
	}
	instances, err := s.m.FindAllAppInstances(a.Slug())
	if err != nil {
		return err
	}
	t := s.tasks[slug]
	data := appPageData{
		App:               a,
		Instance:          instance,
		Instances:         instances,
		AvailableNetworks: installer.CreateNetworks(global),
		Task:              t,
		CurrentPage:       instance.Id,
	}
	return s.tmpl.app.Execute(c.Response(), data)
}
