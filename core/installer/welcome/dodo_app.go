package welcome

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"

	"github.com/gorilla/mux"
)

const (
	ConfigRepoName = "config"
	namespacesFile = "/namespaces.json"
)

type DodoAppServer struct {
	l                sync.Locker
	st               Store
	port             int
	apiPort          int
	self             string
	sshKey           string
	gitRepoPublicKey string
	client           soft.Client
	namespace        string
	env              installer.EnvConfig
	nsc              installer.NamespaceCreator
	jc               installer.JobCreator
	workers          map[string]map[string]struct{}
	appNs            map[string]string
}

// TODO(gio): Initialize appNs on startup
func NewDodoAppServer(
	st Store,
	port int,
	apiPort int,
	self string,
	sshKey string,
	gitRepoPublicKey string,
	client soft.Client,
	namespace string,
	nsc installer.NamespaceCreator,
	jc installer.JobCreator,
	env installer.EnvConfig,
) (*DodoAppServer, error) {
	s := &DodoAppServer{
		&sync.Mutex{},
		st,
		port,
		apiPort,
		self,
		sshKey,
		gitRepoPublicKey,
		client,
		namespace,
		env,
		nsc,
		jc,
		map[string]map[string]struct{}{},
		map[string]string{},
	}
	config, err := client.GetRepo(ConfigRepoName)
	if err != nil {
		return nil, err
	}
	r, err := config.Reader(namespacesFile)
	if err == nil {
		defer r.Close()
		if err := json.NewDecoder(r).Decode(&s.appNs); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (s *DodoAppServer) Start() error {
	e := make(chan error)
	go func() {
		r := mux.NewRouter()
		r.HandleFunc("/status/{app-name}", s.handleAppStatus).Methods(http.MethodGet)
		r.HandleFunc("/status", s.handleStatus).Methods(http.MethodGet)
		e <- http.ListenAndServe(fmt.Sprintf(":%d", s.port), r)
	}()
	go func() {
		r := mux.NewRouter()
		r.HandleFunc("/update", s.handleUpdate)
		r.HandleFunc("/api/apps/{app-name}/workers", s.handleRegisterWorker).Methods(http.MethodPost)
		r.HandleFunc("/api/apps", s.handleCreateApp).Methods(http.MethodPost)
		r.HandleFunc("/api/add-admin-key", s.handleAddAdminKey).Methods(http.MethodPost)
		e <- http.ListenAndServe(fmt.Sprintf(":%d", s.apiPort), r)
	}()
	return <-e
}

func (s *DodoAppServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	apps, err := s.st.GetApps()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, a := range apps {
		fmt.Fprintf(w, "%s\n", a)
	}
}

func (s *DodoAppServer) handleAppStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appName, ok := vars["app-name"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	commits, err := s.st.GetCommitHistory(appName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, c := range commits {
		fmt.Fprintf(w, "%s %s\n", c.Hash, c.Message)
	}
}

type updateReq struct {
	Ref        string `json:"ref"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	After string `json:"after"`
}

func (s *DodoAppServer) handleUpdate(w http.ResponseWriter, r *http.Request) {
	fmt.Println("update")
	var req updateReq
	var contents strings.Builder
	io.Copy(&contents, r.Body)
	c := contents.String()
	fmt.Println(c)
	if err := json.NewDecoder(strings.NewReader(c)).Decode(&req); err != nil {
		fmt.Println(err)
		return
	}
	if req.Ref != "refs/heads/master" || req.Repository.Name == ConfigRepoName {
		return
	}
	// TODO(gio): Create commit record on app init as well
	go func() {
		if err := s.updateDodoApp(req.Repository.Name, s.appNs[req.Repository.Name]); err != nil {
			if err := s.st.CreateCommit(req.Repository.Name, req.After, err.Error()); err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				return
			}
		}
		if err := s.st.CreateCommit(req.Repository.Name, req.After, "OK"); err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
		for addr, _ := range s.workers[req.Repository.Name] {
			go func() {
				// TODO(gio): make port configurable
				http.Get(fmt.Sprintf("http://%s/update", addr))
			}()
		}
	}()
}

type registerWorkerReq struct {
	Address string `json:"address"`
}

func (s *DodoAppServer) handleRegisterWorker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appName, ok := vars["app-name"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	var req registerWorkerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, ok := s.workers[appName]; !ok {
		s.workers[appName] = map[string]struct{}{}
	}
	s.workers[appName][req.Address] = struct{}{}
}

type createAppReq struct {
	AdminPublicKey string `json:"adminPublicKey"`
}

type createAppResp struct {
	AppName string `json:"appName"`
}

func (s *DodoAppServer) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	var req createAppReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	g := installer.NewFixedLengthRandomNameGenerator(3)
	appName, err := g.Generate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.CreateApp(appName, req.AdminPublicKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := createAppResp{appName}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *DodoAppServer) CreateApp(appName, adminPublicKey string) error {
	s.l.Lock()
	defer s.l.Unlock()
	fmt.Printf("Creating app: %s\n", appName)
	if ok, err := s.client.RepoExists(appName); err != nil {
		return err
	} else if ok {
		return nil
	}
	if err := s.st.CreateApp(appName); err != nil {
		return err
	}
	if err := s.client.AddRepository(appName); err != nil {
		return err
	}
	appRepo, err := s.client.GetRepo(appName)
	if err != nil {
		return err
	}
	if err := InitRepo(appRepo); err != nil {
		return err
	}
	apps := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	app, err := installer.FindEnvApp(apps, "dodo-app-instance")
	if err != nil {
		return err
	}
	suffixGen := installer.NewFixedLengthRandomSuffixGenerator(3)
	suffix, err := suffixGen.Generate()
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s%s%s", s.env.NamespacePrefix, app.Namespace(), suffix)
	s.appNs[appName] = namespace
	if err := s.updateDodoApp(appName, namespace); err != nil {
		return err
	}
	repo, err := s.client.GetRepo(ConfigRepoName)
	if err != nil {
		return err
	}
	hf := installer.NewGitHelmFetcher()
	m, err := installer.NewAppManager(repo, s.nsc, s.jc, hf, "/")
	if err != nil {
		return err
	}
	if err := repo.Do(func(fs soft.RepoFS) (string, error) {
		w, err := fs.Writer(namespacesFile)
		if err != nil {
			return "", err
		}
		defer w.Close()
		if err := json.NewEncoder(w).Encode(s.appNs); err != nil {
			return "", err
		}
		if _, err := m.Install(
			app,
			appName,
			"/"+appName,
			namespace,
			map[string]any{
				"repoAddr":         s.client.GetRepoAddress(appName),
				"repoHost":         strings.Split(s.client.Address(), ":")[0],
				"gitRepoPublicKey": s.gitRepoPublicKey,
			},
			installer.WithConfig(&s.env),
			installer.WithNoPublish(),
			installer.WithNoLock(),
		); err != nil {
			return "", err
		}
		return fmt.Sprintf("Installed app: %s", appName), nil
	}); err != nil {
		return err
	}
	cfg, err := m.FindInstance(appName)
	if err != nil {
		return err
	}
	fluxKeys, ok := cfg.Input["fluxKeys"]
	if !ok {
		return fmt.Errorf("Fluxcd keys not found")
	}
	fluxPublicKey, ok := fluxKeys.(map[string]any)["public"]
	if !ok {
		return fmt.Errorf("Fluxcd keys not found")
	}
	if ok, err := s.client.UserExists("fluxcd"); err != nil {
		return err
	} else if ok {
		if err := s.client.AddPublicKey("fluxcd", fluxPublicKey.(string)); err != nil {
			return err
		}
	} else {
		if err := s.client.AddUser("fluxcd", fluxPublicKey.(string)); err != nil {
			return err
		}
	}
	if err := s.client.AddReadOnlyCollaborator(appName, "fluxcd"); err != nil {
		return err
	}
	if err := s.client.AddWebhook(appName, fmt.Sprintf("http://%s/update", s.self), "--active=true", "--events=push", "--content-type=json"); err != nil {
		return err
	}
	if user, err := s.client.FindUser(adminPublicKey); err != nil {
		return err
	} else if user != "" {
		if err := s.client.AddReadWriteCollaborator(appName, user); err != nil {
			return err
		}
	} else {
		if err := s.client.AddUser(appName, adminPublicKey); err != nil {
			return err
		}
		if err := s.client.AddReadWriteCollaborator(appName, appName); err != nil {
			return err
		}
	}
	return nil
}

type addAdminKeyReq struct {
	Public string `json:"public"`
}

func (s *DodoAppServer) handleAddAdminKey(w http.ResponseWriter, r *http.Request) {
	var req addAdminKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.client.AddPublicKey("admin", req.Public); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *DodoAppServer) updateDodoApp(name, namespace string) error {
	repo, err := s.client.GetRepo(name)
	if err != nil {
		return err
	}
	hf := installer.NewGitHelmFetcher()
	m, err := installer.NewAppManager(repo, s.nsc, s.jc, hf, "/.dodo")
	if err != nil {
		return err
	}
	appCfg, err := soft.ReadFile(repo, "app.cue")
	if err != nil {
		return err
	}
	app, err := installer.NewDodoApp(appCfg)
	if err != nil {
		return err
	}
	lg := installer.GitRepositoryLocalChartGenerator{"app", namespace}
	if _, err := m.Install(
		app,
		"app",
		"/.dodo/app",
		namespace,
		map[string]any{
			"repoAddr":      repo.FullAddress(),
			"managerAddr":   fmt.Sprintf("http://%s", s.self),
			"appId":         name,
			"sshPrivateKey": s.sshKey,
		},
		installer.WithConfig(&s.env),
		installer.WithLocalChartGenerator(lg),
		installer.WithBranch("dodo"),
		installer.WithForce(),
	); err != nil {
		return err
	}
	return nil
}

const goMod = `module dodo.app

go 1.18
`

const mainGo = `package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var port = flag.Int("port", 8080, "Port to listen on")

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from Dodo App!")
}

func main() {
	flag.Parse()
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
`

const appCue = `app: {
	type: "golang:1.22.0"
	run: "main.go"
	ingress: {
		network: "Private" // or Public
		subdomain: "testapp"
		auth: enabled: false
	}
}
`

func InitRepo(repo soft.RepoIO) error {
	return repo.Do(func(fs soft.RepoFS) (string, error) {
		{
			w, err := fs.Writer("go.mod")
			if err != nil {
				return "", err
			}
			defer w.Close()
			fmt.Fprint(w, goMod)
		}
		{
			w, err := fs.Writer("main.go")
			if err != nil {
				return "", err
			}
			defer w.Close()
			fmt.Fprintf(w, "%s", mainGo)
		}
		{
			w, err := fs.Writer("app.cue")
			if err != nil {
				return "", err
			}
			defer w.Close()
			fmt.Fprint(w, appCue)
		}
		return "go web app template", nil
	})
}
