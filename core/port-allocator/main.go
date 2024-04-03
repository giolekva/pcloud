package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"

	"golang.org/x/crypto/ssh"
)

var port = flag.Int("port", 8080, "Port to listen on")
var repoAddr = flag.String("repo-addr", "", "Git repository address where Helm releases are stored")
var sshKey = flag.String("ssh-key", "", "Path to SHH key used to connect with Git repository")
var ingressNginxPath = flag.String("ingress-nginx-path", "", "Path to the ingress-nginx Helm release")

type client interface {
	ReadRelease() (map[string]any, error)
	WriteRelease(rel map[string]any, meta string) error
}

type repoClient struct {
	repo installer.RepoIO
	path string
}

func (c *repoClient) ReadRelease() (map[string]any, error) {
	if err := c.repo.Pull(); err != nil {
		return nil, err
	}
	rel, err := c.repo.ReadYaml(c.path)
	if err != nil {
		return nil, err
	}
	ingressRel, ok := rel.(map[string]any)
	if !ok {
		panic("MUST NOT REACH!")
	}
	return ingressRel, nil
}

func (c *repoClient) WriteRelease(rel map[string]any, meta string) error {
	if err := c.repo.WriteYaml(c.path, rel); err != nil {
		return err
	}
	return c.repo.CommitAndPush(meta)
}

type server struct {
	s      *http.Server
	r      *http.ServeMux
	client client
}

func newServer(port int, client client) *server {
	r := http.NewServeMux()
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	return &server{s, r, client}
}

func (s *server) Start() error {
	s.r.HandleFunc("/api/allocate", s.handleAllocate)
	if err := s.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *server) Close() error {
	return s.s.Close()
}

type allocateReq struct {
	Protocol      string `json:"protocol"`
	SourcePort    int    `json:"sourcePort"`
	TargetService string `json:"targetService"`
	TargetPort    int    `json:"targetPort"`
}

func extractAllocateReq(r io.Reader) (allocateReq, error) {
	var req allocateReq
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return allocateReq{}, err
	}
	req.Protocol = strings.ToLower(req.Protocol)
	if req.Protocol != "tcp" && req.Protocol != "udp" {
		return allocateReq{}, fmt.Errorf("Unexpected protocol %s", req.Protocol)
	}
	return req, nil
}

func extractPorts(rel map[string]any) (map[string]any, map[string]any, error) {
	spec, ok := rel["spec"]
	if !ok {
		return nil, nil, fmt.Errorf("spec not found")
	}
	specM, ok := spec.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("spec is not a map")
	}
	values, ok := specM["values"]
	if !ok {
		return nil, nil, fmt.Errorf("spec.values not found")
	}
	valuesM, ok := values.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("spec.values is not a map")
	}
	tcp, ok := valuesM["tcp"]
	if !ok {
		tcp = map[string]any{}
		valuesM["tcp"] = tcp
	}
	udp, ok := valuesM["udp"]
	if !ok {
		udp = map[string]any{}
		valuesM["udp"] = udp
	}
	tcpM, ok := tcp.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("spec.values.tcp is not a map")
	}
	udpM, ok := udp.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("spec.values.udp is not a map")
	}
	return tcpM, udpM, nil
}

func addPort(pm map[string]any, req allocateReq) error {
	sourcePortStr := strconv.Itoa(req.SourcePort)
	if _, ok := pm[sourcePortStr]; ok || req.SourcePort == 80 || req.SourcePort == 443 {
		return fmt.Errorf("port %d is already taken", req.SourcePort)
	}
	pm[sourcePortStr] = fmt.Sprintf("%s:%d", req.TargetService, req.TargetPort)
	return nil
}

func (s *server) handleAllocate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only post method is supported", http.StatusBadRequest)
		return
	}
	req, err := extractAllocateReq(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("%+v\n", req)
	ingressRel, err := s.client.ReadRelease()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("%+v\n", ingressRel)
	tcp, udp, err := extractPorts(ingressRel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("%+v %+v\n", tcp, udp)
	switch req.Protocol {
	case "tcp":
		if err := addPort(tcp, req); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	case "udp":
		if err := addPort(udp, req); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	default:
		panic("MUST NOT REACH")
	}
	commitMsg := fmt.Sprintf("ingress: port map %d %s", req.SourcePort, req.Protocol)
	if err := s.client.WriteRelease(ingressRel, commitMsg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// TODO(gio): deduplicate
func createRepoClient(addr string, keyPath string) (installer.RepoIO, error) {
	sshKey, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(sshKey)
	if err != nil {
		return nil, err
	}
	repoAddr, err := soft.ParseRepositoryAddress(addr)
	if err != nil {
		return nil, err
	}
	repo, err := soft.CloneRepository(repoAddr, signer)
	if err != nil {
		return nil, err
	}
	return installer.NewRepoIO(repo, signer), nil
}

func main() {
	flag.Parse()
	repo, err := createRepoClient(*repoAddr, *sshKey)
	if err != nil {
		log.Fatal(err)
	}
	s := newServer(
		*port,
		&repoClient{repo, *ingressNginxPath},
	)
	log.Fatal(s.Start())
}
