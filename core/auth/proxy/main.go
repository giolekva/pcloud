package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

var port = flag.Int("port", 3000, "Port to listen on")
var whoAmIAddr = flag.String("auth-addr", "", "Kratos whoami endpoint address")
var sessionCookie = flag.String("session-cookie", "session", "Name of the session cookie")
var upstream = flag.String("upstream", "", "Upstream service address")

type user struct {
	Identity struct {
		Traits struct {
			Username string `json:"username"`
		} `json:"traits"`
	} `json:"identity"`
}

type authError struct {
	Error struct {
		Status string `json:"status"`
	} `json:"error"`
}

func getAddr(r *http.Request) *url.URL {
	return &url.URL{
		Scheme:      r.Header["X-Forwarded-Scheme"][0],
		Host:        r.Header["X-Forwarded-Host"][0],
		RawPath:     r.URL.RawPath,
		RawQuery:    r.URL.RawQuery,
		RawFragment: r.URL.RawFragment,
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	user, err := queryWhoAmI(r.Cookies())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if user == nil {
		if r.Method != http.MethodGet {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		addr := fmt.Sprintf("https://accounts-ui.t12.dodo.cloud/login?return_to=%s", getAddr(r).String())
		http.Redirect(w, r, addr, http.StatusSeeOther)
		return
	}
	rc := r.Clone(context.Background())
	rc.Header.Add("X-User", user.Identity.Traits.Username)
	ru := url.URL{
		Scheme:      "http",
		Host:        *upstream,
		RawPath:     r.URL.RawPath,
		RawQuery:    r.URL.RawQuery,
		RawFragment: r.URL.RawFragment,
		// Path:        r.URL.Path,
		// Query:       r.URL.Query,
		// Fragment:    r.URL.Fragment,
	}
	rc.URL = &ru
	rc.RequestURI = ""
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	if resp, err := client.Do(rc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if _, err := io.Copy(w, resp.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func queryWhoAmI(cookies []*http.Cookie) (*user, error) {
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
	addr, err := url.Parse(*whoAmIAddr)
	if err != nil {
		return nil, err
	}
	client.Jar.SetCookies(addr, cookies)
	resp, err := client.Get(*whoAmIAddr)
	if err != nil {
		return nil, err
	}
	data := make(map[string]any)
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	// TODO(gio): remove debugging
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	fmt.Println(string(b))
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return nil, err
	}
	tmp := buf.String()
	if resp.StatusCode == http.StatusOK {
		u := &user{}
		if err := json.NewDecoder(strings.NewReader(tmp)).Decode(u); err != nil {
			return nil, err
		}
		return u, nil
	}
	e := &authError{}
	if err := json.NewDecoder(strings.NewReader(tmp)).Decode(e); err != nil {
		return nil, err
	}
	if e.Error.Status == "Unauthorized" {
		return nil, nil
	}
	return nil, fmt.Errorf("Unknown error: %s", tmp)
}

func main() {
	flag.Parse()
	http.HandleFunc("/", handle)
	fmt.Printf("Starting HTTP server on port: %d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
