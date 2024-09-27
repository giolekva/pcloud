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
	"os"

	"github.com/gorilla/mux"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

//go:embed welcome-tmpl/*
var welcomeTmpls embed.FS

//go:embed static/*
var staticAssets embed.FS

//go:embed stat/*
var statAssets embed.FS

type welcomeTmplts struct {
	createAccount        *template.Template
	createAccountSuccess *template.Template
}

func parseTemplatesWelcome(fs embed.FS) (welcomeTmplts, error) {
	base, err := template.New("base.html").ParseFS(fs, "welcome-tmpl/base.html")
	if err != nil {
		return welcomeTmplts{}, err
	}
	parse := func(path string) (*template.Template, error) {
		if b, err := base.Clone(); err != nil {
			return nil, err
		} else {
			return b.ParseFS(fs, path)
		}
	}
	createAccount, err := parse("welcome-tmpl/create-account.html")
	if err != nil {
		return welcomeTmplts{}, err
	}
	createAccountSuccess, err := parse("welcome-tmpl/create-account-success.html")
	if err != nil {
		return welcomeTmplts{}, err
	}
	return welcomeTmplts{createAccount, createAccountSuccess}, nil
}

type Server struct {
	port              int
	repo              soft.RepoIO
	nsCreator         installer.NamespaceCreator
	hf                installer.HelmFetcher
	createAccountAddr string
	loginAddr         string
	membershipsAddr   string
	tmpl              welcomeTmplts
}

func NewServer(
	port int,
	repo soft.RepoIO,
	nsCreator installer.NamespaceCreator,
	hf installer.HelmFetcher,
	createAccountAddr string,
	loginAddr string,
	membershipsAddr string,
) (*Server, error) {
	tmplts, err := parseTemplatesWelcome(welcomeTmpls)
	if err != nil {
		return nil, err
	}
	return &Server{
		port,
		repo,
		nsCreator,
		hf,
		createAccountAddr,
		loginAddr,
		membershipsAddr,
		tmplts,
	}, nil
}

func (s *Server) Start() {
	r := mux.NewRouter()
	r.PathPrefix("/stat/").Handler(cachingHandler{http.FileServer(http.FS(statAssets))})
	r.Path("/").Methods("POST").HandlerFunc(s.createAccount)
	r.Path("/").Methods("GET").HandlerFunc(s.createAccountForm)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil))
}

func (s *Server) createAccountForm(w http.ResponseWriter, r *http.Request) {
	s.renderRegistrationForm(w, formData{})
}

type formData struct {
	UsernameErrors []string
	PasswordErrors []string
	Data           createAccountReq
}

type cpFormData struct {
	UsernameErrors []string
	PasswordErrors []string
	Password       string
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

func (s *Server) renderRegistrationForm(w http.ResponseWriter, data formData) {
	if err := s.tmpl.createAccount.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) renderRegistrationSuccess(w http.ResponseWriter, loginAddr string) {
	data := struct {
		LoginAddr string
	}{
		LoginAddr: loginAddr,
	}
	if err := s.tmpl.createAccountSuccess.Execute(w, data); err != nil {
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
			s.renderRegistrationForm(w, formData{
				usernameErrors,
				passwordErrors,
				req,
			})
			return
		}
	}
	if err := s.createUser(req.Username); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderRegistrationSuccess(w, s.loginAddr)
}

type firstAccount struct {
	Created bool     `json:"created"`
	Domain  string   `json:"domain"`
	Groups  []string `json:"groups"`
}

type initRequest struct {
	User   string   `json:"user"`
	Email  string   `json:"email"`
	Groups []string `json:"groups"`
}

type createUserRequest struct {
	User  string `json:"user"`
	Email string `json:"email"`
}

func (s *Server) createUser(username string) error {
	_, err := s.repo.Do(func(r soft.RepoFS) (string, error) {
		var fa firstAccount
		if err := soft.ReadYaml(r, "first-account.yaml", &fa); err != nil {
			return "", err
		}
		var resp *http.Response
		var err error
		if fa.Created {
			req := createUserRequest{username, fmt.Sprintf("%s@%s", username, fa.Domain)}
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(req); err != nil {
				return "", err
			}
			resp, err = http.Post(
				fmt.Sprintf("%s/api/users", s.membershipsAddr),
				"applications/json",
				&buf,
			)
		} else {
			req := initRequest{username, fmt.Sprintf("%s@%s", username, fa.Domain), fa.Groups}
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(req); err != nil {
				return "", err
			}
			resp, err = http.Post(
				fmt.Sprintf("%s/api/init", s.membershipsAddr),
				"applications/json",
				&buf,
			)
			fa.Created = true
			if err := soft.WriteYaml(r, "first-account.yaml", fa); err != nil {
				return "", err
			}
		}
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		fmt.Printf("Memberships resp: %d", resp.StatusCode)
		io.Copy(os.Stdout, resp.Body)
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("memberships error")
		}
		return "initialized groups for first account", nil
	})
	return err
}
