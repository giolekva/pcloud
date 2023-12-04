package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/labstack/echo/v4"
)

var port = flag.Int("port", 3000, "Port to listen on")
var config = flag.String("config", "", "Path to headscale config")
var acls = flag.String("acls", "", "Path to the headscale acls file")
var domain = flag.String("domain", "", "Environment domain")

// TODO(gio): ingress-private user name must be configurable
const defaultACLs = `
{
  "hosts": {
    "private-network": "10.1.0.0/24",
  },
  "autoApprovers": {
    "routes": {
      "private-network": ["private-network-proxy@{{ .Domain }}"],
    },
  },
  "acls": [
    { // Everyone can access ingress-private service
      "action": "accept",
      "src": ["*"],
      "dst": ["private-network:*"],
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

func updateACLs(domain, acls string) error {
	tmpl, err := template.New("acls").Parse(defaultACLs)
	if err != nil {
		return err
	}
	out, err := os.Create(acls)
	if err != nil {
		return err
	}
	defer out.Close()
	return tmpl.Execute(out, map[string]any{
		"Domain": domain,
	})
}

func main() {
	flag.Parse()
	updateACLs(*domain, *acls)
	c := newClient(*config)
	s := newServer(*port, c)
	s.start()
}
