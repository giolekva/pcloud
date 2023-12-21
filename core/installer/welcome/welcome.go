package welcome

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/giolekva/pcloud/core/installer"
)

//go:embed create-account.html
var indexHtml []byte

//go:embed static/*
var staticAssets embed.FS

type Server struct {
	port              int
	repo              installer.RepoIO
	nsCreator         installer.NamespaceCreator
	createAccountAddr string
}

func NewServer(
	port int,
	repo installer.RepoIO,
	nsCreator installer.NamespaceCreator,
	createAccountAddr string,
) *Server {
	return &Server{
		port,
		repo,
		nsCreator,
		createAccountAddr,
	}
}

func (s *Server) Start() {
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticAssets)))
	r.Path("/").Methods("POST").HandlerFunc(s.createAdminAccount)
	r.Path("/").Methods("GET").HandlerFunc(s.createAdminAccountForm)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil))
}

func (s *Server) createAdminAccountForm(w http.ResponseWriter, _ *http.Request) {
	if _, err := w.Write(indexHtml); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type createAccountReq struct {
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	SecretToken string `json:"secretToken,omitempty"`
}

type apiCreateAccountReq struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func getFormValue(v url.Values, name string) (string, error) {
	items, ok := v[name]
	if !ok || len(items) != 1 {
		return "", fmt.Errorf("%s not found", name)
	}
	return items[0], nil
}

func extractReq(r *http.Request) (createAccountReq, error) {
	var req createAccountReq
	if err := func() error {
		var err error
		if err = r.ParseForm(); err != nil {
			return err
		}
		if req.Username, err = getFormValue(r.PostForm, "username"); err != nil {
			return err
		}
		if req.Password, err = getFormValue(r.PostForm, "password"); err != nil {
			return err
		}
		if req.SecretToken, err = getFormValue(r.PostForm, "secret-token"); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return createAccountReq{}, err
		}
	}
	return req, nil
}

func (s *Server) createAdminAccount(w http.ResponseWriter, r *http.Request) {
	req, err := extractReq(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	{
		var buf bytes.Buffer
		cr := apiCreateAccountReq{req.Username, req.Password}
		if err := json.NewEncoder(&buf).Encode(cr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp, err := http.Post(s.createAccountAddr, "application/json", &buf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// TODO(gio): better handle status code and error message
		if resp.StatusCode != http.StatusOK {
			var e bytes.Buffer
			io.Copy(&e, resp.Body)
			http.Error(w, e.String(), http.StatusInternalServerError)
			return
		}
	}
	{
		config, err := s.repo.ReadConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		nsGen := installer.NewPrefixGenerator(config.Values.NamespacePrefix)
		suffixGen := installer.NewEmptySuffixGenerator()
		appManager, err := installer.NewAppManager(s.repo, s.nsCreator)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		appsRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
		{
			app, err := appsRepo.Find("headscale-user")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := appManager.Install(*app, nsGen, suffixGen, map[string]any{
				"Username": req.Username,
				"PreAuthKey": map[string]any{
					"Enabled": false,
				},
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	if _, err := w.Write([]byte("OK")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
