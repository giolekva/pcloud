package welcome

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

//go:embed launcher-tmpl/launcher.html
var indexHTML embed.FS

//go:embed static/*
var files embed.FS

type AppLauncherInfo struct {
	Name        string
	Description string
	Icon        template.HTML
	Help        []HelpDocument
	Url         string
}

type HelpDocument struct {
	Title    string
	Contents string
	Children []HelpDocument
}

type AppDirectory interface {
	GetAllApps() ([]AppLauncherInfo, error)
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
	loggedInUsername := "longusername"
	allAppsInfo, err := s.appDirectory.GetAllApps()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
