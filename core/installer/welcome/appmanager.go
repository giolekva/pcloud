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
	"github.com/gorilla/mux"

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
	ta         map[string]installer.EnvApp
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
		ta:         make(map[string]installer.EnvApp),
		tmpl:       tmpl,
	}, nil
}

type cachingHandler struct {
	h http.Handler
}

func (h cachingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=604800")
	h.h.ServeHTTP(w, r)
}

func (s *AppManagerServer) Start() error {
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(cachingHandler{http.FileServer(http.FS(staticAssets))})
	r.HandleFunc("/api/networks", s.handleNetworks).Methods(http.MethodGet)
	r.HandleFunc("/api/app-repo", s.handleAppRepo)
	r.HandleFunc("/api/app/{slug}/install", s.handleAppInstall).Methods(http.MethodPost)
	r.HandleFunc("/api/app/{slug}", s.handleApp).Methods(http.MethodGet)
	r.HandleFunc("/api/instance/{slug}", s.handleInstance).Methods(http.MethodGet)
	r.HandleFunc("/api/instance/{slug}/update", s.handleAppUpdate).Methods(http.MethodPost)
	r.HandleFunc("/api/instance/{slug}/remove", s.handleAppRemove).Methods(http.MethodPost)
	r.HandleFunc("/app/{slug}", s.handleAppUI).Methods(http.MethodGet)
	r.HandleFunc("/instance/{slug}", s.handleInstanceUI).Methods(http.MethodGet)
	r.HandleFunc("/{pageType}", s.handleAppsList).Methods(http.MethodGet)
	r.HandleFunc("/", s.handleAppsList).Methods(http.MethodGet)
	fmt.Printf("Starting HTTP server on port: %d\n", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), r)
}

type app struct {
	Name             string                        `json:"name"`
	Icon             template.HTML                 `json:"icon"`
	ShortDescription string                        `json:"shortDescription"`
	Slug             string                        `json:"slug"`
	Instances        []installer.AppInstanceConfig `json:"instances,omitempty"`
}

func (s *AppManagerServer) handleNetworks(w http.ResponseWriter, r *http.Request) {
	env, err := s.m.Config()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	networks, err := s.m.CreateNetworks(env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(networks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *AppManagerServer) handleAppRepo(w http.ResponseWriter, r *http.Request) {
	all, err := s.r.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := make([]app, len(all))
	for i, a := range all {
		resp[i] = app{a.Name(), a.Icon(), a.Description(), a.Slug(), nil}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *AppManagerServer) handleApp(w http.ResponseWriter, r *http.Request) {
	slug, ok := mux.Vars(r)["slug"]
	if !ok {
		http.Error(w, "empty slug", http.StatusBadRequest)
		return
	}
	a, err := s.r.Find(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	instances, err := s.m.FindAllAppInstances(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := app{a.Name(), a.Icon(), a.Description(), a.Slug(), instances}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *AppManagerServer) handleInstance(w http.ResponseWriter, r *http.Request) {
	slug, ok := mux.Vars(r)["slug"]
	if !ok {
		http.Error(w, "empty slug", http.StatusBadRequest)
		return
	}
	instance, err := s.m.FindInstance(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a, err := s.r.Find(instance.AppId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := app{a.Name(), a.Icon(), a.Description(), a.Slug(), []installer.AppInstanceConfig{*instance}}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *AppManagerServer) handleAppInstall(w http.ResponseWriter, r *http.Request) {
	slug, ok := mux.Vars(r)["slug"]
	if !ok {
		http.Error(w, "empty slug", http.StatusBadRequest)
		return
	}
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var values map[string]any
	if err := json.Unmarshal(contents, &values); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Values: %+v\n", values)
	a, err := installer.FindEnvApp(s.r, slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Found application: %s\n", slug)
	env, err := s.m.Config()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Configuration: %+v\n", env)
	suffixGen := installer.NewFixedLengthRandomSuffixGenerator(3)
	suffix, err := suffixGen.Generate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	instanceId := a.Slug() + suffix
	appDir := fmt.Sprintf("/apps/%s", instanceId)
	namespace := fmt.Sprintf("%s%s%s", env.NamespacePrefix, a.Namespace(), suffix)
	t := tasks.NewInstallTask(s.h, func() (installer.ReleaseResources, error) {
		return s.m.Install(a, instanceId, appDir, namespace, values)
	})
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	go s.reconciler.Reconcile(ctx)
	if _, ok := s.tasks[instanceId]; ok {
		panic("MUST NOT REACH!")
	}
	s.tasks[instanceId] = t
	s.ta[instanceId] = a
	t.OnDone(func(err error) {
		delete(s.tasks, instanceId)
		delete(s.ta, instanceId)
	})
	go t.Start()
	if _, err := fmt.Fprintf(w, "/instance/%s", instanceId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *AppManagerServer) handleAppUpdate(w http.ResponseWriter, r *http.Request) {
	slug, ok := mux.Vars(r)["slug"]
	if !ok {
		http.Error(w, "empty slug", http.StatusBadRequest)
		return
	}
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var values map[string]any
	if err := json.Unmarshal(contents, &values); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, ok := s.tasks[slug]; ok {
		http.Error(w, "Update already in progress", http.StatusBadRequest)
		return
	}
	rr, err := s.m.Update(slug, values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	go s.reconciler.Reconcile(ctx)
	t := tasks.NewMonitorRelease(s.h, rr)
	t.OnDone(func(err error) {
		delete(s.tasks, slug)
	})
	s.tasks[slug] = t
	go t.Start()
	if _, err := fmt.Fprintf(w, "/instance/%s", slug); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *AppManagerServer) handleAppRemove(w http.ResponseWriter, r *http.Request) {
	slug, ok := mux.Vars(r)["slug"]
	if !ok {
		http.Error(w, "empty slug", http.StatusBadRequest)
		return
	}
	if err := s.m.Remove(slug); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	go s.reconciler.Reconcile(ctx)
	if _, err := fmt.Fprint(w, "/"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type PageData struct {
	Apps         []app
	CurrentPage  string
	SearchTarget string
	SearchValue  string
}

func (s *AppManagerServer) handleAppsList(w http.ResponseWriter, r *http.Request) {
	pageType := mux.Vars(r)["pageType"]
	if pageType == "" {
		pageType = "all"
	}
	searchQuery := r.FormValue("query")
	apps, err := s.r.Filter(searchQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := make([]app, 0)
	for _, a := range apps {
		instances, err := s.m.FindAllAppInstances(a.Slug())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		switch pageType {
		case "installed":
			if len(instances) != 0 {
				resp = append(resp, app{a.Name(), a.Icon(), a.Description(), a.Slug(), instances})
			}
		case "not-installed":
			if len(instances) == 0 {
				resp = append(resp, app{a.Name(), a.Icon(), a.Description(), a.Slug(), nil})
			}
		default:
			resp = append(resp, app{a.Name(), a.Icon(), a.Description(), a.Slug(), instances})
		}
	}
	data := PageData{
		Apps:         resp,
		CurrentPage:  pageType,
		SearchTarget: pageType,
		SearchValue:  searchQuery,
	}
	if err := s.tmpl.index.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type appPageData struct {
	App               installer.EnvApp
	Instance          *installer.AppInstanceConfig
	Instances         []installer.AppInstanceConfig
	AvailableNetworks []installer.Network
	Task              tasks.Task
	CurrentPage       string
}

func (s *AppManagerServer) handleAppUI(w http.ResponseWriter, r *http.Request) {
	global, err := s.m.Config()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slug, ok := mux.Vars(r)["slug"]
	if !ok {
		http.Error(w, "empty slug", http.StatusBadRequest)
		return
	}
	a, err := installer.FindEnvApp(s.r, slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	instances, err := s.m.FindAllAppInstances(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	networks, err := s.m.CreateNetworks(global)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := appPageData{
		App:               a,
		Instances:         instances,
		AvailableNetworks: networks,
		CurrentPage:       a.Name(),
	}
	if err := s.tmpl.app.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *AppManagerServer) handleInstanceUI(w http.ResponseWriter, r *http.Request) {
	global, err := s.m.Config()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slug, ok := mux.Vars(r)["slug"]
	if !ok {
		http.Error(w, "empty slug", http.StatusBadRequest)
		return
	}
	t, ok := s.tasks[slug]
	instance, err := s.m.FindInstance(slug)
	if err != nil && !ok {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var a installer.EnvApp
	if instance != nil {
		a, err = s.m.GetInstanceApp(instance.Id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		var ok bool
		a, ok = s.ta[slug]
		if !ok {
			panic("MUST NOT REACH!")
		}
	}
	instances, err := s.m.FindAllAppInstances(a.Slug())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	networks, err := s.m.CreateNetworks(global)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := appPageData{
		App:               a,
		Instance:          instance,
		Instances:         instances,
		AvailableNetworks: networks,
		Task:              t,
		CurrentPage:       slug,
	}
	if err := s.tmpl.app.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
