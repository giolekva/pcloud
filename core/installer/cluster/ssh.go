package cluster

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"strings"
)

type SSHClient struct {
	client *ssh.Client
}

func (c *SSHClient) Close() error {
	return c.client.Close()
}

func (c *SSHClient) Exec(cmd string) (string, error) {
	ses, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer ses.Close()
	var out bytes.Buffer
	ses.Stdout = &out
	ses.Stderr = os.Stdout
	err = ses.Run(cmd)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func GetHostname(c *SSHClient) (string, error) {
	name, err := c.Exec("hostname")
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(name), nil
}

func InstallTailscale(c *SSHClient) error {
	return nil
	fmt.Println("Installing Tailscale")
	if _, err := c.Exec("which tailscale"); err == nil {
		return nil
	}
	_, err := c.Exec(tailscaleInstallCmd)
	return err
}

func TailscaleUp(c *SSHClient, loginServer, hostname, authKey string) error {
	return nil
	fmt.Println("Starting up Tailscale")
	if _, err := c.Exec("sudo tailscale down"); err != nil {
		return err
	}
	cmd := fmt.Sprintf(tailscaleUpCmd, loginServer, authKey, hostname)
	fmt.Println(cmd)
	_, err := c.Exec(cmd)
	return err
}

func InstallK3s(c *SSHClient) error {
	fmt.Println("Starting k3s")
	if _, err := c.Exec("which k3s"); err == nil {
		return nil
	}
	_, err := c.Exec("curl -sfL https://get.k3s.io | sh -s - --cluster-init --disable traefik --disable local-storage --disable servicelb --kube-proxy-arg proxy-mode=ipvs --kube-proxy-arg ipvs-strict-arp --flannel-backend vxlan --cluster-cidr=10.45.0.0/16 --service-cidr=10.46.0.0/16 # --flannel-iface=tailscale0")
	return err
}

func InstallK3sJoinServer(c *SSHClient, serverAddr, token string) error {
	fmt.Println("Starting k3s")
	if _, err := c.Exec("which k3s"); err == nil {
		return nil
	}
	_, err := c.Exec(fmt.Sprintf("curl -sfL https://get.k3s.io | sh -s - server --server=https://%s --token=%s --disable traefik --disable local-storage --disable servicelb --kube-proxy-arg proxy-mode=ipvs --kube-proxy-arg ipvs-strict-arp --flannel-backend vxlan --cluster-cidr=10.45.0.0/16 --service-cidr=10.46.0.0/16 # --flannel-iface=tailscale0", serverAddr, token))
	return err
}

func InstallK3sJoinAgent(c *SSHClient, serverAddr, token string) error {
	fmt.Println("Starting k3s")
	if _, err := c.Exec("which k3s"); err == nil {
		return nil
	}
	_, err := c.Exec(fmt.Sprintf("curl -sfL https://get.k3s.io | sh -s - agent --server=https://%s --token=%s", serverAddr, token))
	return err
}

func UninstallK3sServer(c *SSHClient) error {
	fmt.Println("Uninstalling k3s")
	if _, err := c.Exec("which k3s-uninstall.sh"); err != nil {
		return nil
	}
	_, err := c.Exec("k3s-uninstall.sh")
	return err
}

func UninstallK3sAgent(c *SSHClient) error {
	fmt.Println("Uninstalling k3s")
	if _, err := c.Exec("which k3s-agent-uninstall.sh"); err != nil {
		return nil
	}
	_, err := c.Exec("k3s-agent-uninstall.sh")
	return err
}

func GetTailscaleIP(c *SSHClient) (string, error) {
	fmt.Println("Getting Tailscale IP")
	if _, err := c.Exec("sudo apt-get install net-tools -y"); err != nil {
		return "", err
	}
	ip, err := c.Exec("sudo ifconfig | grep 10.42")
	if err != nil {
		return "", err
	}
	return strings.Fields(ip)[1], nil
	// ip, err := c.Exec("sudo tailscale ip")
	// return strings.TrimSpace(ip), err
}

func GetKubeconfig(c *SSHClient) (string, error) {
	// return "", nil
	fmt.Println("Getting Kubeconfig")
	out, err := c.Exec("sudo cat /etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return "", err
	}
	ip, err := GetTailscaleIP(c)
	if err != nil {
		return "", err
	}
	return strings.Replace(out, "server: https://127.0.0.1:6443", fmt.Sprintf("server: https://%s:6443", ip), 1), nil
}

func GetServerToken(c *SSHClient) (string, error) {
	fmt.Println("Getting server token")
	out, err := c.Exec("sudo cat /var/lib/rancher/k3s/server/node-token")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), err
}
