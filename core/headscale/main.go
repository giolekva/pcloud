package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"text/template"

	"github.com/labstack/echo/v4"
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
      "{{ .ipSubnet }}": ["*"],
    },
  },
  "acls": [
    { // Everyone has passthough access to private-network-proxy node
      "action": "accept",
      "src": ["*"],
      "dst": ["{{ .ipSubnet }}:*", "private-network-proxy:0"],
    },
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

func (s *server) start() {
	e := echo.New()
	e.POST("/user/:user/preauthkey", s.createReusablePreAuthKey)
	e.POST("/user", s.createUser)
	e.POST("/routes/:id/enable", s.enableRoute)
	log.Fatal(e.Start(fmt.Sprintf(":%d", s.port)))
}

type createUserReq struct {
	Name string `json:"name"`
}

func (s *server) createUser(c echo.Context) error {
	var req createUserReq
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return err
	}
	if err := s.client.createUser(req.Name); err != nil {
		return err
	} else {
		return c.String(http.StatusOK, "")
	}
}

func (s *server) createReusablePreAuthKey(c echo.Context) error {
	if key, err := s.client.createPreAuthKey(c.Param("user")); err != nil {
		return err
	} else {
		return c.String(http.StatusOK, key)
	}
}

func (s *server) enableRoute(c echo.Context) error {
	if err := s.client.enableRoute(c.Param("id")); err != nil {
		return err
	} else {
		return c.String(http.StatusOK, "")
	}
}

func updateACLs(cidr net.IPNet, aclsPath string) error {
	tmpl, err := template.New("acls").Parse(defaultACLs)
	if err != nil {
		return err
	}
	out, err := os.Create(aclsPath)
	if err != nil {
		return err
	}
	defer out.Close()
	return tmpl.Execute(out, map[string]any{
		"ipSubnet": cidr.String(),
	})
}

func main() {
	flag.Parse()
	_, cidr, err := net.ParseCIDR(*ipSubnet)
	if err != nil {
		panic(err)
	}
	updateACLs(*cidr, *acls)
	c := newClient(*config)
	s := newServer(*port, c)
	s.start()
}
