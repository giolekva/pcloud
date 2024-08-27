package welcome

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/giolekva/pcloud/core/installer"

	"github.com/Masterminds/sprig/v3"
	"github.com/gomarkdown/markdown"
)

//go:embed launcher-tmpl/launcher.html
var indexHTML embed.FS

type AppLauncherInfo struct {
	Id         string
	Name       string
	Icon       template.HTML
	Help       []HelpDocumentRendered
	URL        string
	DisplayURL string
}

type HelpDocumentRendered struct {
	Title    string
	Contents template.HTML
	Children []HelpDocumentRendered
}

type AppDirectory interface {
	GetAllApps() ([]AppLauncherInfo, error)
}

type AppManagerDirectory struct {
	AppManager *installer.AppManager
}

func (d *AppManagerDirectory) GetAllApps() ([]AppLauncherInfo, error) {
	all, err := d.AppManager.GetAllInstances()
	if err != nil {
		return nil, err
	}
	ret := []AppLauncherInfo{}
	for _, a := range all {
		if a.URL == "" && len(a.Help) == 0 {
			continue
		}
		ret = append(ret, AppLauncherInfo{
			Id:         a.Id,
			Name:       a.AppId,
			Icon:       template.HTML(a.Icon),
			Help:       toMarkdown(a.Help),
			URL:        a.URL,
			DisplayURL: shortenURL(a.URL, a.Env.Domain),
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Name == "app-manager" {
			return true
		}
		if ret[j].Name == "app-manager" {
			return false
		}
		if ret[i].Name == "headscale" {
			return ret[j].Name != "app-manager"
		}
		if ret[j].Name == "headscale" {
			return false
		}
		return ret[i].Name < ret[j].Name
	})
	return ret, nil
}

type LauncherServer struct {
	port         int
	logoutURL    string
	appDirectory AppDirectory
	homeTmpl     *template.Template
}

func NewLauncherServer(
	port int,
	logoutURL string,
	appDirectory AppDirectory,
) (*LauncherServer, error) {
	tmpl, err := indexHTML.ReadFile("launcher-tmpl/launcher.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}
	t := template.New("index").Funcs(template.FuncMap(sprig.FuncMap())).Funcs(template.FuncMap{
		"GetUserInitials": getUserInitials,
		"CleanAppName":    cleanAppName,
	})
	t, err = t.Parse(string(tmpl))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}
	return &LauncherServer{
		port,
		logoutURL,
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

func shortenURL(url, domain string) string {
	return strings.Replace(url, domain, "..", 1)
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
	http.Handle("/stat/", cachingHandler{http.FileServer(http.FS(statAssets))})
	http.HandleFunc("/", s.homeHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil))
}

type homeHandlerData struct {
	LoggedInUsername string
	AllAppsInfo      []AppLauncherInfo
	LogoutURL        string
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
		LogoutURL:        s.logoutURL,
	}
	if err := s.homeTmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func toMarkdown(help []installer.HelpDocument) []HelpDocumentRendered {
	if help == nil {
		return nil
	}
	var ret []HelpDocumentRendered
	for _, h := range help {
		ret = append(ret, HelpDocumentRendered{
			Title:    h.Title,
			Contents: template.HTML(markdown.ToHTML([]byte(h.Contents), nil, nil)),
			Children: toMarkdown(h.Children),
		})
	}
	return ret
}
