package welcome

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/gorilla/mux"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/cluster"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/tasks"
)

//go:embed appmanager-tmpl/*
var appTmpls embed.FS

type taskForward struct {
	task       tasks.Task
	redirectTo string
}

type AppManagerServer struct {
	l            sync.Locker
	port         int
	repo         soft.RepoIO
	m            *installer.AppManager
	r            installer.AppRepository
	fr           installer.AppRepository
	reconciler   *tasks.FixedReconciler
	h            installer.HelmReleaseMonitor
	cnc          installer.ClusterNetworkConfigurator
	vpnAPIClient installer.VPNAPIClient
	tasks        map[string]taskForward
	ta           map[string]installer.EnvApp
	tmpl         tmplts
}

type tmplts struct {
	index       *template.Template
	app         *template.Template
	allClusters *template.Template
	cluster     *template.Template
	task        *template.Template
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
	allClusters, err := parse("appmanager-tmpl/all-clusters.html")
	if err != nil {
		return tmplts{}, err
	}
	cluster, err := parse("appmanager-tmpl/cluster.html")
	if err != nil {
		return tmplts{}, err
	}
	task, err := parse("appmanager-tmpl/task.html")
	if err != nil {
		return tmplts{}, err
	}
	return tmplts{index, app, allClusters, cluster, task}, nil
}

func NewAppManagerServer(
	port int,
	repo soft.RepoIO,
	m *installer.AppManager,
	r installer.AppRepository,
	fr installer.AppRepository,
	reconciler *tasks.FixedReconciler,
	h installer.HelmReleaseMonitor,
	cnc installer.ClusterNetworkConfigurator,
	vpnAPIClient installer.VPNAPIClient,
) (*AppManagerServer, error) {
	tmpl, err := parseTemplatesAppManager(appTmpls)
	if err != nil {
		return nil, err
	}
	return &AppManagerServer{
		l:            &sync.Mutex{},
		port:         port,
		repo:         repo,
		m:            m,
		r:            r,
		fr:           fr,
		reconciler:   reconciler,
		h:            h,
		cnc:          cnc,
		vpnAPIClient: vpnAPIClient,
		tasks:        make(map[string]taskForward),
		ta:           make(map[string]installer.EnvApp),
		tmpl:         tmpl,
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
	r.PathPrefix("/stat/").Handler(cachingHandler{http.FileServer(http.FS(statAssets))})
	r.HandleFunc("/api/networks", s.handleNetworks).Methods(http.MethodGet)
	r.HandleFunc("/api/app-repo", s.handleAppRepo)
	r.HandleFunc("/api/app/{slug}/install", s.handleAppInstall).Methods(http.MethodPost)
	r.HandleFunc("/api/app/{slug}", s.handleApp).Methods(http.MethodGet)
	r.HandleFunc("/api/instance/{slug}", s.handleInstance).Methods(http.MethodGet)
	r.HandleFunc("/api/instance/{slug}/update", s.handleAppUpdate).Methods(http.MethodPost)
	r.HandleFunc("/api/instance/{slug}/remove", s.handleAppRemove).Methods(http.MethodPost)
	r.HandleFunc("/clusters/{cluster}/servers/{server}/remove", s.handleClusterRemoveServer).Methods(http.MethodPost)
	r.HandleFunc("/clusters/{cluster}/servers", s.handleClusterAddServer).Methods(http.MethodPost)
	r.HandleFunc("/clusters/{name}", s.handleCluster).Methods(http.MethodGet)
	r.HandleFunc("/clusters/{name}/remove", s.handleRemoveCluster).Methods(http.MethodPost)
	r.HandleFunc("/clusters", s.handleAllClusters).Methods(http.MethodGet)
	r.HandleFunc("/clusters", s.handleCreateCluster).Methods(http.MethodPost)
	r.HandleFunc("/app/{slug}", s.handleAppUI).Methods(http.MethodGet)
	r.HandleFunc("/instance/{slug}", s.handleInstanceUI).Methods(http.MethodGet)
	r.HandleFunc("/tasks/{slug}", s.handleTaskStatus).Methods(http.MethodGet)
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
	instances, err := s.m.GetAllAppInstances(slug)
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
	instance, err := s.m.GetInstance(slug)
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
	s.l.Lock()
	defer s.l.Unlock()
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
		rr, err := s.m.Install(a, instanceId, appDir, namespace, values)
		if err == nil {
			ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
			go s.reconciler.Reconcile(ctx)
		}
		return rr, err
	})
	if _, ok := s.tasks[instanceId]; ok {
		panic("MUST NOT REACH!")
	}
	s.tasks[instanceId] = taskForward{t, fmt.Sprintf("/instance/%s", instanceId)}
	s.ta[instanceId] = a
	t.OnDone(func(err error) {
		go func() {
			time.Sleep(30 * time.Second)
			s.l.Lock()
			defer s.l.Unlock()
			delete(s.tasks, instanceId)
			delete(s.ta, instanceId)
		}()
	})
	go t.Start()
	if _, err := fmt.Fprintf(w, "/tasks/%s", instanceId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *AppManagerServer) handleAppUpdate(w http.ResponseWriter, r *http.Request) {
	s.l.Lock()
	defer s.l.Unlock()
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
		go func() {
			time.Sleep(30 * time.Second)
			s.l.Lock()
			defer s.l.Unlock()
			delete(s.tasks, slug)
		}()
	})
	s.tasks[slug] = taskForward{t, fmt.Sprintf("/instance/%s", slug)}
	go t.Start()
	if _, err := fmt.Fprintf(w, "/tasks/%s", slug); err != nil {
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
		instances, err := s.m.GetAllAppInstances(a.Slug())
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
	AvailableClusters []cluster.State
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
	instances, err := s.m.GetAllAppInstances(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	networks, err := s.m.CreateNetworks(global)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	clusters, err := s.m.GetClusters()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := appPageData{
		App:               a,
		Instances:         instances,
		AvailableNetworks: networks,
		AvailableClusters: clusters,
		CurrentPage:       a.Name(),
	}
	if err := s.tmpl.app.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *AppManagerServer) handleInstanceUI(w http.ResponseWriter, r *http.Request) {
	s.l.Lock()
	defer s.l.Unlock()
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
	instance, err := s.m.GetInstance(slug)
	if err != nil && !ok {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok && !(t.task.Status() == tasks.StatusDone || t.task.Status() == tasks.StatusFailed) {
		http.Redirect(w, r, fmt.Sprintf("/tasks/%s", slug), http.StatusSeeOther)
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
	instances, err := s.m.GetAllAppInstances(a.Slug())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	networks, err := s.m.CreateNetworks(global)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	clusters, err := s.m.GetClusters()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := appPageData{
		App:               a,
		Instance:          instance,
		Instances:         instances,
		AvailableNetworks: networks,
		AvailableClusters: clusters,
		Task:              t.task,
		CurrentPage:       slug,
	}
	if err := s.tmpl.app.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type taskStatusData struct {
	CurrentPage string
	Task        tasks.Task
}

func (s *AppManagerServer) handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	s.l.Lock()
	defer s.l.Unlock()
	slug, ok := mux.Vars(r)["slug"]
	if !ok {
		http.Error(w, "empty slug", http.StatusBadRequest)
		return
	}
	t, ok := s.tasks[slug]
	if !ok {
		http.Error(w, "task not found", http.StatusInternalServerError)

		return
	}
	if ok && (t.task.Status() == tasks.StatusDone || t.task.Status() == tasks.StatusFailed) {
		http.Redirect(w, r, t.redirectTo, http.StatusSeeOther)
		return
	}
	data := taskStatusData{
		CurrentPage: "",
		Task:        t.task,
	}
	if err := s.tmpl.task.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type clustersData struct {
	CurrentPage string
	Clusters    []cluster.State
}

func (s *AppManagerServer) handleAllClusters(w http.ResponseWriter, r *http.Request) {
	clusters, err := s.m.GetClusters()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := clustersData{
		"clusters",
		clusters,
	}
	if err := s.tmpl.allClusters.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type clusterData struct {
	CurrentPage string
	Cluster     cluster.State
}

func (s *AppManagerServer) handleCluster(w http.ResponseWriter, r *http.Request) {
	name, ok := mux.Vars(r)["name"]
	if !ok {
		http.Error(w, "empty name", http.StatusBadRequest)
		return
	}
	m, err := s.getClusterManager(name)
	if err != nil {
		if errors.Is(err, installer.ErrorNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	data := clusterData{
		"clusters",
		m.State(),
	}
	if err := s.tmpl.cluster.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *AppManagerServer) handleClusterRemoveServer(w http.ResponseWriter, r *http.Request) {
	s.l.Lock()
	defer s.l.Unlock()
	cName, ok := mux.Vars(r)["cluster"]
	if !ok {
		http.Error(w, "empty name", http.StatusBadRequest)
		return
	}
	if _, ok := s.tasks[cName]; ok {
		http.Error(w, "cluster task in progress", http.StatusLocked)
		return
	}
	sName, ok := mux.Vars(r)["server"]
	if !ok {
		http.Error(w, "empty name", http.StatusBadRequest)
		return
	}
	m, err := s.getClusterManager(cName)
	if err != nil {
		if errors.Is(err, installer.ErrorNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	task := tasks.NewClusterRemoveServerTask(m, sName, s.repo)
	task.OnDone(func(err error) {
		go func() {
			time.Sleep(30 * time.Second)
			s.l.Lock()
			defer s.l.Unlock()
			delete(s.tasks, cName)
		}()
	})
	go task.Start()
	s.tasks[cName] = taskForward{task, fmt.Sprintf("/clusters/%s", cName)}
	http.Redirect(w, r, fmt.Sprintf("/tasks/%s", cName), http.StatusSeeOther)
}

func (s *AppManagerServer) getClusterManager(cName string) (cluster.Manager, error) {
	clusters, err := s.m.GetClusters()
	if err != nil {
		return nil, err
	}
	var c *cluster.State
	for _, i := range clusters {
		if i.Name == cName {
			c = &i
			break
		}
	}
	if c == nil {
		return nil, installer.ErrorNotFound
	}
	return cluster.RestoreKubeManager(*c)
}

func (s *AppManagerServer) handleClusterAddServer(w http.ResponseWriter, r *http.Request) {
	s.l.Lock()
	defer s.l.Unlock()
	cName, ok := mux.Vars(r)["cluster"]
	if !ok {
		http.Error(w, "empty name", http.StatusBadRequest)
		return
	}
	if _, ok := s.tasks[cName]; ok {
		http.Error(w, "cluster task in progress", http.StatusLocked)
		return
	}
	m, err := s.getClusterManager(cName)
	if err != nil {
		if errors.Is(err, installer.ErrorNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	t := r.PostFormValue("type")
	ip := net.ParseIP(r.PostFormValue("ip"))
	if ip == nil {
		http.Error(w, "invalid ip", http.StatusBadRequest)
		return
	}
	port := 22
	if p := r.PostFormValue("port"); p != "" {
		port, err = strconv.Atoi(p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	server := cluster.Server{
		IP:       ip,
		Port:     port,
		User:     r.PostFormValue("user"),
		Password: r.PostFormValue("password"),
	}
	var task tasks.Task
	switch strings.ToLower(t) {
	case "controller":
		if len(m.State().Controllers) == 0 {
			task = tasks.NewClusterInitTask(m, server, s.cnc, s.repo, s.setupRemoteCluster())
		} else {
			task = tasks.NewClusterJoinControllerTask(m, server, s.repo)
		}
	case "worker":
		task = tasks.NewClusterJoinWorkerTask(m, server, s.repo)
	default:
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}
	task.OnDone(func(err error) {
		go func() {
			time.Sleep(30 * time.Second)
			s.l.Lock()
			defer s.l.Unlock()
			delete(s.tasks, cName)
		}()
	})
	go task.Start()
	s.tasks[cName] = taskForward{task, fmt.Sprintf("/clusters/%s", cName)}
	http.Redirect(w, r, fmt.Sprintf("/tasks/%s", cName), http.StatusSeeOther)
}

func (s *AppManagerServer) handleCreateCluster(w http.ResponseWriter, r *http.Request) {
	cName := r.PostFormValue("name")
	if cName == "" {
		http.Error(w, "no name", http.StatusBadRequest)
		return
	}
	st := cluster.State{Name: cName}
	if _, err := s.repo.Do(func(fs soft.RepoFS) (string, error) {
		if err := soft.WriteJson(fs, fmt.Sprintf("/clusters/%s/config.json", cName), st); err != nil {
			return "", err
		}
		return fmt.Sprintf("create cluster: %s", cName), nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/clusters/%s", cName), http.StatusSeeOther)
}

func (s *AppManagerServer) handleRemoveCluster(w http.ResponseWriter, r *http.Request) {
	cName, ok := mux.Vars(r)["name"]
	if !ok {
		http.Error(w, "empty name", http.StatusBadRequest)
		return
	}
	if _, ok := s.tasks[cName]; ok {
		http.Error(w, "cluster task in progress", http.StatusLocked)
		return
	}
	m, err := s.getClusterManager(cName)
	if err != nil {
		if errors.Is(err, installer.ErrorNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	task := tasks.NewRemoveClusterTask(m, s.cnc, s.repo)
	task.OnDone(func(err error) {
		go func() {
			time.Sleep(30 * time.Second)
			s.l.Lock()
			defer s.l.Unlock()
			delete(s.tasks, cName)
		}()
	})
	go task.Start()
	s.tasks[cName] = taskForward{task, fmt.Sprintf("/clusters/%s", cName)}
	http.Redirect(w, r, fmt.Sprintf("/tasks/%s", cName), http.StatusSeeOther)
}

func (s *AppManagerServer) setupRemoteCluster() cluster.ClusterSetupFunc {
	const vpnUser = "private-network-proxy"
	return func(name, kubeconfig, ingressClassName string) (net.IP, error) {
		hostname := fmt.Sprintf("cluster-%s", name)
		t := tasks.NewInstallTask(s.h, func() (installer.ReleaseResources, error) {
			app, err := installer.FindEnvApp(s.fr, "cluster-network")
			if err != nil {
				return installer.ReleaseResources{}, err
			}
			env, err := s.m.Config()
			if err != nil {
				return installer.ReleaseResources{}, err
			}
			instanceId := fmt.Sprintf("%s-%s", app.Slug(), name)
			appDir := fmt.Sprintf("/clusters/%s/ingress", name)
			namespace := fmt.Sprintf("%scluster-network-%s", env.NamespacePrefix, name)
			rr, err := s.m.Install(app, instanceId, appDir, namespace, map[string]any{
				"cluster": map[string]any{
					"name":             name,
					"kubeconfig":       kubeconfig,
					"ingressClassName": ingressClassName,
				},
				// TODO(gio): remove hardcoded user
				"vpnUser":          vpnUser,
				"vpnProxyHostname": hostname,
			})
			if err != nil {
				return installer.ReleaseResources{}, err
			}
			ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
			go s.reconciler.Reconcile(ctx)
			return rr, err
		})
		ch := make(chan error)
		t.OnDone(func(err error) {
			ch <- err
		})
		go t.Start()
		err := <-ch
		if err != nil {
			return nil, err
		}
		for {
			ip, err := s.vpnAPIClient.GetNodeIP(vpnUser, hostname)
			if err == nil {
				return ip, nil
			}
			if errors.Is(err, installer.ErrorNotFound) {
				time.Sleep(5 * time.Second)
			}
		}
	}
}
