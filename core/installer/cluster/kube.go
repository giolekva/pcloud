package cluster

import (
	"context"
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
	"sync"
	"time"

	"github.com/giolekva/pcloud/core/installer/kube"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/drain"
)

type KubeManager struct {
	l                sync.Locker
	name             string
	ingressClassName string
	ingressIP        net.IP
	kubeCfg          string
	serverAddr       string
	serverToken      string
	controllers      []Server
	workers          []Server
	storageEnabled   bool
}

func NewKubeManager() *KubeManager {
	return &KubeManager{l: &sync.Mutex{}}
}

func RestoreKubeManager(st State) (*KubeManager, error) {
	return &KubeManager{
		l:                &sync.Mutex{},
		name:             st.Name,
		ingressClassName: st.IngressClassName,
		ingressIP:        st.IngressIP,
		kubeCfg:          st.Kubeconfig,
		serverAddr:       st.ServerAddr,
		serverToken:      st.ServerToken,
		controllers:      st.Controllers,
		workers:          st.Workers,
		storageEnabled:   st.StorageEnabled,
	}, nil
}

func (m *KubeManager) State() State {
	m.l.Lock()
	defer m.l.Unlock()
	return State{
		m.name,
		m.ingressClassName,
		m.ingressIP,
		m.serverAddr,
		m.serverToken,
		m.kubeCfg,
		m.controllers,
		m.workers,
		m.storageEnabled,
	}
}

func (m *KubeManager) EnableStorage() {
	m.storageEnabled = true
}

func (m *KubeManager) Init(s Server, setupFn ClusterIngressSetupFunc) (net.IP, error) {
	m.l.Lock()
	defer m.l.Unlock()
	if m.kubeCfg != "" {
		return nil, fmt.Errorf("already initialized")
	}
	c, err := m.connect(&s)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if err := InstallTailscale(c); err != nil {
		return nil, err
	}
	const loginServer = "https://headscale.v1.dodo.cloud"
	if err := TailscaleUp(c, loginServer, s.Name, s.AuthKey); err != nil {
		return nil, err
	}
	if err := InstallK3s(c); err != nil {
		return nil, err
	}
	kubeCfg, err := GetKubeconfig(c)
	if err != nil {
		return nil, err
	}
	m.kubeCfg = kubeCfg
	serverIP, err := GetTailscaleIP(c)
	if err != nil {
		return nil, err
	}
	m.serverAddr = fmt.Sprintf("%s:6443", serverIP)
	serverToken, err := GetServerToken(c)
	if err != nil {
		return nil, err
	}
	m.serverToken = serverToken
	m.controllers = []Server{s}
	m.ingressClassName = "default"
	ingressIP, err := setupFn(m.name, m.kubeCfg, m.ingressClassName)
	if err != nil {
		return nil, err
	}
	m.ingressIP = ingressIP
	return ingressIP, nil
}

func (m *KubeManager) JoinController(s Server) error {
	m.l.Lock()
	defer m.l.Unlock()
	if m.kubeCfg == "" {
		return fmt.Errorf("not initialized")
	}
	if i := m.findServerByIP(s.IP); i != nil {
		return fmt.Errorf("already exists")
	}
	c, err := m.connect(&s)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := InstallTailscale(c); err != nil {
		return err
	}
	const loginServer = "https://headscale.v1.dodo.cloud"
	if err := TailscaleUp(c, loginServer, s.Name, s.AuthKey); err != nil {
		return err
	}
	if err := InstallK3sJoinServer(c, m.serverAddr, m.serverToken); err != nil {
		return err
	}
	m.controllers = append(m.controllers, s)
	return nil
}

func (m *KubeManager) JoinWorker(s Server) error {
	m.l.Lock()
	defer m.l.Unlock()
	if m.kubeCfg == "" {
		return fmt.Errorf("not initialized")
	}
	if i := m.findServerByIP(s.IP); i != nil {
		return fmt.Errorf("already exists")
	}
	c, err := m.connect(&s)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := InstallTailscale(c); err != nil {
		return err
	}
	const loginServer = "https://headscale.v1.dodo.cloud"
	if err := TailscaleUp(c, loginServer, s.Name, s.AuthKey); err != nil {
		return err
	}
	if err := InstallK3sJoinAgent(c, m.serverAddr, m.serverToken); err != nil {
		return err
	}
	m.workers = append(m.workers, s)
	return nil
}

func (m *KubeManager) RemoveServer(name string) error {
	m.l.Lock()
	defer m.l.Unlock()
	client, err := kube.NewKubeClient(kube.KubeConfigOpts{
		KubeConfig: m.kubeCfg,
	})
	if err != nil {
		return err
	}
	helper := &drain.Helper{
		Ctx:                 context.Background(),
		Client:              client,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Out:                 os.Stdout,
		ErrOut:              os.Stdout,
		// We want to proceed even when pods are using emptyDir volumes
		DeleteEmptyDirData: true,
		Timeout:            10 * time.Minute,
	}
	node, err := client.CoreV1().Nodes().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if err := drain.RunCordonOrUncordon(helper, node, true); err != nil {
		return err
	}
	if err := drain.RunNodeDrain(helper, name); err != nil {
		return err
	}
	if err := client.CoreV1().Nodes().Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	for i, s := range m.controllers {
		if s.Name == name {
			c, err := m.connect(&s)
			if err != nil {
				return err
			}
			defer c.Close()
			if err := UninstallK3sServer(c); err != nil {
				return err
			}
			m.controllers = append(m.controllers[:i], m.controllers[i+1:]...)
			return nil
		}
	}
	for i, s := range m.workers {
		if s.Name == name {
			c, err := m.connect(&s)
			if err != nil {
				return err
			}
			defer c.Close()
			if err := UninstallK3sAgent(c); err != nil {
				return err
			}
			m.workers = append(m.workers[:i], m.workers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("not found")
}

// Expects manager state to be locked by caller.
func (m *KubeManager) findServerByIP(ip net.IP) *Server {
	for _, s := range m.controllers {
		if s.IP.Equal(ip) {
			return &s
		}
	}
	for _, s := range m.workers {
		if ip.Equal(s.IP) {
			return &s
		}
	}
	return nil
}

func (m *KubeManager) connect(s *Server) (*SSHClient, error) {
	cfg := &ssh.ClientConfig{
		User:    s.User,
		Auth:    []ssh.AuthMethod{},
		Timeout: 10 * time.Second,
	}
	if s.ClientKey != "" {
		clientKey, err := ssh.ParsePrivateKey([]byte(s.ClientKey))
		if err != nil {
			return nil, err
		}
		cfg.Auth = append(cfg.Auth, ssh.PublicKeys(clientKey))
	}
	if s.Password != "" {
		cfg.Auth = append(cfg.Auth, ssh.Password(s.Password))
	}
	if s.HostKey != "" {
		hostKey, err := ssh.ParsePublicKey([]byte(s.HostKey))
		if err != nil {
			return nil, err
		}
		cfg.HostKeyCallback = ssh.FixedHostKey(hostKey)
	} else {
		cfg.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", s.IP.String(), s.Port), cfg)
	if err != nil {
		return nil, err
	}
	ret := &SSHClient{client}
	s.Name, err = GetHostname(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
