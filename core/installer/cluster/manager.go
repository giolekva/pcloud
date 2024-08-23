package cluster

import (
	"net"
)

const (
	whichTailscale      = "which tailscale"
	tailscaleInstallCmd = "curl -fsSL https://tailscale.com/install.sh | sh"
	tailscaleUpCmd      = "sudo tailscale up --login-server=%s --auth-key=%s --hostname=%s --reset"
)

type Server struct {
	Name      string `json:"name"`
	IP        net.IP `json:"ip"`
	Port      int    `json:"port"`
	HostKey   string `json:"hostKey"`
	User      string `json:"user"`
	Password  string `json:"password"`
	ClientKey string `json:"clientKey"`
	AuthKey   string `json:"authKey"`
}

type State struct {
	Name             string   `json:"name"`
	IngressClassName string   `json:"ingressClassName"`
	IngressIP        net.IP   `json:"ingressIP"`
	ServerAddr       string   `json:"serverAddr"`
	ServerToken      string   `json:"serverToken"`
	Kubeconfig       string   `json:"kubeconfig"`
	Controllers      []Server `json:"controllers"`
	Workers          []Server `json:"workers"`
}

type ClusterSetupFunc func(name, kubeconfig, ingressClassName string) (net.IP, error)

type Manager interface {
	Init(s Server, setupFn ClusterSetupFunc) (net.IP, error)
	JoinController(s Server) error
	JoinWorker(s Server) error
	RemoveServer(name string) error
	State() State
}
