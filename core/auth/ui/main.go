package main

import (
	"bytes"
	"crypto/tls"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/itaysk/regogo"
)

var port = flag.Int("port", 8080, "Port to listen on")
var kratos = flag.String("kratos", "https://accounts.lekva.me", "Kratos URL")
var hydra = flag.String("hydra", "hydra.pcloud", "Hydra admin server address")
var emailDomain = flag.String("email-domain", "lekva.me", "Email domain")

var apiPort = flag.Int("api-port", 8081, "API Port to listen on")
var kratosAPI = flag.String("kratos-api", "", "Kratos API address")

var enableRegistration = flag.Bool("enable-registration", false, "If true account registration will be enabled")

var ErrNotLoggedIn = errors.New("Not logged in")

//go:embed templates/*
var tmpls embed.FS

//go:embed static
var static embed.FS

type Templates struct {
	WhoAmI   *template.Template
	Register *template.Template
	Login    *template.Template
	Consent  *template.Template
}

func ParseTemplates(fs embed.FS) (*Templates, error) {
	base, err := template.ParseFS(fs, "templates/base.html")
	if err != nil {
		return nil, err
	}
	parse := func(path string) (*template.Template, error) {
		if b, err := base.Clone(); err != nil {
			return nil, err
		} else {
			return b.ParseFS(fs, path)
		}
	}
	whoami, err := parse("templates/whoami.html")
	if err != nil {
		return nil, err
	}
	register, err := parse("templates/register.html")
	if err != nil {
		return nil, err
	}
	login, err := parse("templates/login.html")
	if err != nil {
		return nil, err
	}
	consent, err := parse("templates/consent.html")
	if err != nil {
		return nil, err
	}
	return &Templates{whoami, register, login, consent}, nil
}

type Server struct {
	r                  *mux.Router
	serv               *http.Server
	kratos             string
	hydra              *HydraClient
	tmpls              *Templates
	enableRegistration bool
}

func NewServer(port int, kratos string, hydra *HydraClient, tmpls *Templates, enableRegistration bool) *Server {
	r := mux.NewRouter()
	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	return &Server{r, serv, kratos, hydra, tmpls, enableRegistration}
}

func cacheControlWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(giolekva): enable caching
		// w.Header().Set("Cache-Control", "max-age=2592000") // 30 days
		h.ServeHTTP(w, r)
	})
}

func (s *Server) Start() error {
	var staticFS = http.FS(static)
	fs := http.FileServer(staticFS)
	s.r.PathPrefix("/static/").Handler(cacheControlWrapper(fs))
	if s.enableRegistration {
		s.r.Path("/register").Methods(http.MethodGet).HandlerFunc(s.registerInitiate)
		s.r.Path("/register").Methods(http.MethodPost).HandlerFunc(s.register)
	}
	s.r.Path("/login").Methods(http.MethodGet).HandlerFunc(s.loginInitiate)
	s.r.Path("/login").Methods(http.MethodPost).HandlerFunc(s.login)
	s.r.Path("/consent").Methods(http.MethodGet).HandlerFunc(s.consent)
	s.r.Path("/consent").Methods(http.MethodPost).HandlerFunc(s.processConsent)
	s.r.Path("/logout").Methods(http.MethodGet).HandlerFunc(s.logout)
	s.r.Path("/").HandlerFunc(s.whoami)
	return s.serv.ListenAndServe()
}

func getCSRFToken(flowType, flow string, cookies []*http.Cookie) (string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	b, err := url.Parse(*kratos + "/self-service/" + flowType + "/browser")
	if err != nil {
		return "", err
	}
	client.Jar.SetCookies(b, cookies)
	resp, err := client.Get(fmt.Sprintf(*kratos+"/self-service/"+flowType+"/flows?id=%s", flow))
	if err != nil {
		return "", err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	token, err := regogo.Get(string(respBody), "input.ui.nodes[0].attributes.value")
	if err != nil {
		return "", err
	}
	return token.String(), nil
}

func (s *Server) registerInitiate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flow, ok := r.Form["flow"]
	if !ok {
		http.Redirect(w, r, s.kratos+"/self-service/registration/browser", http.StatusSeeOther)
		return
	}
	csrfToken, err := getCSRFToken("registration", flow[0], r.Cookies())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if err := s.tmpls.Register.Execute(w, csrfToken); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type regReq struct {
	CSRFToken string       `json:"csrf_token"`
	Method    string       `json:"method"`
	Password  string       `json:"password"`
	Traits    regReqTraits `json:"traits"`
}

type regReqTraits struct {
	Username string `json:"username"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flow, ok := r.Form["flow"]
	if !ok {
		http.Redirect(w, r, s.kratos+"/self-service/registration/browser", http.StatusSeeOther)
		return
	}
	req := regReq{
		CSRFToken: r.FormValue("csrf_token"),
		Method:    "password",
		Password:  r.FormValue("password"),
		Traits: regReqTraits{
			Username: r.FormValue("username"),
		},
	}
	var reqBody bytes.Buffer
	if err := json.NewEncoder(&reqBody).Encode(req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if resp, err := postToKratos("registration", flow[0], r.Cookies(), &reqBody); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		for _, c := range resp.Cookies() {
			http.SetCookie(w, c)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Login flow

func (s *Server) loginInitiate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if challenge, ok := r.Form["login_challenge"]; ok {
		username, err := getWhoAmIFromKratos(r.Cookies())
		if err != nil && err != ErrNotLoggedIn {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err == nil {
			redirectTo, err := s.hydra.LoginAcceptChallenge(challenge[0], username)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, redirectTo, http.StatusSeeOther)
			return
		}
		// TODO(giolekva): encrypt
		http.SetCookie(w, &http.Cookie{
			Name:     "login_challenge",
			Value:    challenge[0],
			HttpOnly: true,
		})
	}
	returnTo := r.Form.Get("return_to")
	flow, ok := r.Form["flow"]
	if !ok {
		addr := s.kratos + "/self-service/login/browser"
		if returnTo != "" {
			addr += fmt.Sprintf("?return_to=%s", returnTo)
		}
		http.Redirect(w, r, addr, http.StatusSeeOther)
		return
	}
	csrfToken, err := getCSRFToken("login", flow[0], r.Cookies())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if err := s.tmpls.Login.Execute(w, map[string]any{
		"csrfToken":          csrfToken,
		"enableRegistration": s.enableRegistration,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type loginReq struct {
	CSRFToken string `json:"csrf_token"`
	Method    string `json:"method"`
	Password  string `json:"password"`
	Username  string `json:"password_identifier"`
}

func postToKratos(flowType, flow string, cookies []*http.Cookie, req io.Reader) (*http.Response, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	b, err := url.Parse(*kratos + "/self-service/" + flowType + "/browser")
	if err != nil {
		return nil, err
	}
	client.Jar.SetCookies(b, cookies)
	resp, err := client.Post(fmt.Sprintf(*kratos+"/self-service/"+flowType+"?flow=%s", flow), "application/json", req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func postFormToKratos(flowType, flow string, cookies []*http.Cookie, data url.Values) (*http.Response, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	b, err := url.Parse(*kratos + "/self-service/" + flowType + "/browser")
	if err != nil {
		return nil, err
	}
	client.Jar.SetCookies(b, cookies)
	resp, err := client.PostForm(fmt.Sprintf(*kratos+"/self-service/"+flowType+"?flow=%s", flow), data)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type logoutResp struct {
	LogoutURL string `json:"logout_url"`
}

func getLogoutURLFromKratos(cookies []*http.Cookie) (string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	b, err := url.Parse(*kratos + "/self-service/logout/browser")
	if err != nil {
		return "", err
	}
	client.Jar.SetCookies(b, cookies)
	resp, err := client.Get(*kratos + "/self-service/logout/browser")
	if err != nil {
		return "", err
	}
	var lr logoutResp
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return "", err
	}
	return lr.LogoutURL, nil
}

func getWhoAmIFromKratos(cookies []*http.Cookie) (string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	b, err := url.Parse(*kratos + "/sessions/whoami")
	if err != nil {
		return "", err
	}
	client.Jar.SetCookies(b, cookies)
	resp, err := client.Get(*kratos + "/sessions/whoami")
	if err != nil {
		return "", err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	username, err := regogo.Get(string(respBody), "input.identity.traits.username")
	if err != nil {
		return "", err
	}
	if username.String() == "" {
		return "", ErrNotLoggedIn
	}
	return username.String(), nil

}

func extractError(r io.Reader) error {
	respBody, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	fmt.Printf("++ %s\n", respBody)
	t, err := regogo.Get(string(respBody), "input.ui.messages[0].type")
	if err != nil {
		return err
	}
	if t.String() == "error" {
		message, err := regogo.Get(string(respBody), "input.ui.messages[0].text")
		if err != nil {
			return err
		}
		return errors.New(message.String())
	}
	return nil
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flow, ok := r.Form["flow"]
	if !ok {
		http.Redirect(w, r, s.kratos+"/self-service/login/browser", http.StatusSeeOther)
		return
	}
	req := url.Values{
		"csrf_token": []string{r.FormValue("csrf_token")},
		"method":     []string{"password"},
		"password":   []string{r.FormValue("password")},
		"identifier": []string{r.FormValue("username")},
	}
	resp, err := postFormToKratos("login", flow[0], r.Cookies(), req)
	fmt.Printf("--- %d\n", resp.StatusCode)
	var vv bytes.Buffer
	io.Copy(&vv, resp.Body)
	fmt.Println(vv.String())
	if err != nil {
		if challenge, _ := r.Cookie("login_challenge"); challenge != nil {
			redirectTo, err := s.hydra.LoginRejectChallenge(challenge.Value, err.Error())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, redirectTo, http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, c := range resp.Cookies() {
		http.SetCookie(w, c)
	}
	if challenge, _ := r.Cookie("login_challenge"); challenge != nil {
		username, err := getWhoAmIFromKratos(resp.Cookies())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		redirectTo, err := s.hydra.LoginAcceptChallenge(challenge.Value, username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, redirectTo, http.StatusSeeOther)
		return
	}
	if resp.StatusCode == http.StatusSeeOther {
		http.Redirect(w, r, resp.Header.Get("Location"), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if logoutURL, err := getLogoutURLFromKratos(r.Cookies()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		http.Redirect(w, r, logoutURL, http.StatusSeeOther)
	}
}

func (s *Server) whoami(w http.ResponseWriter, r *http.Request) {
	if username, err := getWhoAmIFromKratos(r.Cookies()); err != nil {
		if errors.Is(err, ErrNotLoggedIn) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		if err := s.tmpls.WhoAmI.Execute(w, username); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// TODO(giolekva): verify if logged in
func (s *Server) consent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	challenge, ok := r.Form["consent_challenge"]
	if !ok {
		http.Error(w, "Consent challenge not provided", http.StatusBadRequest)
		return
	}
	consent, err := s.hydra.GetConsentChallenge(challenge[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	username, err := getWhoAmIFromKratos(r.Cookies())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	acceptedScopes := consent.RequestedScopes
	idToken := map[string]string{
		"username": username,
		"email":    username + "@" + *emailDomain,
	}
	// TODO(gio): is auto consent safe? should such behaviour be configurable?
	if redirectTo, err := s.hydra.ConsentAccept(r.FormValue("consent_challenge"), acceptedScopes, idToken); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Redirect(w, r, redirectTo, http.StatusSeeOther)
	}
	// w.Header().Set("Content-Type", "text/html")
	// if err := s.tmpls.Consent.Execute(w, consent.RequestedScopes); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
}

func (s *Server) processConsent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	username, err := getWhoAmIFromKratos(r.Cookies())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, accepted := r.Form["allow"]; accepted {
		acceptedScopes, _ := r.Form["scope"]
		idToken := map[string]string{
			"username": username,
			"email":    username + "@" + *emailDomain,
		}
		if redirectTo, err := s.hydra.ConsentAccept(r.FormValue("consent_challenge"), acceptedScopes, idToken); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			http.Redirect(w, r, redirectTo, http.StatusSeeOther)
		}
		return
	} else {
		// TODO(giolekva): implement rejection logic
	}
}

func main() {
	flag.Parse()
	t, err := ParseTemplates(tmpls)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		s := NewAPIServer(*apiPort, *kratosAPI)
		log.Fatal(s.Start())
	}()
	func() {
		s := NewServer(
			*port,
			*kratos,
			NewHydraClient(*hydra),
			t,
			*enableRegistration,
		)
		log.Fatal(s.Start())
	}()
}
