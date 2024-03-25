package welcome

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/giolekva/pcloud/core/installer"
)

//go:embed create-account.html
var indexHtml []byte

//go:embed create-account-success.html
var successHtml []byte

//go:embed static/*
var staticAssets embed.FS

type Server struct {
	port              int
	repo              installer.RepoIO
	nsCreator         installer.NamespaceCreator
	createAccountAddr string
	loginAddr         string
}

func NewServer(
	port int,
	repo installer.RepoIO,
	nsCreator installer.NamespaceCreator,
	createAccountAddr string,
	loginAddr string,
) *Server {
	return &Server{
		port,
		repo,
		nsCreator,
		createAccountAddr,
		loginAddr,
	}
}

func (s *Server) Start() {
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticAssets)))
	r.Path("/").Methods("POST").HandlerFunc(s.createAccount)
	r.Path("/").Methods("GET").HandlerFunc(s.createAccountForm)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil))
}

func (s *Server) createAccountForm(w http.ResponseWriter, r *http.Request) {
	renderRegistrationForm(w, formData{})
}

type formData struct {
	UsernameErrors []string
	PasswordErrors []string
	Data           createAccountReq
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

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Errors []ValidationError `json:"errors"`
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

func renderRegistrationForm(w http.ResponseWriter, data formData) {
	tmpl, err := template.New("create-account").Parse(string(indexHtml))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func renderRegistrationSuccess(w http.ResponseWriter, loginAddr string) {
	data := struct {
		LoginAddr string
	}{
		LoginAddr: loginAddr,
	}
	tmpl, err := template.New("create-account-success").Parse(string(successHtml))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
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
			var respBody bytes.Buffer
			if _, err := io.Copy(&respBody, resp.Body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			respStr := respBody.String()
			log.Println(respStr)
			http.Error(w, respStr, http.StatusInternalServerError)
			return
		}
		if resp.StatusCode != http.StatusOK {
			var errResponse ErrorResponse
			if err := json.NewDecoder(resp.Body).Decode(&errResponse); err != nil {
				http.Error(w, "Error Decoding JSON", http.StatusInternalServerError)
				return
			}
			var usernameErrors, passwordErrors []string
			for _, err := range errResponse.Errors {
				if err.Field == "username" {
					usernameErrors = append(usernameErrors, err.Message)
				}
				if err.Field == "password" {
					passwordErrors = append(passwordErrors, err.Message)
				}
			}
			renderRegistrationForm(w, formData{
				usernameErrors,
				passwordErrors,
				req,
			})
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
			if err := appManager.Install(app, nsGen, suffixGen, map[string]any{
				"username": req.Username,
				"preAuthKey": map[string]any{
					"enabled": false,
				},
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	renderRegistrationSuccess(w, s.loginAddr)
}
