package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
)

var port = flag.Int("port", 3000, "Port to listen on")
var config = flag.String("config", "", "Path to headscale config")
var acls = flag.String("acls", "", "Path to the headscale acls file")
var ipSubnet = flag.String("ip-subnet", "10.1.0.0/24", "IP subnet of the private network")

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
  ],
}
`

type server struct {
	port   int
	client *client
}

func newServer(port int, client *client) *server {
	return &server{
		port,
		client,
	}
}

func (s *server) start() error {
	r := mux.NewRouter()
	r.HandleFunc("/user/{user}/preauthkey", s.createReusablePreAuthKey).Methods(http.MethodPost)
	r.HandleFunc("/user", s.createUser).Methods(http.MethodPost)
	r.HandleFunc("/routes/{id}/enable", s.enableRoute).Methods(http.MethodPost)
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

func updateACLs(cidrs []string, aclsPath string) error {
	tmpl, err := template.New("acls").Parse(defaultACLs)
	if err != nil {
		return err
	}
	out, err := os.Create(aclsPath)
	if err != nil {
		return err
	}
	defer out.Close()
	tmpl.Execute(os.Stdout, map[string]any{
		"cidrs": cidrs,
	})
	return tmpl.Execute(out, map[string]any{
		"cidrs": cidrs,
	})
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
	updateACLs(cidrs, *acls)
	c := newClient(*config)
	s := newServer(*port, c)
	log.Fatal(s.start())
}
