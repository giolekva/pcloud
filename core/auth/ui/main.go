package main

import (
	"bytes"
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

var ErrNotLoggedIn = errors.New("Not logged in")

//go:embed templates/*
var tmpls embed.FS

type Templates struct {
	WhoAmI       *template.Template
	Registration *template.Template
	Login        *template.Template
	Consent      *template.Template
}

func ParseTemplates(fs embed.FS) (*Templates, error) {
	whoami, err := template.ParseFS(fs, "templates/whoami.html")
	if err != nil {
		return nil, err
	}
	registration, err := template.ParseFS(fs, "templates/registration.html")
	if err != nil {
		return nil, err
	}
	login, err := template.ParseFS(fs, "templates/login.html")
	if err != nil {
		return nil, err
	}
	consent, err := template.ParseFS(fs, "templates/consent.html")
	if err != nil {
		return nil, err
	}
	return &Templates{whoami, registration, login, consent}, nil
}

type Server struct {
	kratos string
	hydra  *HydraClient
	tmpls  *Templates
}

func (s *Server) Start(port int) error {
	r := mux.NewRouter()
	http.Handle("/", r)
	r.Path("/registration").Methods(http.MethodGet).HandlerFunc(s.registrationInitiate)
	r.Path("/registration").Methods(http.MethodPost).HandlerFunc(s.registration)
	r.Path("/login").Methods(http.MethodGet).HandlerFunc(s.loginInitiate)
	r.Path("/login").Methods(http.MethodPost).HandlerFunc(s.login)
	r.Path("/consent").Methods(http.MethodGet).HandlerFunc(s.consent)
	r.Path("/consent").Methods(http.MethodPost).HandlerFunc(s.processConsent)
	r.Path("/logout").Methods(http.MethodGet).HandlerFunc(s.logout)
	r.Path("/").HandlerFunc(s.whoami)
	fmt.Printf("Starting HTTP server on port: %d\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func getCSRFToken(flowType, flow string, cookies []*http.Cookie) (string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Jar: jar,
	}
	b, err := url.Parse("https://accounts.lekva.me/self-service/" + flowType + "/browser")
	if err != nil {
		return "", err
	}
	client.Jar.SetCookies(b, cookies)
	resp, err := client.Get(fmt.Sprintf("https://accounts.lekva.me/self-service/"+flowType+"/flows?id=%s", flow))
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

func (s *Server) registrationInitiate(w http.ResponseWriter, r *http.Request) {
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
	if err := s.tmpls.Registration.Execute(w, csrfToken); err != nil {
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

func (s *Server) registration(w http.ResponseWriter, r *http.Request) {
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
		// TODO(giolekva): encrypt
		http.SetCookie(w, &http.Cookie{
			Name:     "login_challenge",
			Value:    challenge[0],
			HttpOnly: true,
		})
	} else {
		// http.SetCookie(w, &http.Cookie{
		// 	Name:     "login_challenge",
		// 	Value:    "",
		// 	Expires:  time.Unix(0, 0),
		// 	HttpOnly: true,
		// })
	}
	flow, ok := r.Form["flow"]
	if !ok {
		http.Redirect(w, r, s.kratos+"/self-service/login/browser", http.StatusSeeOther)
		return
	}
	csrfToken, err := getCSRFToken("login", flow[0], r.Cookies())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if err := s.tmpls.Login.Execute(w, csrfToken); err != nil {
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
	}
	b, err := url.Parse("https://accounts.lekva.me/self-service/" + flowType + "/browser")
	if err != nil {
		return nil, err
	}
	client.Jar.SetCookies(b, cookies)
	resp, err := client.Post(fmt.Sprintf("https://accounts.lekva.me/self-service/"+flowType+"?flow=%s", flow), "application/json", req)
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
	}
	b, err := url.Parse("https://accounts.lekva.me/self-service/logout/browser")
	if err != nil {
		return "", err
	}
	client.Jar.SetCookies(b, cookies)
	resp, err := client.Get("https://accounts.lekva.me/self-service/logout/browser")
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
	}
	b, err := url.Parse("https://accounts.lekva.me/sessions/whoami")
	if err != nil {
		return "", err
	}
	client.Jar.SetCookies(b, cookies)
	resp, err := client.Get("https://accounts.lekva.me/sessions/whoami")
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
	req := loginReq{
		CSRFToken: r.FormValue("csrf_token"),
		Method:    "password",
		Password:  r.FormValue("password"),
		Username:  r.FormValue("username"),
	}
	var reqBody bytes.Buffer
	if err := json.NewEncoder(&reqBody).Encode(req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := postToKratos("login", flow[0], r.Cookies(), &reqBody)
	if err == nil {
		err = extractError(resp.Body)
	}
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
	http.Redirect(w, r, "/", http.StatusSeeOther)
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
	w.Header().Set("Content-Type", "text/html")
	if err := s.tmpls.Consent.Execute(w, consent.RequestedScopes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
			"email":    username + "@lekva.me",
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
	s := &Server{
		kratos: *kratos,
		hydra:  NewHydraClient(*hydra),
		tmpls:  t,
	}
	log.Fatal(s.Start(*port))
}
