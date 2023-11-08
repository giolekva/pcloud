package apprepo

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"sigs.k8s.io/yaml"
)

type App interface {
	Name() string
	Version() string
	Reader() (io.ReadCloser, error)
}

type Server struct {
	schemeWithHost string
	port           int
	apps           []App
}

func NewServer(schemeWithHost string, port int, apps []App) *Server {
	return &Server{schemeWithHost, port, apps}
}

func (s *Server) Start() error {
	r := mux.NewRouter()
	r.Path("/").Methods("GET").HandlerFunc(s.allApps)
	r.Path("/app/{name}/{version}.tar.gz").Methods("GET").HandlerFunc(s.app)
	http.Handle("/", r)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}

func (s *Server) allApps(w http.ResponseWriter, r *http.Request) {
	entries := make(map[string][]map[string]any)
	for _, a := range s.apps {
		e, ok := entries[a.Name()]
		if !ok {
			e = make([]map[string]any, 0)
		}
		e = append(e, map[string]any{
			"version": a.Version(),
			"urls":    []string{fmt.Sprintf("%s/app/%s/%s.tar.gz", s.schemeWithHost, a.Name(), a.Version())},
		})
		entries[a.Name()] = e
	}
	resp := map[string]any{
		"apiVersion": "v1",
		"entries":    entries,
	}
	b, err := yaml.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}

func (s *Server) app(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	version := vars["version"]
	for _, a := range s.apps {
		if a.Name() == name && a.Version() == version {
			r, err := a.Reader()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer r.Close()
			io.Copy(w, r)
			return
		}
	}
	http.Error(w, "Not found", http.StatusNotFound)
}
