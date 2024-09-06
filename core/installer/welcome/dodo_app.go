package welcome

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/rand"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/tasks"

	"cuelang.org/go/cue"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
)

//go:embed dodo-app-tmpl/*
var dodoAppTmplFS embed.FS

//go:embed all:app-tmpl
var appTmplsFS embed.FS

const (
	ConfigRepoName = "config"
	appConfigsFile = "/apps.json"
	loginPath      = "/login"
	logoutPath     = "/logout"
	staticPath     = "/stat/"
	apiPublicData  = "/api/public-data"
	apiCreateApp   = "/api/apps"
	sessionCookie  = "dodo-app-session"
	userCtx        = "user"
	initCommitMsg  = "init"
)

type dodoAppTmplts struct {
	index        *template.Template
	appStatus    *template.Template
	commitStatus *template.Template
	logs         *template.Template
}

func parseTemplatesDodoApp(fs embed.FS) (dodoAppTmplts, error) {
	base, err := template.ParseFS(fs, "dodo-app-tmpl/base.html")
	if err != nil {
		return dodoAppTmplts{}, err
	}
	parse := func(path string) (*template.Template, error) {
		if b, err := base.Clone(); err != nil {
			return nil, err
		} else {
			return b.ParseFS(fs, path)
		}
	}
	index, err := parse("dodo-app-tmpl/index.html")
	if err != nil {
		return dodoAppTmplts{}, err
	}
	appStatus, err := parse("dodo-app-tmpl/app_status.html")
	if err != nil {
		return dodoAppTmplts{}, err
	}
	commitStatus, err := parse("dodo-app-tmpl/commit_status.html")
	if err != nil {
		return dodoAppTmplts{}, err
	}
	logs, err := parse("dodo-app-tmpl/logs.html")
	if err != nil {
		return dodoAppTmplts{}, err
	}
	return dodoAppTmplts{index, appStatus, commitStatus, logs}, nil
}

type DodoAppServer struct {
	l                 sync.Locker
	st                Store
	nf                NetworkFilter
	ug                UserGetter
	port              int
	apiPort           int
	self              string
	repoPublicAddr    string
	sshKey            string
	gitRepoPublicKey  string
	client            soft.Client
	namespace         string
	envAppManagerAddr string
	env               installer.EnvConfig
	nsc               installer.NamespaceCreator
	jc                installer.JobCreator
	vpnKeyGen         installer.VPNAPIClient
	workers           map[string]map[string]struct{}
	appConfigs        map[string]appConfig
	tmplts            dodoAppTmplts
	appTmpls          AppTmplStore
	external          bool
	fetchUsersAddr    string
	reconciler        tasks.Reconciler
	logs              map[string]string
}

type appConfig struct {
	Namespace string `json:"namespace"`
	Network   string `json:"network"`
}

// TODO(gio): Initialize appNs on startup
func NewDodoAppServer(
	st Store,
	nf NetworkFilter,
	ug UserGetter,
	port int,
	apiPort int,
	self string,
	repoPublicAddr string,
	sshKey string,
	gitRepoPublicKey string,
	client soft.Client,
	namespace string,
	envAppManagerAddr string,
	nsc installer.NamespaceCreator,
	jc installer.JobCreator,
	vpnKeyGen installer.VPNAPIClient,
	env installer.EnvConfig,
	external bool,
	fetchUsersAddr string,
	reconciler tasks.Reconciler,
) (*DodoAppServer, error) {
	tmplts, err := parseTemplatesDodoApp(dodoAppTmplFS)
	if err != nil {
		return nil, err
	}
	apps, err := fs.Sub(appTmplsFS, "app-tmpl")
	if err != nil {
		return nil, err
	}
	appTmpls, err := NewAppTmplStoreFS(apps)
	if err != nil {
		return nil, err
	}
	s := &DodoAppServer{
		&sync.Mutex{},
		st,
		nf,
		ug,
		port,
		apiPort,
		self,
		repoPublicAddr,
		sshKey,
		gitRepoPublicKey,
		client,
		namespace,
		envAppManagerAddr,
		env,
		nsc,
		jc,
		vpnKeyGen,
		map[string]map[string]struct{}{},
		map[string]appConfig{},
		tmplts,
		appTmpls,
		external,
		fetchUsersAddr,
		reconciler,
		map[string]string{},
	}
	config, err := client.GetRepo(ConfigRepoName)
	if err != nil {
		return nil, err
	}
	r, err := config.Reader(appConfigsFile)
	if err == nil {
		defer r.Close()
		if err := json.NewDecoder(r).Decode(&s.appConfigs); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (s *DodoAppServer) getAppConfig(app, branch string) appConfig {
	return s.appConfigs[fmt.Sprintf("%s-%s", app, branch)]
}

func (s *DodoAppServer) setAppConfig(app, branch string, cfg appConfig) {
	s.appConfigs[fmt.Sprintf("%s-%s", app, branch)] = cfg
}

func (s *DodoAppServer) Start() error {
	// if err := s.client.DisableKeyless(); err != nil {
	// 	return err
	// }
	// if err := s.client.DisableAnonAccess(); err != nil {
	// 	return err
	// }
	e := make(chan error)
	go func() {
		r := mux.NewRouter()
		r.Use(s.mwAuth)
		r.PathPrefix(staticPath).Handler(cachingHandler{http.FileServer(http.FS(statAssets))})
		r.HandleFunc(logoutPath, s.handleLogout).Methods(http.MethodGet)
		r.HandleFunc(apiPublicData, s.handleAPIPublicData)
		r.HandleFunc(apiCreateApp, s.handleAPICreateApp).Methods(http.MethodPost)
		r.HandleFunc("/{app-name}"+loginPath, s.handleLoginForm).Methods(http.MethodGet)
		r.HandleFunc("/{app-name}"+loginPath, s.handleLogin).Methods(http.MethodPost)
		r.HandleFunc("/{app-name}/logs", s.handleAppLogs).Methods(http.MethodGet)
		r.HandleFunc("/{app-name}/{hash}", s.handleAppCommit).Methods(http.MethodGet)
		r.HandleFunc("/{app-name}/dev-branch/create", s.handleCreateDevBranch).Methods(http.MethodPost)
		r.HandleFunc("/{app-name}/branch/{branch}", s.handleAppStatus).Methods(http.MethodGet)
		r.HandleFunc("/{app-name}", s.handleAppStatus).Methods(http.MethodGet)
		r.HandleFunc("/", s.handleStatus).Methods(http.MethodGet)
		r.HandleFunc("/", s.handleCreateApp).Methods(http.MethodPost)
		e <- http.ListenAndServe(fmt.Sprintf(":%d", s.port), r)
	}()
	go func() {
		r := mux.NewRouter()
		r.HandleFunc("/update", s.handleAPIUpdate)
		r.HandleFunc("/api/apps/{app-name}/workers", s.handleAPIRegisterWorker).Methods(http.MethodPost)
		r.HandleFunc("/api/add-public-key", s.handleAPIAddPublicKey).Methods(http.MethodPost)
		r.HandleFunc("/api/apps/{app-name}/branch/{branch}/env-profile", s.handleBranchEnvProfile).Methods(http.MethodGet)
		if !s.external {
			r.HandleFunc("/api/sync-users", s.handleAPISyncUsers).Methods(http.MethodGet)
		}
		e <- http.ListenAndServe(fmt.Sprintf(":%d", s.apiPort), r)
	}()
	if !s.external {
		go func() {
			s.syncUsers()
			for {
				delay := time.Duration(rand.Intn(60)+60) * time.Second
				time.Sleep(delay)
				s.syncUsers()
			}
		}()
	}
	return <-e
}

type UserGetter interface {
	Get(r *http.Request) string
	Encode(w http.ResponseWriter, user string) error
}

type externalUserGetter struct {
	sc *securecookie.SecureCookie
}

func NewExternalUserGetter() UserGetter {
	return &externalUserGetter{securecookie.New(
		securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32),
	)}
}

func (ug *externalUserGetter) Get(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	var user string
	if err := ug.sc.Decode(sessionCookie, cookie.Value, &user); err != nil {
		return ""
	}
	return user
}

func (ug *externalUserGetter) Encode(w http.ResponseWriter, user string) error {
	if encoded, err := ug.sc.Encode(sessionCookie, user); err == nil {
		cookie := &http.Cookie{
			Name:     sessionCookie,
			Value:    encoded,
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		return nil
	} else {
		return err
	}
}

type internalUserGetter struct{}

func NewInternalUserGetter() UserGetter {
	return internalUserGetter{}
}

func (ug internalUserGetter) Get(r *http.Request) string {
	return r.Header.Get("X-User")
}

func (ug internalUserGetter) Encode(w http.ResponseWriter, user string) error {
	return nil
}

func (s *DodoAppServer) mwAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, loginPath) ||
			strings.HasPrefix(r.URL.Path, logoutPath) ||
			strings.HasPrefix(r.URL.Path, staticPath) ||
			strings.HasPrefix(r.URL.Path, apiPublicData) ||
			strings.HasPrefix(r.URL.Path, apiCreateApp) {
			next.ServeHTTP(w, r)
			return
		}
		user := s.ug.Get(r)
		if user == "" {
			vars := mux.Vars(r)
			appName, ok := vars["app-name"]
			if !ok || appName == "" {
				http.Error(w, "missing app-name", http.StatusBadRequest)
				return
			}
			http.Redirect(w, r, fmt.Sprintf("/%s%s", appName, loginPath), http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userCtx, user)))
	})
}

func (s *DodoAppServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	// TODO(gio): move to UserGetter
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
	if err := s.ug.Encode(w, user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/%s", appName), http.StatusSeeOther)
}

type navItem struct {
	Name    string
	Address string
}

type statusData struct {
	Navigation []navItem
	Apps       []string
	Networks   []installer.Network
	Types      []string
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
	networks, err := s.getNetworks(user.(string))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var types []string
	for _, t := range s.appTmpls.Types() {
		types = append(types, strings.Replace(t, "-", ":", 1))
	}
	n := []navItem{navItem{"Home", "/"}}
	data := statusData{n, apps, networks, types}
	if err := s.tmplts.index.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type appStatusData struct {
	Navigation      []navItem
	Name            string
	GitCloneCommand string
	Commits         []CommitMeta
	LastCommit      resourceData
	Branches        []string
}

func (s *DodoAppServer) handleAppStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appName, ok := vars["app-name"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	branch, ok := vars["branch"]
	if !ok || branch == "" {
		branch = "master"
	}
	u := r.Context().Value(userCtx)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, ok := u.(string)
	if !ok {
		http.Error(w, "could not get user", http.StatusInternalServerError)
		return
	}
	owner, err := s.st.GetAppOwner(appName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if owner != user {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	commits, err := s.st.GetCommitHistory(appName, branch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var lastCommitResources resourceData
	if len(commits) > 0 {
		lastCommit, err := s.st.GetCommit(commits[len(commits)-1].Hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r, err := extractResourceData(lastCommit.Resources.Helm)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		lastCommitResources = r
	}
	branches, err := s.st.GetBranches(appName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := appStatusData{
		Navigation: []navItem{
			navItem{"Home", "/"},
			navItem{appName, "/" + appName},
		},
		Name:            appName,
		GitCloneCommand: fmt.Sprintf("git clone %s/%s\n\n\n", s.repoPublicAddr, appName),
		Commits:         commits,
		LastCommit:      lastCommitResources,
		Branches:        branches,
	}
	if branch != "master" {
		data.Navigation = append(data.Navigation, navItem{branch, fmt.Sprintf("/%s/branch/%s", appName, branch)})
	}
	if err := s.tmplts.appStatus.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type appEnv struct {
	Profile string `json:"envProfile"`
}

func (s *DodoAppServer) handleBranchEnvProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appName, ok := vars["app-name"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	branch, ok := vars["branch"]
	if !ok || branch == "" {
		branch = "master"
	}
	info, err := s.st.GetLastCommitInfo(appName, branch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var e appEnv
	if err := json.NewDecoder(bytes.NewReader(info.Resources.RenderedRaw)).Decode(&e); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, e.Profile)
}

type volume struct {
	Name string
	Size string
}

type postgresql struct {
	Name    string
	Version string
	Volume  string
}

type ingress struct {
	Host string
}

type vm struct {
	Name     string
	User     string
	CPUCores int
	Memory   string
}

type resourceData struct {
	Volume         []volume
	PostgreSQL     []postgresql
	Ingress        []ingress
	VirtualMachine []vm
}

type commitStatusData struct {
	Navigation []navItem
	AppName    string
	Commit     Commit
	Resources  resourceData
}

func (s *DodoAppServer) handleAppCommit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appName, ok := vars["app-name"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	hash, ok := vars["hash"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	u := r.Context().Value(userCtx)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, ok := u.(string)
	if !ok {
		http.Error(w, "could not get user", http.StatusInternalServerError)
		return
	}
	owner, err := s.st.GetAppOwner(appName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if owner != user {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	commit, err := s.st.GetCommit(hash)
	if err != nil {
		// TODO(gio): not-found ?
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var res strings.Builder
	if err := json.NewEncoder(&res).Encode(commit.Resources.Helm); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resData, err := extractResourceData(commit.Resources.Helm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := commitStatusData{
		Navigation: []navItem{
			navItem{"Home", "/"},
			navItem{appName, "/" + appName},
			navItem{hash, "/" + appName + "/" + hash},
		},
		AppName:   appName,
		Commit:    commit,
		Resources: resData,
	}
	if err := s.tmplts.commitStatus.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type logData struct {
	Navigation []navItem
	AppName    string
	Logs       template.HTML
}

func (s *DodoAppServer) handleAppLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appName, ok := vars["app-name"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	u := r.Context().Value(userCtx)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, ok := u.(string)
	if !ok {
		http.Error(w, "could not get user", http.StatusInternalServerError)
		return
	}
	owner, err := s.st.GetAppOwner(appName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if owner != user {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	data := logData{
		Navigation: []navItem{
			navItem{"Home", "/"},
			navItem{appName, "/" + appName},
			navItem{"Logs", "/" + appName + "/logs"},
		},
		AppName: appName,
		Logs:    template.HTML(strings.ReplaceAll(s.logs[appName], "\n", "<br/>")),
	}
	if err := s.tmplts.logs.Execute(w, data); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type apiUpdateReq struct {
	Ref        string `json:"ref"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	After   string `json:"after"`
	Commits []struct {
		Id      string `json:"id"`
		Message string `json:"message"`
	} `json:"commits"`
}

func (s *DodoAppServer) handleAPIUpdate(w http.ResponseWriter, r *http.Request) {
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
	if strings.HasPrefix(req.Ref, "refs/heads/dodo_") || req.Repository.Name == ConfigRepoName {
		return
	}
	branch, ok := strings.CutPrefix(req.Ref, "refs/heads/")
	if !ok {
		http.Error(w, "invalid branch", http.StatusBadRequest)
		return
	}
	// TODO(gio): Create commit record on app init as well
	go func() {
		owner, err := s.st.GetAppOwner(req.Repository.Name)
		if err != nil {
			return
		}
		networks, err := s.getNetworks(owner)
		if err != nil {
			return
		}
		apps := installer.NewInMemoryAppRepository(installer.CreateAllApps())
		instanceAppStatus, err := installer.FindEnvApp(apps, "dodo-app-instance-status")
		if err != nil {
			return
		}
		found := false
		commitMsg := ""
		for _, c := range req.Commits {
			if c.Id == req.After {
				found = true
				commitMsg = c.Message
				break
			}
		}
		if !found {
			fmt.Printf("Error: could not find commit message")
			return
		}
		s.l.Lock()
		defer s.l.Unlock()
		resources, err := s.updateDodoApp(instanceAppStatus, req.Repository.Name, branch, s.getAppConfig(req.Repository.Name, branch).Namespace, networks, owner)
		if err = s.createCommit(req.Repository.Name, branch, req.After, commitMsg, err, resources); err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			return
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
	Logs    string `json:"logs"`
}

func (s *DodoAppServer) handleAPIRegisterWorker(w http.ResponseWriter, r *http.Request) {
	// TODO(gio): lock
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
	s.logs[appName] = req.Logs
}

func (s *DodoAppServer) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	u := r.Context().Value(userCtx)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, ok := u.(string)
	if !ok {
		http.Error(w, "could not get user", http.StatusInternalServerError)
		return
	}
	network := r.FormValue("network")
	if network == "" {
		http.Error(w, "missing network", http.StatusBadRequest)
		return
	}
	subdomain := r.FormValue("subdomain")
	if subdomain == "" {
		http.Error(w, "missing subdomain", http.StatusBadRequest)
		return
	}
	appType := r.FormValue("type")
	if appType == "" {
		http.Error(w, "missing type", http.StatusBadRequest)
		return
	}
	g := installer.NewFixedLengthRandomNameGenerator(3)
	appName, err := g.Generate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok, err := s.client.UserExists(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !ok {
		http.Error(w, "user sync has not finished, please try again in few minutes", http.StatusFailedDependency)
		return
	}
	if err := s.st.CreateUser(user, nil, network); err != nil && !errors.Is(err, ErrorAlreadyExists) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.st.CreateApp(appName, user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.createApp(user, appName, appType, network, subdomain); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/%s", appName), http.StatusSeeOther)
}

func (s *DodoAppServer) handleCreateDevBranch(w http.ResponseWriter, r *http.Request) {
	u := r.Context().Value(userCtx)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, ok := u.(string)
	if !ok {
		http.Error(w, "could not get user", http.StatusInternalServerError)
		return
	}
	vars := mux.Vars(r)
	appName, ok := vars["app-name"]
	if !ok || appName == "" {
		http.Error(w, "missing app-name", http.StatusBadRequest)
		return
	}
	branch := r.FormValue("branch")
	if branch == "" {
		http.Error(w, "missing network", http.StatusBadRequest)
		return
	}
	if err := s.createDevBranch(appName, "master", branch, user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/branch/%s", appName, branch), http.StatusSeeOther)
}

type apiCreateAppReq struct {
	AppType        string `json:"type"`
	AdminPublicKey string `json:"adminPublicKey"`
	Network        string `json:"network"`
	Subdomain      string `json:"subdomain"`
}

type apiCreateAppResp struct {
	AppName  string `json:"appName"`
	Password string `json:"password"`
}

func (s *DodoAppServer) handleAPICreateApp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
	user, err := s.client.FindUser(req.AdminPublicKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if user != "" {
		http.Error(w, "public key already registered", http.StatusBadRequest)
		return
	}
	user = appName
	if err := s.client.AddUser(user, req.AdminPublicKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	password := generatePassword()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.st.CreateUser(user, hashed, req.Network); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.st.CreateApp(appName, user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.createApp(user, appName, req.AppType, req.Network, req.Subdomain); err != nil {
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

func (s *DodoAppServer) isNetworkUseAllowed(network string) bool {
	if !s.external {
		return true
	}
	for _, cfg := range s.appConfigs {
		if strings.ToLower(cfg.Network) == network {
			return false
		}
	}
	return true
}

func (s *DodoAppServer) createApp(user, appName, appType, network, subdomain string) error {
	s.l.Lock()
	defer s.l.Unlock()
	fmt.Printf("Creating app: %s\n", appName)
	network = strings.ToLower(network)
	if !s.isNetworkUseAllowed(network) {
		return fmt.Errorf("network already used: %s", network)
	}
	if ok, err := s.client.RepoExists(appName); err != nil {
		return err
	} else if ok {
		return nil
	}
	networks, err := s.getNetworks(user)
	if err != nil {
		return err
	}
	n, ok := installer.NetworkMap(networks)[network]
	if !ok {
		return fmt.Errorf("network not found: %s\n", network)
	}
	if err := s.client.AddRepository(appName); err != nil {
		return err
	}
	appRepo, err := s.client.GetRepo(appName)
	if err != nil {
		return err
	}
	files, err := s.renderAppConfigTemplate(appType, n, subdomain)
	if err != nil {
		return err
	}
	return s.createAppForBranch(appRepo, appName, "master", user, network, files)
}

func (s *DodoAppServer) createDevBranch(appName, fromBranch, toBranch, user string) error {
	s.l.Lock()
	defer s.l.Unlock()
	fmt.Printf("Creating dev branch app: %s %s %s\n", appName, fromBranch, toBranch)
	appRepo, err := s.client.GetRepoBranch(appName, fromBranch)
	if err != nil {
		return err
	}
	appCfg, err := soft.ReadFile(appRepo, "app.cue")
	if err != nil {
		return err
	}
	network, branchCfg, err := createDevBranchAppConfig(appCfg, toBranch, user)
	if err != nil {
		return err
	}
	return s.createAppForBranch(appRepo, appName, toBranch, user, network, map[string][]byte{"app.cue": branchCfg})
}

func (s *DodoAppServer) createAppForBranch(
	repo soft.RepoIO,
	appName string,
	branch string,
	user string,
	network string,
	files map[string][]byte,
) error {
	commit, err := repo.Do(func(fs soft.RepoFS) (string, error) {
		for path, contents := range files {
			if err := soft.WriteFile(fs, path, string(contents)); err != nil {
				return "", err
			}
		}
		return "init", nil
	}, soft.WithCommitToBranch(branch))
	if err != nil {
		return err
	}
	networks, err := s.getNetworks(user)
	if err != nil {
		return err
	}
	apps := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	instanceApp, err := installer.FindEnvApp(apps, "dodo-app-instance")
	if err != nil {
		return err
	}
	instanceAppStatus, err := installer.FindEnvApp(apps, "dodo-app-instance-status")
	if err != nil {
		return err
	}
	suffixGen := installer.NewFixedLengthRandomSuffixGenerator(3)
	suffix, err := suffixGen.Generate()
	if err != nil {
		return err
	}
	namespace := fmt.Sprintf("%s%s%s", s.env.NamespacePrefix, instanceApp.Namespace(), suffix)
	s.setAppConfig(appName, branch, appConfig{namespace, network})
	resources, err := s.updateDodoApp(instanceAppStatus, appName, branch, namespace, networks, user)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return err
	}
	if err = s.createCommit(appName, branch, commit, initCommitMsg, err, resources); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return err
	}
	configRepo, err := s.client.GetRepo(ConfigRepoName)
	if err != nil {
		return err
	}
	hf := installer.NewGitHelmFetcher()
	m, err := installer.NewAppManager(configRepo, s.nsc, s.jc, hf, s.vpnKeyGen, "/")
	if err != nil {
		return err
	}
	appPath := fmt.Sprintf("/%s/%s", appName, branch)
	_, err = configRepo.Do(func(fs soft.RepoFS) (string, error) {
		w, err := fs.Writer(appConfigsFile)
		if err != nil {
			return "", err
		}
		defer w.Close()
		if err := json.NewEncoder(w).Encode(s.appConfigs); err != nil {
			return "", err
		}
		if _, err := m.Install(
			instanceApp,
			appName,
			appPath,
			namespace,
			map[string]any{
				"repoAddr":         s.client.GetRepoAddress(appName),
				"repoHost":         strings.Split(s.client.Address(), ":")[0],
				"branch":           fmt.Sprintf("dodo_%s", branch),
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
	})
	if err != nil {
		return err
	}
	return s.initAppACLs(m, appPath, appName, branch, user)
}

func (s *DodoAppServer) initAppACLs(m *installer.AppManager, path, appName, branch, user string) error {
	cfg, err := m.GetInstance(path)
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
	if branch != "master" {
		return nil
	}
	if err := s.client.AddReadOnlyCollaborator(appName, "fluxcd"); err != nil {
		return err
	}
	if err := s.client.AddReadWriteCollaborator(appName, user); err != nil {
		return err
	}
	if err := s.client.AddWebhook(appName, fmt.Sprintf("http://%s/update", s.self), "--active=true", "--events=push", "--content-type=json"); err != nil {
		return err
	}
	if !s.external {
		go func() {
			users, err := s.client.GetAllUsers()
			if err != nil {
				fmt.Println(err)
				return
			}
			for _, user := range users {
				// TODO(gio): fluxcd should have only read access
				if err := s.client.AddReadWriteCollaborator(appName, user); err != nil {
					fmt.Println(err)
				}
			}
		}()
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	go s.reconciler.Reconcile(ctx, s.namespace, "config")
	return nil
}

type apiAddAdminKeyReq struct {
	User      string `json:"user"`
	PublicKey string `json:"publicKey"`
}

func (s *DodoAppServer) handleAPIAddPublicKey(w http.ResponseWriter, r *http.Request) {
	var req apiAddAdminKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.User == "" {
		http.Error(w, "invalid user", http.StatusBadRequest)
		return
	}
	if req.PublicKey == "" {
		http.Error(w, "invalid public key", http.StatusBadRequest)
		return
	}
	if err := s.client.AddPublicKey(req.User, req.PublicKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type dodoAppRendered struct {
	App struct {
		Ingress struct {
			Network   string `json:"network"`
			Subdomain string `json:"subdomain"`
		} `json:"ingress"`
	} `json:"app"`
	Input struct {
		AppId string `json:"appId"`
	} `json:"input"`
}

// TODO(gio): must not require owner, now we need it to bootstrap dev vm.
func (s *DodoAppServer) updateDodoApp(
	appStatus installer.EnvApp,
	name string,
	branch string,
	namespace string,
	networks []installer.Network,
	owner string,
) (installer.ReleaseResources, error) {
	repo, err := s.client.GetRepoBranch(name, branch)
	if err != nil {
		return installer.ReleaseResources{}, err
	}
	hf := installer.NewGitHelmFetcher()
	m, err := installer.NewAppManager(repo, s.nsc, s.jc, hf, s.vpnKeyGen, "/.dodo")
	if err != nil {
		return installer.ReleaseResources{}, err
	}
	appCfg, err := soft.ReadFile(repo, "app.cue")
	if err != nil {
		return installer.ReleaseResources{}, err
	}
	app, err := installer.NewDodoApp(appCfg)
	if err != nil {
		return installer.ReleaseResources{}, err
	}
	lg := installer.GitRepositoryLocalChartGenerator{"app", namespace}
	var ret installer.ReleaseResources
	if _, err := repo.Do(func(r soft.RepoFS) (string, error) {
		ret, err = m.Install(
			app,
			"app",
			"/.dodo/app",
			namespace,
			map[string]any{
				"repoAddr":       repo.FullAddress(),
				"repoPublicAddr": s.repoPublicAddr,
				"managerAddr":    fmt.Sprintf("http://%s", s.self),
				"appId":          name,
				"branch":         branch,
				"sshPrivateKey":  s.sshKey,
				"username":       owner,
			},
			installer.WithNoPull(),
			installer.WithNoPublish(),
			installer.WithConfig(&s.env),
			installer.WithNetworks(networks),
			installer.WithLocalChartGenerator(lg),
			installer.WithNoLock(),
		)
		if err != nil {
			return "", err
		}
		var rendered dodoAppRendered
		if err := json.NewDecoder(bytes.NewReader(ret.RenderedRaw)).Decode(&rendered); err != nil {
			return "", nil
		}
		if _, err := m.Install(
			appStatus,
			"status",
			"/.dodo/status",
			s.namespace,
			map[string]any{
				"appName":      rendered.Input.AppId,
				"network":      rendered.App.Ingress.Network,
				"appSubdomain": rendered.App.Ingress.Subdomain,
			},
			installer.WithNoPull(),
			installer.WithNoPublish(),
			installer.WithConfig(&s.env),
			installer.WithNetworks(networks),
			installer.WithLocalChartGenerator(lg),
			installer.WithNoLock(),
		); err != nil {
			return "", err
		}
		return "install app", nil
	},
		soft.WithCommitToBranch(fmt.Sprintf("dodo_%s", branch)),
		soft.WithForce(),
	); err != nil {
		return installer.ReleaseResources{}, err
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	go s.reconciler.Reconcile(ctx, namespace, "app")
	return ret, nil
}

func (s *DodoAppServer) renderAppConfigTemplate(appType string, network installer.Network, subdomain string) (map[string][]byte, error) {
	appType = strings.Replace(appType, ":", "-", 1)
	appTmpl, err := s.appTmpls.Find(appType)
	if err != nil {
		return nil, err
	}
	return appTmpl.Render(network, subdomain)
}

func generatePassword() string {
	return "foo"
}

func (s *DodoAppServer) getNetworks(user string) ([]installer.Network, error) {
	addr := fmt.Sprintf("%s/api/networks", s.envAppManagerAddr)
	resp, err := http.Get(addr)
	if err != nil {
		return nil, err
	}
	networks := []installer.Network{}
	if json.NewDecoder(resp.Body).Decode(&networks); err != nil {
		return nil, err
	}
	return s.nf.Filter(user, networks)
}

type publicNetworkData struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type publicData struct {
	Networks []publicNetworkData `json:"networks"`
	Types    []string            `json:"types"`
}

func (s *DodoAppServer) handleAPIPublicData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	s.l.Lock()
	defer s.l.Unlock()
	networks, err := s.getNetworks("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var ret publicData
	for _, n := range networks {
		if s.isNetworkUseAllowed(strings.ToLower(n.Name)) {
			ret.Networks = append(ret.Networks, publicNetworkData{n.Name, n.Domain})
		}
	}
	for _, t := range s.appTmpls.Types() {
		ret.Types = append(ret.Types, strings.Replace(t, "-", ":", 1))
	}
	if err := json.NewEncoder(w).Encode(ret); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *DodoAppServer) createCommit(name, branch, hash, message string, err error, resources installer.ReleaseResources) error {
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		if err := s.st.CreateCommit(name, branch, hash, message, "FAILED", err.Error(), nil); err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			return err
		}
		return err
	}
	var resB bytes.Buffer
	if err := json.NewEncoder(&resB).Encode(resources); err != nil {
		if err := s.st.CreateCommit(name, branch, hash, message, "FAILED", err.Error(), nil); err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			return err
		}
		return err
	}
	if err := s.st.CreateCommit(name, branch, hash, message, "OK", "", resB.Bytes()); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return err
	}
	return nil
}

func pickNetwork(networks []installer.Network, network string) []installer.Network {
	for _, n := range networks {
		if n.Name == network {
			return []installer.Network{n}
		}
	}
	return []installer.Network{}
}

type NetworkFilter interface {
	Filter(user string, networks []installer.Network) ([]installer.Network, error)
}

type noNetworkFilter struct{}

func NewNoNetworkFilter() NetworkFilter {
	return noNetworkFilter{}
}

func (f noNetworkFilter) Filter(user string, networks []installer.Network) ([]installer.Network, error) {
	return networks, nil
}

type filterByOwner struct {
	st Store
}

func NewNetworkFilterByOwner(st Store) NetworkFilter {
	return &filterByOwner{st}
}

func (f *filterByOwner) Filter(user string, networks []installer.Network) ([]installer.Network, error) {
	if user == "" {
		return networks, nil
	}
	network, err := f.st.GetUserNetwork(user)
	if err != nil {
		return nil, err
	}
	ret := []installer.Network{}
	for _, n := range networks {
		if n.Name == network {
			ret = append(ret, n)
		}
	}
	return ret, nil
}

type allowListFilter struct {
	allowed []string
}

func NewAllowListFilter(allowed []string) NetworkFilter {
	return &allowListFilter{allowed}
}

func (f *allowListFilter) Filter(user string, networks []installer.Network) ([]installer.Network, error) {
	ret := []installer.Network{}
	for _, n := range networks {
		if slices.Contains(f.allowed, n.Name) {
			ret = append(ret, n)
		}
	}
	return ret, nil
}

type combinedNetworkFilter struct {
	filters []NetworkFilter
}

func NewCombinedFilter(filters ...NetworkFilter) NetworkFilter {
	return &combinedNetworkFilter{filters}
}

func (f *combinedNetworkFilter) Filter(user string, networks []installer.Network) ([]installer.Network, error) {
	ret := networks
	var err error
	for _, f := range f.filters {
		ret, err = f.Filter(user, ret)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

type user struct {
	Username      string   `json:"username"`
	Email         string   `json:"email"`
	SSHPublicKeys []string `json:"sshPublicKeys,omitempty"`
}

func (s *DodoAppServer) handleAPISyncUsers(_ http.ResponseWriter, _ *http.Request) {
	go s.syncUsers()
}

func (s *DodoAppServer) syncUsers() {
	if s.external {
		panic("MUST NOT REACH!")
	}
	resp, err := http.Get(fmt.Sprintf("%s?selfAddress=%s/api/sync-users", s.fetchUsersAddr, s.self))
	if err != nil {
		return
	}
	users := []user{}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		fmt.Println(err)
		return
	}
	validUsernames := make(map[string]user)
	for _, u := range users {
		validUsernames[u.Username] = u
	}
	allClientUsers, err := s.client.GetAllUsers()
	if err != nil {
		fmt.Println(err)
		return
	}
	keyToUser := make(map[string]string)
	for _, clientUser := range allClientUsers {
		if clientUser == "admin" || clientUser == "fluxcd" {
			continue
		}
		userData, ok := validUsernames[clientUser]
		if !ok {
			if err := s.client.RemoveUser(clientUser); err != nil {
				fmt.Println(err)
				return
			}
		} else {
			existingKeys, err := s.client.GetUserPublicKeys(clientUser)
			if err != nil {
				fmt.Println(err)
				return
			}
			for _, existingKey := range existingKeys {
				cleanKey := soft.CleanKey(existingKey)
				keyOk := slices.ContainsFunc(userData.SSHPublicKeys, func(key string) bool {
					return cleanKey == soft.CleanKey(key)
				})
				if !keyOk {
					if err := s.client.RemovePublicKey(clientUser, existingKey); err != nil {
						fmt.Println(err)
					}
				} else {
					keyToUser[cleanKey] = clientUser
				}
			}
		}
	}
	for _, u := range users {
		if err := s.st.CreateUser(u.Username, nil, ""); err != nil && !errors.Is(err, ErrorAlreadyExists) {
			fmt.Println(err)
			return
		}
		if len(u.SSHPublicKeys) == 0 {
			continue
		}
		ok, err := s.client.UserExists(u.Username)
		if err != nil {
			fmt.Println(err)
			return
		}
		if !ok {
			if err := s.client.AddUser(u.Username, u.SSHPublicKeys[0]); err != nil {
				fmt.Println(err)
				return
			}
		} else {
			for _, key := range u.SSHPublicKeys {
				cleanKey := soft.CleanKey(key)
				if user, ok := keyToUser[cleanKey]; ok {
					if u.Username != user {
						panic("MUST NOT REACH! IMPOSSIBLE KEY USER RECORD")
					}
					continue
				}
				if err := s.client.AddPublicKey(u.Username, cleanKey); err != nil {
					fmt.Println(err)
					return
				}
			}
		}
	}
	repos, err := s.client.GetAllRepos()
	if err != nil {
		return
	}
	for _, r := range repos {
		if r == ConfigRepoName {
			continue
		}
		for _, u := range users {
			if err := s.client.AddReadWriteCollaborator(r, u.Username); err != nil {
				fmt.Println(err)
				continue
			}
		}
	}
}

func extractResourceData(resources []installer.Resource) (resourceData, error) {
	var ret resourceData
	for _, r := range resources {
		t, ok := r.Annotations["dodo.cloud/resource-type"]
		if !ok {
			continue
		}
		switch t {
		case "volume":
			name, ok := r.Annotations["dodo.cloud/resource.volume.name"]
			if !ok {
				return resourceData{}, fmt.Errorf("no name")
			}
			size, ok := r.Annotations["dodo.cloud/resource.volume.size"]
			if !ok {
				return resourceData{}, fmt.Errorf("no size")
			}
			ret.Volume = append(ret.Volume, volume{name, size})
		case "postgresql":
			name, ok := r.Annotations["dodo.cloud/resource.postgresql.name"]
			if !ok {
				return resourceData{}, fmt.Errorf("no name")
			}
			version, ok := r.Annotations["dodo.cloud/resource.postgresql.version"]
			if !ok {
				return resourceData{}, fmt.Errorf("no version")
			}
			volume, ok := r.Annotations["dodo.cloud/resource.postgresql.volume"]
			if !ok {
				return resourceData{}, fmt.Errorf("no volume")
			}
			ret.PostgreSQL = append(ret.PostgreSQL, postgresql{name, version, volume})
		case "ingress":
			host, ok := r.Annotations["dodo.cloud/resource.ingress.host"]
			if !ok {
				return resourceData{}, fmt.Errorf("no host")
			}
			ret.Ingress = append(ret.Ingress, ingress{host})
		case "virtual-machine":
			name, ok := r.Annotations["dodo.cloud/resource.virtual-machine.name"]
			if !ok {
				return resourceData{}, fmt.Errorf("no name")
			}
			user, ok := r.Annotations["dodo.cloud/resource.virtual-machine.user"]
			if !ok {
				return resourceData{}, fmt.Errorf("no user")
			}
			cpuCoresS, ok := r.Annotations["dodo.cloud/resource.virtual-machine.cpu-cores"]
			if !ok {
				return resourceData{}, fmt.Errorf("no cpu cores")
			}
			cpuCores, err := strconv.Atoi(cpuCoresS)
			if err != nil {
				return resourceData{}, fmt.Errorf("invalid cpu cores: %s", cpuCoresS)
			}
			memory, ok := r.Annotations["dodo.cloud/resource.virtual-machine.memory"]
			if !ok {
				return resourceData{}, fmt.Errorf("no memory")
			}
			ret.VirtualMachine = append(ret.VirtualMachine, vm{name, user, cpuCores, memory})
		default:
			fmt.Printf("Unknown resource: %+v\n", r.Annotations)
		}
	}
	return ret, nil
}

func createDevBranchAppConfig(from []byte, branch, username string) (string, []byte, error) {
	cfg, err := installer.ParseCueAppConfig(installer.CueAppData{"app.cue": from})
	if err != nil {
		return "", nil, err
	}
	if err := cfg.Err(); err != nil {
		return "", nil, err
	}
	if err := cfg.Validate(); err != nil {
		return "", nil, err
	}
	subdomain := cfg.LookupPath(cue.ParsePath("app.ingress.subdomain"))
	if err := subdomain.Err(); err != nil {
		return "", nil, err
	}
	subdomainStr, err := subdomain.String()
	network := cfg.LookupPath(cue.ParsePath("app.ingress.network"))
	if err := network.Err(); err != nil {
		return "", nil, err
	}
	networkStr, err := network.String()
	if err != nil {
		return "", nil, err
	}
	newCfg := map[string]any{}
	if err := cfg.Decode(&newCfg); err != nil {
		return "", nil, err
	}
	app, ok := newCfg["app"].(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("not a map")
	}
	app["ingress"].(map[string]any)["subdomain"] = fmt.Sprintf("%s-%s", branch, subdomainStr)
	app["dev"] = map[string]any{
		"enabled":  true,
		"username": username,
	}
	buf, err := json.MarshalIndent(newCfg, "", "\t")
	if err != nil {
		return "", nil, err
	}
	return networkStr, buf, nil
}
