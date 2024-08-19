package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
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
	start        = 49152
	end          = 65535
)

var port = flag.Int("port", 8080, "Port to listen on")
var repoAddr = flag.String("repo-addr", "", "Git repository address where Helm releases are stored")
var sshKey = flag.String("ssh-key", "", "Path to SHH key used to connect with Git repository")
var ingressNginxPath = flag.String("ingress-nginx-path", "", "Path to the ingress-nginx Helm release")
var minPreOpenPorts = flag.Int("min-pre-open-ports", 5, "Minimum number of pre-open ports to keep in reserve")
var preOpenPortsBatchSize = flag.Int("pre-open-ports-batch-size", 10, "Number of new ports to open at a time")

type client interface {
	ReservePort() (int, string, error)
	ReleaseReservedPort(port ...int)
	AddPortForwarding(protocol string, port int, secret, dest string) error
	RemovePortForwarding(protocol string, port int) error
}

type repoClient struct {
	l                     sync.Locker
	repo                  soft.RepoIO
	path                  string
	secretGenerator       SecretGenerator
	minPreOpenPorts       int
	preOpenPortsBatchSize int
	preOpenPorts          []int
	blocklist             map[int]struct{}
	reserve               map[int]string
	availablePorts        []int
}

func newRepoClient(
	repo soft.RepoIO,
	path string,
	minPreOpenPorts int,
	preOpenPortsBatchSize int,
	secretGenerator SecretGenerator,
) (client, error) {
	ret := &repoClient{
		l:                     &sync.Mutex{},
		repo:                  repo,
		path:                  path,
		secretGenerator:       secretGenerator,
		minPreOpenPorts:       minPreOpenPorts,
		preOpenPortsBatchSize: preOpenPortsBatchSize,
		preOpenPorts:          []int{},
		blocklist:             map[int]struct{}{},
		reserve:               map[int]string{},
		availablePorts:        []int{},
	}
	st, err := ret.readState(repo)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	} else {
		ret.preOpenPorts = st.PreOpenPorts
		ret.blocklist = st.Blocklist
		ret.reserve = st.Reserve
	}
	for i := start; i < end; i++ {
		if _, ok := ret.blocklist[i]; !ok {
			ret.availablePorts = append(ret.availablePorts, i)
		}
	}
	if err := ret.preOpenNewPorts(); err != nil {
		return nil, err
	}
	var reservedPorts []int
	for k := range ret.reserve {
		reservedPorts = append(reservedPorts, k)
	}
	go func() {
		time.Sleep(30 * time.Minute)
		ret.ReleaseReservedPort(reservedPorts...)
	}()
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
	secret, err := c.secretGenerator()
	if err != nil {
		return -1, "", err
	}
	c.reserve[port] = secret
	return port, secret, nil
}

func (c *repoClient) ReleaseReservedPort(port ...int) {
	if len(port) == 0 {
		return
	}
	c.l.Lock()
	defer c.l.Unlock()
	if _, err := c.repo.Do(func(fs soft.RepoFS) (string, error) {
		for _, p := range port {
			delete(c.reserve, p)
			c.preOpenPorts = append(c.preOpenPorts, p)
		}
		if err := c.writeState(fs); err != nil {
			return "", err
		}
		return fmt.Sprintf("Released port reservations: %+v", port), nil
	}); err != nil {
		panic(err)
	}
}

type state struct {
	PreOpenPorts []int            `json:"preOpenPorts"`
	Blocklist    map[int]struct{} `json:"blocklist"`
	Reserve      map[int]string   `json:"reserve"`
}

func (c *repoClient) preOpenNewPorts() error {
	c.l.Lock()
	defer c.l.Unlock()
	if len(c.preOpenPorts) >= c.minPreOpenPorts {
		return nil
	}
	var ports []int
	for count := c.preOpenPortsBatchSize; count > 0; count-- {
		if len(c.availablePorts) == 0 {
			return fmt.Errorf("could not open new port")
		}
		r := rand.Intn(len(c.availablePorts))
		p := c.availablePorts[r]
		c.availablePorts[r] = c.availablePorts[len(c.availablePorts)-1]
		c.availablePorts = c.availablePorts[:len(c.availablePorts)-1]
		ports = append(ports, p)
		c.preOpenPorts = append(c.preOpenPorts, p)
		c.blocklist[p] = struct{}{}
	}
	_, err := c.repo.Do(func(fs soft.RepoFS) (string, error) {
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
			fmt.Printf("%+v\n", tcp)
			fmt.Printf("%+v\n", udp)
			for _, p := range ports {
				ps := strconv.Itoa(p)
				tcp[ps] = p
				udp[ps] = p
			}
			fmt.Printf("%+v\n", tcp)
			fmt.Printf("%+v\n", udp)
			if err := c.writeRelease(fs, rel); err != nil {
				return "", err
			}
		}
		fmt.Printf("Pre opened new ports: %+v\n", ports)
		return "preopen new ports", nil
	})
	return err
}

func (c *repoClient) AddPortForwarding(protocol string, port int, secret, dest string) error {
	protocol = strings.ToLower(protocol)
	defer func() {
		if err := c.preOpenNewPorts(); err != nil {
			panic(err)
		}
	}()
	c.l.Lock()
	defer c.l.Unlock()
	if sec, ok := c.reserve[port]; !ok || sec != secret {
		return fmt.Errorf("wrong secret")
	}
	delete(c.reserve, port)
	_, err := c.repo.Do(func(fs soft.RepoFS) (string, error) {
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
	return err
}

func (c *repoClient) RemovePortForwarding(protocol string, port int) error {
	protocol = strings.ToLower(protocol)
	c.l.Lock()
	defer c.l.Unlock()
	_, err := c.repo.Do(func(fs soft.RepoFS) (string, error) {
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
	return err
}

func (c *repoClient) readState(fs soft.RepoFS) (state, error) {
	r, err := fs.Reader(fmt.Sprintf("%s-state.json", c.path))
	if err != nil {
		return state{}, err
	}
	defer r.Close()
	var ret state
	if err := json.NewDecoder(r).Decode(&ret); err != nil {
		return state{}, err
	}
	return ret, err
}

func (c *repoClient) writeState(fs soft.RepoFS) error {
	w, err := fs.Writer(fmt.Sprintf("%s-state.json", c.path))
	if err != nil {
		return err
	}
	defer w.Close()
	if err := json.NewEncoder(w).Encode(state{c.preOpenPorts, c.blocklist, c.reserve}); err != nil {
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

type SecretGenerator func() (string, error)

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
		generateSecret,
	)
	if err != nil {
		log.Fatal(err)
	}
	s := newServer(*port, c)
	log.Fatal(s.Start())
}
