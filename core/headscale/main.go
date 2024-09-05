package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"golang.org/x/exp/rand"

	"github.com/gorilla/mux"
)

var port = flag.Int("port", 3000, "Port to listen on")
var config = flag.String("config", "", "Path to headscale config")
var acls = flag.String("acls", "", "Path to the headscale acls file")
var ipSubnet = flag.String("ip-subnet", "10.1.0.0/24", "IP subnet of the private network")
var fetchUsersAddr = flag.String("fetch-users-addr", "", "API endpoint to fetch user data")
var self = flag.String("self", "", "Self address")

// TODO(gio): make internal network cidr and proxy user configurable
const defaultACLs = `
{
  "autoApprovers": {
    "routes": {
      {{- range .cidrs }}
      "{{ . }}": ["*"],
      {{- end }}
    },
  },
  "acls": [
    {{- range .cidrs }}
    { // Everyone has passthough access to private-network-proxy node
      "action": "accept",
      "src": ["*"],
      "dst": ["{{ . }}:*", "private-network-proxy:0"],
    },
    {{- end }}
    {{- range .users }}
    { // Everyone has passthough access to private-network-proxy node
      "action": "accept",
      "src": ["{{ . }}"],
      "dst": ["{{ . }}:*"],
    },
    {{- end }}
  ],
}
`

type server struct {
	port           int
	client         *client
	fetchUsersAddr string
	self           string
	aclsPath       string
	aclsReloadPath string
	cidrs          []string
}

func newServer(port int, client *client, fetchUsersAddr, self, aclsPath string, cidrs []string) *server {
	return &server{
		port,
		client,
		fetchUsersAddr,
		self,
		aclsPath,
		fmt.Sprintf("%s-reload", aclsPath), // TODO(gio): take from the flag
		cidrs,
	}
}

func (s *server) start() error {
	f, err := os.Create(s.aclsReloadPath)
	if err != nil {
		return err
	}
	f.Close()
	r := mux.NewRouter()
	r.HandleFunc("/sync-users", s.handleSyncUsers).Methods(http.MethodGet)
	r.HandleFunc("/user/{user}/preauthkey", s.createReusablePreAuthKey).Methods(http.MethodPost)
	r.HandleFunc("/user/{user}/preauthkey", s.expireReusablePreAuthKey).Methods(http.MethodDelete)
	r.HandleFunc("/user/{user}/node/{node}/expire", s.expireUserNode).Methods(http.MethodPost)
	r.HandleFunc("/user/{user}/node/{node}", s.removeUserNode).Methods(http.MethodDelete)
	r.HandleFunc("/user", s.createUser).Methods(http.MethodPost)
	r.HandleFunc("/routes/{id}/enable", s.enableRoute).Methods(http.MethodPost)
	go func() {
		rand.Seed(uint64(time.Now().UnixNano()))
		s.syncUsers()
		for {
			delay := time.Duration(rand.Intn(60)+60) * time.Second
			time.Sleep(delay)
			s.syncUsers()
		}
	}()
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), r)
}

type createUserReq struct {
	Name string `json:"name"`
}

func (s *server) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.client.createUser(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) createReusablePreAuthKey(w http.ResponseWriter, r *http.Request) {
	user, ok := mux.Vars(r)["user"]
	if !ok {
		http.Error(w, "no user", http.StatusBadRequest)
		return
	}
	if key, err := s.client.createPreAuthKey(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		fmt.Fprint(w, key)
	}
}

type expirePreAuthKeyReq struct {
	AuthKey string `json:"authKey"`
}

func (s *server) expireReusablePreAuthKey(w http.ResponseWriter, r *http.Request) {
	user, ok := mux.Vars(r)["user"]
	if !ok {
		http.Error(w, "no user", http.StatusBadRequest)
		return
	}
	var req expirePreAuthKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.client.expirePreAuthKey(user, req.AuthKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) expireUserNode(w http.ResponseWriter, r *http.Request) {
	fmt.Println("expire node")
	user, ok := mux.Vars(r)["user"]
	if !ok {
		http.Error(w, "no user", http.StatusBadRequest)
		return
	}
	node, ok := mux.Vars(r)["node"]
	if !ok {
		http.Error(w, "no user", http.StatusBadRequest)
		return
	}
	if err := s.client.expireUserNode(user, node); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) removeUserNode(w http.ResponseWriter, r *http.Request) {
	user, ok := mux.Vars(r)["user"]
	if !ok {
		http.Error(w, "no user", http.StatusBadRequest)
		return
	}
	node, ok := mux.Vars(r)["node"]
	if !ok {
		http.Error(w, "no user", http.StatusBadRequest)
		return
	}
	if err := s.client.removeUserNode(user, node); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) handleSyncUsers(_ http.ResponseWriter, _ *http.Request) {
	go s.syncUsers()
}

type user struct {
	Username string `json:"username"`
}

func (s *server) syncUsers() {
	resp, err := http.Get(fmt.Sprintf("%s?selfAddress=%s/sync-users", s.fetchUsersAddr, s.self))
	if err != nil {
		fmt.Println(err)
		return
	}
	users := []user{}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		fmt.Println(err)
		return
	}
	var usernames []string
	for _, u := range users {
		usernames = append(usernames, u.Username)
		if err := s.client.createUser(u.Username); err != nil && !errors.Is(err, ErrorAlreadyExists) {
			fmt.Println(err)
			continue
		}
	}
	currentACLs, err := ioutil.ReadFile(s.aclsPath)
	if err != nil {
		fmt.Println(err)
	}
	newACLs, err := updateACLs(s.aclsPath, s.cidrs, usernames)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	if !bytes.Equal(currentACLs, newACLs) {
		if err := os.Remove(s.aclsReloadPath); err != nil {
			fmt.Println(err)
		}
	}
}

func (s *server) enableRoute(w http.ResponseWriter, r *http.Request) {
	id, ok := mux.Vars(r)["id"]
	if !ok {
		http.Error(w, "no id", http.StatusBadRequest)
		return
	}
	if err := s.client.enableRoute(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func updateACLs(aclsPath string, cidrs []string, users []string) ([]byte, error) {
	tmpl, err := template.New("acls").Parse(defaultACLs)
	if err != nil {
		return nil, err
	}
	out, err := os.Create(aclsPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	var ret bytes.Buffer
	if err := tmpl.Execute(io.MultiWriter(out, &ret), map[string]any{
		"cidrs": cidrs,
		"users": users,
	}); err != nil {
		return nil, err
	}
	return ret.Bytes(), nil
}

func main() {
	flag.Parse()
	var cidrs []string
	for _, ips := range strings.Split(*ipSubnet, ",") {
		_, cidr, err := net.ParseCIDR(ips)
		if err != nil {
			panic(err)
		}
		cidrs = append(cidrs, cidr.String())
	}
	c := newClient(*config)
	s := newServer(*port, c, *fetchUsersAddr, *self, *acls, cidrs)
	log.Fatal(s.start())
}
