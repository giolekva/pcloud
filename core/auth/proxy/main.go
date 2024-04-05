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

	"golang.org/x/exp/slices"
)

var port = flag.Int("port", 3000, "Port to listen on")
var whoAmIAddr = flag.String("whoami-addr", "", "Kratos whoami endpoint address")
var loginAddr = flag.String("login-addr", "", "Login page address")
var membershipAddr = flag.String("membership-addr", "", "Group membership API endpoint")
var groups = flag.String("groups", "", "Comma separated list of groups. User must be part of at least one of them. If empty group membership will not be checked.")
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

func getAddr(r *http.Request) (*url.URL, error) {
	return url.Parse(fmt.Sprintf(
		"%s://%s%s",
		r.Header["X-Forwarded-Scheme"][0],
		r.Header["X-Forwarded-Host"][0],
		r.URL.RequestURI()))
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
		curr, err := getAddr(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		addr := fmt.Sprintf("%s?return_to=%s", *loginAddr, curr.String())
		http.Redirect(w, r, addr, http.StatusSeeOther)
		return
	}
	if *groups != "" {
		hasPermission := false
		tg, err := getTransitiveGroups(user.Identity.Traits.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, i := range strings.Split(*groups, ",") {
			if slices.Contains(tg, strings.TrimSpace(i)) {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

	}
	rc := r.Clone(context.Background())
	rc.Header.Add("X-User", user.Identity.Traits.Username)
	rc.Header.Add("X-Forwarded-User", user.Identity.Traits.Username)
	ru, err := url.Parse(fmt.Sprintf("http://%s%s", *upstream, r.URL.RequestURI()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rc.URL = ru
	rc.RequestURI = ""
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(rc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
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

type MembershipInfo struct {
	MemberOf []string `json:"memberOf"`
}

func getTransitiveGroups(user string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%s", *membershipAddr, user))
	if err != nil {
		return nil, err
	}
	var info MembershipInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return info.MemberOf, nil
}

func main() {
	flag.Parse()
	if *groups != "" && *membershipAddr == "" {
		log.Fatal("membership-addr flag is required when groups are provided")
	}
	http.HandleFunc("/", handle)
	fmt.Printf("Starting HTTP server on port: %d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
