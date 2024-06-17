package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/giolekva/pcloud/core/installer/soft"

	"golang.org/x/crypto/ssh"
)

const (
	secretLength = 20
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
	repo soft.RepoIO
	path string
}

func (c *repoClient) ReadRelease() (map[string]any, error) {
	if err := c.repo.Pull(); err != nil {
		return nil, err
	}
	ingressRel := map[string]any{}
	if err := soft.ReadYaml(c.repo, c.path, &ingressRel); err != nil {
		return nil, err
	}
	return ingressRel, nil
}

func (c *repoClient) WriteRelease(rel map[string]any, meta string) error {
	if err := soft.WriteYaml(c.repo, c.path, rel); err != nil {
		return err
	}
	return c.repo.CommitAndPush(meta)
}

type server struct {
	l       sync.Locker
	s       *http.Server
	r       *http.ServeMux
	client  client
	reserve map[int]string
}

func newServer(port int, client client) *server {
	r := http.NewServeMux()
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	return &server{&sync.Mutex{}, s, r, client, make(map[int]string)}
}

func (s *server) Start() error {
	s.r.HandleFunc("/api/reserve", s.handleReserve)
	s.r.HandleFunc("/api/allocate", s.handleAllocate)
	s.r.HandleFunc("/api/remove", s.handleRemove)
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
	Secret        string `json:"secret"`
}

type removeReq struct {
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

func extractRemoveReq(r io.Reader) (removeReq, error) {
	var req removeReq
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return removeReq{}, err
	}
	req.Protocol = strings.ToLower(req.Protocol)
	if req.Protocol != "tcp" && req.Protocol != "udp" {
		return removeReq{}, fmt.Errorf("Unexpected protocol %s", req.Protocol)
	}
	return req, nil
}

type reserveResp struct {
	Port   int    `json:"port"`
	Secret string `json:"secret"`
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
	if _, ok := pm[sourcePortStr]; ok || req.SourcePort == 80 || req.SourcePort == 443 || req.SourcePort == 22 {
		return fmt.Errorf("port %d is already taken", req.SourcePort)
	}
	pm[sourcePortStr] = fmt.Sprintf("%s:%d", req.TargetService, req.TargetPort)
	return nil
}

func removePort(pm map[string]any, req removeReq) error {
	sourcePortStr := strconv.Itoa(req.SourcePort)
	if _, ok := pm[sourcePortStr]; !ok {
		return fmt.Errorf("port %d is not open to remove", req.SourcePort)
	}
	delete(pm, sourcePortStr)
	return nil
}

const start = 49152
const end = 65535

func reservePort(pm map[string]struct{}, reserve map[int]string) (int, error) {
	for i := 0; i < 3; i++ {
		r, err := rand.Int(rand.Reader, big.NewInt(end-start))
		if err != nil {
			return -1, err
		}
		p := start + int(r.Int64())
		ps := strconv.Itoa(p)
		if _, ok := pm[ps]; !ok {
			return p, nil
		}
	}
	return -1, fmt.Errorf("could not generate random port")
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
	s.l.Lock()
	defer s.l.Unlock()
	ingressRel, err := s.client.ReadRelease()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tcp, udp, err := extractPorts(ingressRel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if val, ok := s.reserve[req.SourcePort]; !ok || val != req.Secret {
		http.Error(w, "invalid secret", http.StatusBadRequest)
		return
	} else {
		delete(s.reserve, req.SourcePort)
	}
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

func (s *server) handleReserve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only post method is supported", http.StatusBadRequest)
		return
	}
	s.l.Lock()
	defer s.l.Unlock()
	ingressRel, err := s.client.ReadRelease()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tcp, udp, err := extractPorts(ingressRel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var port int
	used := map[string]struct{}{}
	for p, _ := range tcp {
		used[p] = struct{}{}
	}
	for p, _ := range udp {
		used[p] = struct{}{}
	}
	if port, err = reservePort(used, s.reserve); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	secret, err := generateSecret()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.reserve[port] = secret
	go func() {
		time.Sleep(30 * time.Minute)
		s.l.Lock()
		defer s.l.Unlock()
		delete(s.reserve, port)
	}()
	resp := reserveResp{port, secret}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) handleRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only post method is supported", http.StatusBadRequest)
		return
	}
	req, err := extractRemoveReq(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.l.Lock()
	defer s.l.Unlock()
	ingressRel, err := s.client.ReadRelease()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tcp, udp, err := extractPorts(ingressRel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	switch req.Protocol {
	case "tcp":
		if err := removePort(tcp, req); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	case "udp":
		if err := removePort(udp, req); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	default:
		panic("MUST NOT REACH")
	}
	commitMsg := fmt.Sprintf("ingress: remove port map %d %s", req.SourcePort, req.Protocol)
	if err := s.client.WriteRelease(ingressRel, commitMsg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	delete(s.reserve, req.SourcePort)
}

// TODO(gio): deduplicate
func createRepoClient(addr string, keyPath string) (soft.RepoIO, error) {
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
	return soft.NewRepoIO(repo, signer)
}

func generateSecret() (string, error) {
	b := make([]byte, secretLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("error generating secret: %v", err)
	}
	return string(b), nil
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
