package welcome

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/giolekva/pcloud/core/installer"
)

//go:embed launcher-tmpl/launcher.html
var indexHTML embed.FS

//go:embed static/*
var files embed.FS

type AppLauncherInfo struct {
	Name string
	Icon template.HTML
	Help []installer.HelpDocument
	Url  string
}

type AppDirectory interface {
	GetAllApps() ([]AppLauncherInfo, error)
}

type AppManagerDirectory struct {
	AppManager *installer.AppManager
}

type LauncherServer struct {
	port         int
	logoutUrl    string
	appDirectory AppDirectory
	homeTmpl     *template.Template
}

func NewLauncherServer(
	port int,
	logoutUrl string,
	appDirectory AppDirectory,
) (*LauncherServer, error) {
	tmpl, err := indexHTML.ReadFile("launcher-tmpl/launcher.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}
	t := template.New("index").Funcs(template.FuncMap{
		"GetUserInitials": getUserInitials,
		"CleanAppName":    cleanAppName,
	})
	t, err = t.Parse(string(tmpl))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}
	return &LauncherServer{
		port,
		logoutUrl,
		appDirectory,
		t,
	}, nil
}

func getUserInitials(username string) string {
	if username == "" {
		return ""
	}
	return strings.ToUpper(username[:1])
}

func cleanAppName(name string) string {
	cleanName := strings.ToLower(name)
	cleanName = strings.ReplaceAll(cleanName, " ", "-")
	return cleanName
}

func (d *AppManagerDirectory) GetAllApps() ([]AppLauncherInfo, error) {
	allAppInstances, err := d.AppManager.FindAllInstances()
	if err != nil {
		return nil, err
	}
	var ret []AppLauncherInfo
	for _, appInstance := range allAppInstances {
		appLauncherInfo := AppLauncherInfo{
			Name: appInstance.AppId,
			Icon: template.HTML(appInstance.Icon),
			Help: appInstance.Help,
			Url:  appInstance.Url,
		}
		ret = append(ret, appLauncherInfo)
	}
	return ret, nil
}

func getLoggedInUser(r *http.Request) (string, error) {
	if user := r.Header.Get("X-User"); user != "" {
		return user, nil
	} else {
		return "", fmt.Errorf("unauthenticated")
	}
	// return "Username", nil
}

func (s *LauncherServer) Start() {
	http.Handle("/static/", http.FileServer(http.FS(files)))
	http.HandleFunc("/", s.homeHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil))
}

type homeHandlerData struct {
	LoggedInUsername string
	AllAppsInfo      []AppLauncherInfo
}

func (s *LauncherServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUsername, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	allAppsInfo, err := s.appDirectory.GetAllApps()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	data := homeHandlerData{
		LoggedInUsername: loggedInUsername,
		AllAppsInfo:      allAppsInfo,
	}
	if err := s.homeTmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
