package welcome

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
)

//go:embed dodo-app-tmpl/*
var dodoAppTmplFS embed.FS

const (
	ConfigRepoName = "config"
	namespacesFile = "/namespaces.json"
	loginPath      = "/login"
	logoutPath     = "/logout"
	sessionCookie  = "dodo-app-session"
	userCtx        = "user"
)

type dodoAppTmplts struct {
	index *template.Template
}

func parseTemplatesDodoApp(fs embed.FS) (dodoAppTmplts, error) {
	index, err := template.New("index.html").ParseFS(fs, "dodo-app-tmpl/index.html")
	if err != nil {
		return dodoAppTmplts{}, err
	}
	return dodoAppTmplts{index}, nil
}

type DodoAppServer struct {
	l                 sync.Locker
	st                Store
	port              int
	apiPort           int
	self              string
	sshKey            string
	gitRepoPublicKey  string
	client            soft.Client
	namespace         string
	envAppManagerAddr string
	networks          []string
	env               installer.EnvConfig
	nsc               installer.NamespaceCreator
	jc                installer.JobCreator
	workers           map[string]map[string]struct{}
	appNs             map[string]string
	sc                *securecookie.SecureCookie
	tmplts            dodoAppTmplts
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
	envAppManagerAddr string,
	networks []string,
	nsc installer.NamespaceCreator,
	jc installer.JobCreator,
	env installer.EnvConfig,
) (*DodoAppServer, error) {
	tmplts, err := parseTemplatesDodoApp(dodoAppTmplFS)
	if err != nil {
		return nil, err
	}
	sc := securecookie.New(
		securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32),
	)
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
		envAppManagerAddr,
		networks,
		env,
		nsc,
		jc,
		map[string]map[string]struct{}{},
		map[string]string{},
		sc,
		tmplts,
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
		r.Use(s.mwAuth)
		r.HandleFunc(logoutPath, s.handleLogout).Methods(http.MethodGet)
		r.HandleFunc("/{app-name}"+loginPath, s.handleLoginForm).Methods(http.MethodGet)
		r.HandleFunc("/{app-name}"+loginPath, s.handleLogin).Methods(http.MethodPost)
		r.HandleFunc("/{app-name}", s.handleAppStatus).Methods(http.MethodGet)
		r.HandleFunc("/", s.handleStatus).Methods(http.MethodGet)
		e <- http.ListenAndServe(fmt.Sprintf(":%d", s.port), r)
	}()
	go func() {
		r := mux.NewRouter()
		r.HandleFunc("/update", s.handleApiUpdate)
		r.HandleFunc("/api/apps/{app-name}/workers", s.handleApiRegisterWorker).Methods(http.MethodPost)
		r.HandleFunc("/api/apps", s.handleApiCreateApp).Methods(http.MethodPost)
		r.HandleFunc("/api/add-admin-key", s.handleApiAddAdminKey).Methods(http.MethodPost)
		e <- http.ListenAndServe(fmt.Sprintf(":%d", s.apiPort), r)
	}()
	return <-e
}

func (s *DodoAppServer) mwAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, loginPath) || strings.HasPrefix(r.URL.Path, logoutPath) {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			vars := mux.Vars(r)
			appName, ok := vars["app-name"]
			if !ok || appName == "" {
				http.Error(w, "missing app-name", http.StatusBadRequest)
				return
			}
			http.Redirect(w, r, fmt.Sprintf("/%s%s", appName, loginPath), http.StatusSeeOther)
			return
		}
		var user string
		if err := s.sc.Decode(sessionCookie, cookie.Value, &user); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userCtx, user)))
	})
}

func (s *DodoAppServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *DodoAppServer) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appName, ok := vars["app-name"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, `
<!DOCTYPE html>
<html lang='en'>
	<head>
		<title>dodo: app - login</title>
		<meta charset='utf-8'>
	</head>
	<body>
        <form action="" method="POST">
          <input type="password" placeholder="Password" name="password" required />
          <button type="submit">Login</button>
        </form>
	</body>
</html>
`)
}

func (s *DodoAppServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appName, ok := vars["app-name"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	password := r.FormValue("password")
	if password == "" {
		http.Error(w, "missing password", http.StatusBadRequest)
		return
	}
	user, err := s.st.GetAppOwner(appName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hashed, err := s.st.GetUserPassword(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := bcrypt.CompareHashAndPassword(hashed, []byte(password)); err != nil {
		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return
	}
	if encoded, err := s.sc.Encode(sessionCookie, user); err == nil {
		cookie := &http.Cookie{
			Name:     sessionCookie,
			Value:    encoded,
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
	}
	http.Redirect(w, r, fmt.Sprintf("/%s", appName), http.StatusSeeOther)
}

type statusData struct {
	Apps []string
}

func (s *DodoAppServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userCtx)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	apps, err := s.st.GetUserApps(user.(string))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := statusData{apps}
	if err := s.tmplts.index.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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

type apiUpdateReq struct {
	Ref        string `json:"ref"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	After string `json:"after"`
}

func (s *DodoAppServer) handleApiUpdate(w http.ResponseWriter, r *http.Request) {
	fmt.Println("update")
	var req apiUpdateReq
	var contents strings.Builder
	io.Copy(&contents, r.Body)
	c := contents.String()
	fmt.Println(c)
	if err := json.NewDecoder(strings.NewReader(c)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Ref != "refs/heads/master" || req.Repository.Name == ConfigRepoName {
		return
	}
	// TODO(gio): Create commit record on app init as well
	go func() {
		networks, err := s.getNetworks()
		if err != nil {
			return
		}
		if err := s.updateDodoApp(req.Repository.Name, s.appNs[req.Repository.Name], networks); err != nil {
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

type apiRegisterWorkerReq struct {
	Address string `json:"address"`
}

func (s *DodoAppServer) handleApiRegisterWorker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appName, ok := vars["app-name"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	var req apiRegisterWorkerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, ok := s.workers[appName]; !ok {
		s.workers[appName] = map[string]struct{}{}
	}
	s.workers[appName][req.Address] = struct{}{}
}

type apiCreateAppReq struct {
	AdminPublicKey string `json:"adminPublicKey"`
	NetworkName    string `json:"networkName"`
}

type apiCreateAppResp struct {
	AppName  string `json:"appName"`
	Password string `json:"password"`
}

func (s *DodoAppServer) handleApiCreateApp(w http.ResponseWriter, r *http.Request) {
	var req apiCreateAppReq
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
	password, err := s.CreateApp(appName, req.AdminPublicKey, req.NetworkName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := apiCreateAppResp{
		AppName:  appName,
		Password: password,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *DodoAppServer) CreateApp(appName, adminPublicKey, networkName string) (string, error) {
	s.l.Lock()
	defer s.l.Unlock()
	fmt.Printf("Creating app: %s\n", appName)
	if ok, err := s.client.RepoExists(appName); err != nil {
		return "", err
	} else if ok {
		return "", nil
	}
	user, err := s.client.FindUser(adminPublicKey)
	if err != nil {
		return "", err
	}
	if user == "" {
		user = appName
		if err := s.client.AddUser(user, adminPublicKey); err != nil {
			return "", err
		}
	}
	password := generatePassword()
	// TODO(gio): take admin password for initial application as input
	if appName == "app" {
		password = "app"
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	if err := s.st.CreateUser(user, hashed); err != nil {
		if !errors.Is(err, ErrorAlreadyExists) {
			return "", err
		} else {
			password = ""
		}
	}
	if err := s.st.CreateApp(appName, user); err != nil {
		return "", err
	}
	if err := s.client.AddRepository(appName); err != nil {
		return "", err
	}
	appRepo, err := s.client.GetRepo(appName)
	if err != nil {
		return "", err
	}
	if err := InitRepo(appRepo, networkName); err != nil {
		return "", err
	}
	apps := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	app, err := installer.FindEnvApp(apps, "dodo-app-instance")
	if err != nil {
		return "", err
	}
	suffixGen := installer.NewFixedLengthRandomSuffixGenerator(3)
	suffix, err := suffixGen.Generate()
	if err != nil {
		return "", err
	}
	namespace := fmt.Sprintf("%s%s%s", s.env.NamespacePrefix, app.Namespace(), suffix)
	s.appNs[appName] = namespace
	networks, err := s.getNetworks()
	if err != nil {
		return "", err
	}
	if err := s.updateDodoApp(appName, namespace, networks); err != nil {
		return "", err
	}
	repo, err := s.client.GetRepo(ConfigRepoName)
	if err != nil {
		return "", err
	}
	hf := installer.NewGitHelmFetcher()
	m, err := installer.NewAppManager(repo, s.nsc, s.jc, hf, "/")
	if err != nil {
		return "", err
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
			installer.WithNoNetworks(),
			installer.WithNoPublish(),
			installer.WithNoLock(),
		); err != nil {
			return "", err
		}
		return fmt.Sprintf("Installed app: %s", appName), nil
	}); err != nil {
		return "", err
	}
	cfg, err := m.FindInstance(appName)
	if err != nil {
		return "", err
	}
	fluxKeys, ok := cfg.Input["fluxKeys"]
	if !ok {
		return "", fmt.Errorf("Fluxcd keys not found")
	}
	fluxPublicKey, ok := fluxKeys.(map[string]any)["public"]
	if !ok {
		return "", fmt.Errorf("Fluxcd keys not found")
	}
	if ok, err := s.client.UserExists("fluxcd"); err != nil {
		return "", err
	} else if ok {
		if err := s.client.AddPublicKey("fluxcd", fluxPublicKey.(string)); err != nil {
			return "", err
		}
	} else {
		if err := s.client.AddUser("fluxcd", fluxPublicKey.(string)); err != nil {
			return "", err
		}
	}
	if err := s.client.AddReadOnlyCollaborator(appName, "fluxcd"); err != nil {
		return "", err
	}
	if err := s.client.AddWebhook(appName, fmt.Sprintf("http://%s/update", s.self), "--active=true", "--events=push", "--content-type=json"); err != nil {
		return "", err
	}
	if err := s.client.AddReadWriteCollaborator(appName, user); err != nil {
		return "", err
	}
	return password, nil
}

type apiAddAdminKeyReq struct {
	Public string `json:"public"`
}

func (s *DodoAppServer) handleApiAddAdminKey(w http.ResponseWriter, r *http.Request) {
	var req apiAddAdminKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.client.AddPublicKey("admin", req.Public); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *DodoAppServer) updateDodoApp(name, namespace string, networks []installer.Network) error {
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
		installer.WithNetworks(networks),
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
		network: "%s"
		subdomain: "testapp"
		auth: enabled: false
	}
}
`

func InitRepo(repo soft.RepoIO, networkName string) error {
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
			fmt.Fprintf(w, appCue, networkName)
		}
		return "go web app template", nil
	})
}

func generatePassword() string {
	return "foo"
}

func (s *DodoAppServer) getNetworks() ([]installer.Network, error) {
	addr := fmt.Sprintf("%s/api/networks", s.envAppManagerAddr)
	resp, err := http.Get(addr)
	if err != nil {
		return nil, err
	}
	networks := []installer.Network{}
	if json.NewDecoder(resp.Body).Decode(&networks); err != nil {
		return nil, err
	}
	if len(s.networks) == 0 {
		return networks, nil
	}
	ret := []installer.Network{}
	for _, n := range networks {
		if slices.Contains(s.networks, n.Name) {
			ret = append(ret, n)
		}
	}
	return ret, nil
}
