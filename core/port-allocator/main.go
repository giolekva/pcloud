package main

import (
	"crypto/rand"
	"encoding/base64"
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
var minPreOpenPorts = flag.Int("min-pre-open-ports", 5, "Minimum number of pre-open ports to keep in reserve")
var preOpenPortsBatchSize = flag.Int("pre-open-ports-batch-size", 10, "Number of new ports to open at a time")

type client interface {
	ReservePort() (int, string, error)
	ReleaseReservedPort(port int)
	AddPortForwarding(protocol string, port int, secret, dest string) error
	RemovePortForwarding(protocol string, port int) error
}

type repoClient struct {
	l                     sync.Locker
	repo                  soft.RepoIO
	path                  string
	minPreOpenPorts       int
	preOpenPortsBatchSize int
	preOpenPorts          []int
	blocklist             map[int]struct{}
	reserve               map[int]string
}

func newRepoClient(
	repo soft.RepoIO,
	path string,
	minPreOpenPorts int,
	preOpenPortsBatchSize int,
) (client, error) {
	ret := &repoClient{
		l:                     &sync.Mutex{},
		repo:                  repo,
		path:                  path,
		minPreOpenPorts:       minPreOpenPorts,
		preOpenPortsBatchSize: preOpenPortsBatchSize,
	}
	r, err := repo.Reader(fmt.Sprintf("%s-state.json", path))
	if err != nil {
		// TODO(gio): create empty file on init
		return nil, err
	}
	defer r.Close()
	var st state
	if err := json.NewDecoder(r).Decode(&st); err != nil {
		return nil, err
	}
	ret.preOpenPorts = st.PreOpenPorts
	ret.blocklist = st.Blocklist
	ret.reserve = map[int]string{}
	if len(ret.preOpenPorts) < minPreOpenPorts {
		if err := ret.preOpenNewPorts(); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (c *repoClient) ReservePort() (int, string, error) {
	c.l.Lock()
	defer c.l.Unlock()
	if len(c.preOpenPorts) == 0 {
		return -1, "", fmt.Errorf("no pre-open ports are available")
	}
	port := c.preOpenPorts[0]
	c.preOpenPorts = c.preOpenPorts[1:]
	secret, err := generateSecret()
	if err != nil {
		return -1, "", err
	}
	c.reserve[port] = secret
	return port, secret, nil
}

func (c *repoClient) ReleaseReservedPort(port int) {
	c.l.Lock()
	defer c.l.Unlock()
	delete(c.reserve, port)
	c.preOpenPorts = append(c.preOpenPorts, port)
}

type state struct {
	PreOpenPorts []int            `json:"preOpenPorts"`
	Blocklist    map[int]struct{} `json:"blocklist"`
}

func (c *repoClient) preOpenNewPorts() error {
	c.l.Lock()
	defer c.l.Unlock()
	if len(c.preOpenPorts) >= c.minPreOpenPorts {
		return nil
	}
	var ports []int
	for count := c.preOpenPortsBatchSize; count > 0; count-- {
		generated := false
		for i := 0; i < 3; i++ {
			r, err := rand.Int(rand.Reader, big.NewInt(end-start))
			if err != nil {
				return err
			}
			p := start + int(r.Int64())
			if _, ok := c.blocklist[p]; !ok {
				generated = true
				ports = append(ports, p)
				c.preOpenPorts = append(c.preOpenPorts, p)
				c.blocklist[p] = struct{}{}
				break
			}
		}
		if !generated {
			return fmt.Errorf("could not open new port")
		}
	}
	return c.repo.Do(func(fs soft.RepoFS) (string, error) {
		if err := c.writeState(fs); err != nil {
			return "", err
		}
		rel, err := c.readRelease(fs)
		if err != nil {
			return "", err
		}
		svcType, err := extractString(rel, "spec.values.controller.service.type")
		if err != nil {
			return "", err
		}
		if svcType == "NodePort" {
			tcp, err := extractPorts(rel, "spec.values.controller.service.nodePorts.tcp")
			if err != nil {
				return "", err
			}
			udp, err := extractPorts(rel, "spec.values.controller.service.nodePorts.udp")
			if err != nil {
				return "", err
			}
			for _, p := range ports {
				ps := strconv.Itoa(p)
				tcp[ps] = p
				udp[ps] = p
			}
			if err := c.writeRelease(fs, rel); err != nil {
				return "", err
			}
		}
		fmt.Printf("Pre opened new ports: %+v\n", ports)
		return "preopen new ports", nil
	})
}

func (c *repoClient) AddPortForwarding(protocol string, port int, secret, dest string) error {
	defer func() {
		go func() {
			if err := c.preOpenNewPorts(); err != nil {
				panic(err)
			}
		}()
	}()
	c.l.Lock()
	defer c.l.Unlock()
	if sec, ok := c.reserve[port]; !ok || sec != secret {
		return fmt.Errorf("wrong secret")
	}
	delete(c.reserve, port)
	return c.repo.Do(func(fs soft.RepoFS) (string, error) {
		if err := c.writeState(fs); err != nil {
			return "", err
		}
		rel, err := c.readRelease(fs)
		if err != nil {
			return "", err
		}
		portStr := strconv.Itoa(port)
		switch protocol {
		case "tcp":
			tcp, err := extractPorts(rel, "spec.values.tcp")
			if err != nil {
				return "", err
			}
			tcp[portStr] = dest
		case "udp":
			udp, err := extractPorts(rel, "spec.values.udp")
			if err != nil {
				return "", err
			}
			udp[portStr] = dest
		default:
			panic("MUST NOT REACH")
		}
		if err := c.writeRelease(fs, rel); err != nil {
			return "", err
		}
		return fmt.Sprintf("ingress: port %s map %d %s", protocol, port, dest), nil
	})
}

func (c *repoClient) RemovePortForwarding(protocol string, port int) error {
	c.l.Lock()
	defer c.l.Unlock()
	return c.repo.Do(func(fs soft.RepoFS) (string, error) {
		rel, err := c.readRelease(fs)
		if err != nil {
			return "", err
		}
		switch protocol {
		case "tcp":
			tcp, err := extractPorts(rel, "spec.values.tcp")
			if err != nil {
				return "", err
			}
			if err := removePort(tcp, port); err != nil {
				return "", err
			}
		case "udp":
			udp, err := extractPorts(rel, "spec.values.udp")
			if err != nil {
				return "", err
			}
			if err := removePort(udp, port); err != nil {
				return "", err
			}
		default:
			panic("MUST NOT REACH")
		}
		svcType, err := extractString(rel, "spec.values.controller.service.type")
		if err != nil {
			return "", err
		}
		if svcType == "NodePort" {
			svcTCP, err := extractPorts(rel, "spec.values.controller.service.nodePorts.tcp")
			if err != nil {
				return "", err
			}
			svcUDP, err := extractPorts(rel, "spec.values.controller.service.nodePorts.udp")
			if err != nil {
				return "", err
			}
			if err := removePort(svcTCP, port); err != nil {
				return "", err
			}
			if err := removePort(svcUDP, port); err != nil {
				return "", err
			}
		}
		if err := c.writeRelease(fs, rel); err != nil {
			return "", err
		}
		return fmt.Sprintf("ingress: remove %s port map %d", protocol, port), nil
	})
}

func (c *repoClient) writeState(fs soft.RepoFS) error {
	w, err := fs.Writer(fmt.Sprintf("%s-state.json", c.path))
	if err != nil {
		return err
	}
	defer w.Close()
	if err := json.NewEncoder(w).Encode(state{c.preOpenPorts, c.blocklist}); err != nil {
		return err
	}
	return err
}

func (c *repoClient) readRelease(fs soft.RepoFS) (map[string]any, error) {
	ret := map[string]any{}
	if err := soft.ReadYaml(fs, c.path, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *repoClient) writeRelease(fs soft.RepoFS, rel map[string]any) error {
	return soft.WriteYaml(fs, c.path, rel)
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

func extractField(data map[string]any, path string) (any, error) {
	var val any = data
	for _, i := range strings.Split(path, ".") {
		valM, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map")
		}
		val, ok = valM[i]
		if !ok {
			return nil, fmt.Errorf("%s not found", i)
		}
	}
	return val, nil
}

func extractPorts(data map[string]any, path string) (map[string]any, error) {
	ret, err := extractField(data, path)
	if err != nil {
		return nil, err
	}
	retM, ok := ret.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map")
	}
	return retM, nil
}

func extractString(data map[string]any, path string) (string, error) {
	ret, err := extractField(data, path)
	if err != nil {
		return "", err
	}
	retS, ok := ret.(string)
	if !ok {
		return "", fmt.Errorf("expected map")
	}
	return retS, nil
}

func addPort(pm map[string]any, sourcePort int, targetService string, targetPort int) error {
	sourcePortStr := strconv.Itoa(sourcePort)
	if _, ok := pm[sourcePortStr]; ok || sourcePort == 80 || sourcePort == 443 || sourcePort == 22 {
		return fmt.Errorf("port %d is already taken", sourcePort)
	}
	pm[sourcePortStr] = fmt.Sprintf("%s:%d", targetService, targetPort)
	return nil
}

func removePort(pm map[string]any, port int) error {
	sourcePortStr := strconv.Itoa(port)
	if _, ok := pm[sourcePortStr]; !ok {
		return fmt.Errorf("port %d is not open to remove", port)
	}
	delete(pm, sourcePortStr)
	return nil
}

const start = 49152
const end = 65535

func updateNodePorts(rel map[string]any, protocol string, pm map[string]any) error {
	spec, ok := rel["spec"]
	if !ok {
		return fmt.Errorf("spec not found")
	}
	specM, ok := spec.(map[string]any)
	if !ok {
		return fmt.Errorf("spec is not a map")
	}
	values, ok := specM["values"]
	if !ok {
		return fmt.Errorf("spec.values not found")
	}
	valuesM, ok := values.(map[string]any)
	if !ok {
		return fmt.Errorf("spec.values is not a map")
	}
	controller, ok := valuesM["controller"]
	if !ok {
		return fmt.Errorf("spec.values.controller not found")
	}
	controllerM, ok := controller.(map[string]any)
	if !ok {
		return fmt.Errorf("spec.values.controller is not a map")
	}
	service, ok := controllerM["service"]
	if !ok {
		return fmt.Errorf("spec.values.controller.service not found")
	}
	serviceM, ok := service.(map[string]any)
	if !ok {
		return fmt.Errorf("spec.values.controller.service is not a map")
	}
	nodePorts, ok := serviceM["nodePorts"]
	if !ok {
		return fmt.Errorf("spec.values.controller.service.nodePorts not found")
	}
	nodePortsM, ok := nodePorts.(map[string]any)
	if !ok {
		return fmt.Errorf("spec.values.controller.service.nodePorts is not a map")
	}
	npm := map[string]any{}
	for p, _ := range pm {
		if v, err := strconv.Atoi(p); err != nil {
			return err
		} else {
			npm[p] = v
		}
	}
	nodePortsM[protocol] = npm
	return nil
}

func (s *server) handleAllocate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only post method is supported", http.StatusBadRequest)
		return
	}
	req, err := extractAllocateReq(r.Body)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.client.AddPortForwarding(
		req.Protocol,
		req.SourcePort,
		req.Secret,
		fmt.Sprintf("%s:%d", req.TargetService, req.TargetPort),
	); err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) handleReserve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only post method is supported", http.StatusBadRequest)
		return
	}
	var port int
	var secret string
	var err error
	if port, secret, err = s.client.ReservePort(); err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	go func() {
		time.Sleep(30 * time.Minute)
		s.client.ReleaseReservedPort(port)
	}()
	if err := json.NewEncoder(w).Encode(reserveResp{port, secret}); err != nil {
		fmt.Println(err.Error())
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
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.client.RemovePortForwarding(req.Protocol, req.SourcePort); err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	return base64.StdEncoding.EncodeToString(b), nil
}

func main() {
	flag.Parse()
	repo, err := createRepoClient(*repoAddr, *sshKey)
	if err != nil {
		log.Fatal(err)
	}
	c, err := newRepoClient(
		repo,
		*ingressNginxPath,
		*minPreOpenPorts,
		*preOpenPortsBatchSize,
	)
	if err != nil {
		log.Fatal(err)
	}
	s := newServer(*port, c)
	log.Fatal(s.Start())
}
