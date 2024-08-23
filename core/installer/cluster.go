package installer

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/giolekva/pcloud/core/installer/soft"

	"sigs.k8s.io/yaml"
)

type ClusterNetworkConfigurator interface {
	AddCluster(name string, ingressIP net.IP) error
	RemoveCluster(name string, ingressIP net.IP) error
	AddProxy(src, dst string) error
	RemoveProxy(src, dst string) error
}

type NginxProxyConfigurator struct {
	PrivateSubdomain string
	DNSAPIAddr       string
	Repo             soft.RepoIO
	NginxConfigPath  string
}

type createARecordReq struct {
	Entry string `json:"entry"`
	IP    net.IP `json:"text"`
}

func (c *NginxProxyConfigurator) AddCluster(name string, ingressIP net.IP) error {
	req := createARecordReq{
		Entry: fmt.Sprintf("*.%s.cluster.%s", name, c.PrivateSubdomain),
		IP:    ingressIP,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("%s/create-a-record", c.DNSAPIAddr), "application/json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		io.Copy(&buf, resp.Body)
		return fmt.Errorf(buf.String())
	}
	return nil
}

func (c *NginxProxyConfigurator) RemoveCluster(name string, ingressIP net.IP) error {
	req := createARecordReq{
		Entry: fmt.Sprintf("*.%s.cluster.%s", name, c.PrivateSubdomain),
		IP:    ingressIP,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("%s/delete-a-record", c.DNSAPIAddr), "application/json", &buf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		io.Copy(&buf, resp.Body)
		return fmt.Errorf(buf.String())
	}
	return nil
}

func (c *NginxProxyConfigurator) AddProxy(src, dst string) error {
	_, err := c.Repo.Do(func(fs soft.RepoFS) (string, error) {
		cfg, err := func() (NginxProxyConfig, error) {
			r, err := fs.Reader(c.NginxConfigPath)
			if err != nil {
				return NginxProxyConfig{}, err
			}
			defer r.Close()
			return ParseNginxProxyConfig(r)
		}()
		if err != nil {
			return "", err
		}
		if v, ok := cfg.Proxies[src]; ok {
			return "", fmt.Errorf("mapping from %s already exists (%s)", src, v)
		}
		cfg.Proxies[src] = dst
		w, err := fs.Writer(c.NginxConfigPath)
		if err != nil {
			return "", err
		}
		defer w.Close()
		h := sha256.New()
		o := io.MultiWriter(w, h)
		if err := cfg.Render(o); err != nil {
			return "", err
		}
		hash := base64.StdEncoding.EncodeToString(h.Sum(nil))
		nginxPath := filepath.Join(filepath.Dir(c.NginxConfigPath), "ingress-nginx.yaml")
		nginx, err := func() (map[string]any, error) {
			r, err := fs.Reader(nginxPath)
			if err != nil {
				return nil, err
			}
			defer r.Close()
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				return nil, err
			}
			ret := map[string]any{}
			if err := yaml.Unmarshal(buf.Bytes(), &ret); err != nil {
				return nil, err
			}
			return ret, nil
		}()
		if err != nil {
			return "", err
		}
		cv := nginx["spec"].(map[string]any)["values"].(map[string]any)["controller"].(map[string]any)
		var annotations map[string]any
		if a, ok := cv["podAnnotations"]; ok {
			annotations = a.(map[string]any)
		} else {
			annotations = map[string]any{}
			cv["podAnnotations"] = annotations
		}
		annotations["dodo.cloud/hash"] = string(hash)
		buf, err := yaml.Marshal(nginx)
		if err != nil {
			return "", err
		}
		w, err = fs.Writer(nginxPath)
		if err != nil {
			return "", err
		}
		defer w.Close()
		if _, err := io.Copy(w, bytes.NewReader(buf)); err != nil {
			return "", err
		}
		return fmt.Sprintf("add proxy mapping: %s %s", src, dst), nil
	})
	return err
}

func (c *NginxProxyConfigurator) RemoveProxy(src, dst string) error {
	_, err := c.Repo.Do(func(fs soft.RepoFS) (string, error) {
		cfg, err := func() (NginxProxyConfig, error) {
			r, err := fs.Reader(c.NginxConfigPath)
			if err != nil {
				return NginxProxyConfig{}, err
			}
			defer r.Close()
			return ParseNginxProxyConfig(r)
		}()
		if err != nil {
			return "", err
		}
		if v, ok := cfg.Proxies[src]; !ok || v != dst {
			return "", fmt.Errorf("mapping does not exist: %s %s", src, dst)
		}
		delete(cfg.Proxies, src)
		w, err := fs.Writer(c.NginxConfigPath)
		if err != nil {
			return "", err
		}
		defer w.Close()
		h := sha256.New()
		o := io.MultiWriter(w, h)
		if err := cfg.Render(o); err != nil {
			return "", err
		}
		hash := base64.StdEncoding.EncodeToString(h.Sum(nil))
		nginxPath := filepath.Join(filepath.Dir(c.NginxConfigPath), "ingress-nginx.yaml")
		nginx, err := func() (map[string]any, error) {
			r, err := fs.Reader(nginxPath)
			if err != nil {
				return nil, err
			}
			defer r.Close()
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				return nil, err
			}
			ret := map[string]any{}
			if err := yaml.Unmarshal(buf.Bytes(), &ret); err != nil {
				return nil, err
			}
			return ret, nil
		}()
		if err != nil {
			return "", err
		}
		cv := nginx["spec"].(map[string]any)["values"].(map[string]any)["controller"].(map[string]any)
		var annotations map[string]any
		if a, ok := cv["podAnnotations"]; ok {
			annotations = a.(map[string]any)
		} else {
			annotations = map[string]any{}
			cv["podAnnotations"] = annotations
		}
		annotations["dodo.cloud/hash"] = string(hash)
		buf, err := yaml.Marshal(nginx)
		if err != nil {
			return "", err
		}
		w, err = fs.Writer(nginxPath)
		if err != nil {
			return "", err
		}
		defer w.Close()
		if _, err := io.Copy(w, bytes.NewReader(buf)); err != nil {
			return "", err
		}
		return fmt.Sprintf("remove proxy mapping: %s %s", src, dst), nil
	})
	return err
}

type NginxProxyConfig struct {
	Port      int
	Resolvers []net.IP
	Proxies   map[string]string
	PreConf   []string
}

func ParseNginxProxyConfig(r io.Reader) (NginxProxyConfig, error) {
	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		return NginxProxyConfig{}, err
	}
	ret := NginxProxyConfig{
		Port:      -1,
		Resolvers: nil,
		Proxies:   make(map[string]string),
	}
	lines := strings.Split(buf.String(), "\n")
	insideConf := true
	insideMap := false
	for _, l := range lines {
		items := strings.Fields(strings.TrimSuffix(l, ";"))
		if len(items) == 0 {
			continue
		}
		if strings.Contains(l, "nginx.conf") {
			ret.PreConf = append(ret.PreConf, l)
			insideConf = false
		} else if insideConf {
			ret.PreConf = append(ret.PreConf, l)
		} else if strings.Contains(l, "listen") {
			if len(items) < 2 {
				return NginxProxyConfig{}, fmt.Errorf("invalid listen: %s\n", l)
			}
			port, err := strconv.Atoi(items[1])
			if err != nil {
				return NginxProxyConfig{}, err
			}
			ret.Port = port
		} else if strings.Contains(l, "resolver") {
			if len(items) < 2 {
				return NginxProxyConfig{}, fmt.Errorf("invalid resolver: %s", l)
			}
			ip := net.ParseIP(items[1])
			if ip == nil {
				return NginxProxyConfig{}, fmt.Errorf("invalid resolver ip: %s", l)
			}
			ret.Resolvers = append(ret.Resolvers, ip)
		} else if insideMap {
			if items[0] == "}" {
				insideMap = false
				continue
			}
			if len(items) < 2 {
				return NginxProxyConfig{}, fmt.Errorf("invalid map: %s", l)
			}
			ret.Proxies[items[0]] = items[1]
		} else if items[0] == "map" {
			insideMap = true
		}
	}
	return ret, nil
}

func (c NginxProxyConfig) Render(w io.Writer) error {
	for _, l := range c.PreConf {
		fmt.Fprintln(w, l)
	}
	tmpl, err := template.New("nginx.conf").Parse(nginxConfigTmpl)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, c)
}

const nginxConfigTmpl = `    worker_processes  1;
    worker_rlimit_nofile 8192;
    events {
        worker_connections  1024;
    }
    http {
        map $http_host $backend {
            {{- range $from, $to := .Proxies }}
            {{ $from }} {{ $to }};
            {{- end }}
        }
        server {
            listen {{ .Port }};
            location / {
                {{- range .Resolvers }}
                resolver {{ . }};
                {{- end }}
                proxy_pass http://$backend;
            }
        }
    }`
