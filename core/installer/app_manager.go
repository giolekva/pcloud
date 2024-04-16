package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

const appDir = "/apps"
const configFileName = "config.yaml"
const kustomizationFileName = "kustomization.yaml"

type AppManager struct {
	repoIO    RepoIO
	nsCreator NamespaceCreator
}

func NewAppManager(repoIO RepoIO, nsCreator NamespaceCreator) (*AppManager, error) {
	return &AppManager{
		repoIO,
		nsCreator,
	}, nil
}

func (m *AppManager) Config() (Config, error) {
	var cfg Config
	if err := ReadYaml(m.repoIO, configFileName, &cfg); err != nil {
		return Config{}, err
	} else {
		return cfg, nil
	}
}

func (m *AppManager) appConfig(path string) (AppConfig, error) {
	var cfg AppConfig
	if err := ReadYaml(m.repoIO, path, &cfg); err != nil {
		return AppConfig{}, err
	} else {
		return cfg, nil
	}
}

func (m *AppManager) FindAllInstances(name string) ([]AppConfig, error) {
	kust, err := ReadKustomization(m.repoIO, filepath.Join(appDir, "kustomization.yaml"))
	if err != nil {
		return nil, err
	}
	ret := make([]AppConfig, 0)
	for _, app := range kust.Resources {
		cfg, err := m.appConfig(filepath.Join(appDir, app, "config.yaml"))
		if err != nil {
			return nil, err
		}
		cfg.Id = app
		if cfg.AppId == name {
			ret = append(ret, cfg)
		}
	}
	return ret, nil
}

func (m *AppManager) FindInstance(id string) (AppConfig, error) {
	kust, err := ReadKustomization(m.repoIO, filepath.Join(appDir, "kustomization.yaml"))
	if err != nil {
		return AppConfig{}, err
	}
	for _, app := range kust.Resources {
		if app == id {
			cfg, err := m.appConfig(filepath.Join(appDir, app, "config.yaml"))
			if err != nil {
				return AppConfig{}, err
			}
			cfg.Id = id
			return cfg, nil
		}
	}
	return AppConfig{}, nil
}

func (m *AppManager) AppConfig(name string) (AppConfig, error) {
	configF, err := m.repoIO.Reader(filepath.Join(appDir, name, configFileName))
	if err != nil {
		return AppConfig{}, err
	}
	defer configF.Close()
	var cfg AppConfig
	contents, err := ioutil.ReadAll(configF)
	if err != nil {
		return AppConfig{}, err
	}
	err = yaml.UnmarshalStrict(contents, &cfg)
	return cfg, err
}

type allocatePortReq struct {
	Protocol      string `json:"protocol"`
	SourcePort    int    `json:"sourcePort"`
	TargetService string `json:"targetService"`
	TargetPort    int    `json:"targetPort"`
}

func openPorts(ports []PortForward) error {
	for _, p := range ports {
		var buf bytes.Buffer
		req := allocatePortReq{
			Protocol:      p.Protocol,
			SourcePort:    p.SourcePort,
			TargetService: p.TargetService,
			TargetPort:    p.TargetPort,
		}
		if err := json.NewEncoder(&buf).Encode(req); err != nil {
			return err
		}
		resp, err := http.Post(p.Allocator, "application/json", &buf)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Could not allocate port %d, status code: %d", p.SourcePort, resp.StatusCode)
		}
	}
	return nil
}

func createKustomizationChain(r RepoFS, path string) error {
	for p := filepath.Clean(path); p != "/"; {
		parent, child := filepath.Split(p)
		kustPath := filepath.Join(parent, "kustomization.yaml")
		kust, err := ReadKustomization(r, kustPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				k := NewKustomization()
				kust = &k
			} else {
				return err
			}
		}
		kust.AddResources(child)
		if err := WriteYaml(r, kustPath, kust); err != nil {
			return err
		}
		p = filepath.Clean(parent)
	}
	return nil
}

func InstallApp(repo RepoIO, nsc NamespaceCreator, app App, appDir string, namespace string, initValues map[string]any, derived Derived) error {
	if err := nsc.Create(namespace); err != nil {
		return err
	}
	derived.Release = Release{
		Namespace: namespace,
		RepoAddr:  repo.FullAddress(),
		AppDir:    appDir,
	}
	rendered, err := app.Render(derived)
	if err != nil {
		return err
	}
	if err := openPorts(rendered.Ports); err != nil {
		return err
	}
	return repo.Atomic(func(r RepoFS) (string, error) {
		if err := createKustomizationChain(r, appDir); err != nil {
			return "", err
		}
		{
			if err := r.RemoveDir(appDir); err != nil {
				return "", err
			}
			if err := r.CreateDir(appDir); err != nil {
				return "", err
			}
			cfg := AppConfig{
				AppId:   app.Name(),
				Config:  initValues,
				Derived: derived,
			}
			if err := WriteYaml(r, path.Join(appDir, configFileName), cfg); err != nil {
				return "", err
			}
		}
		{
			appKust := NewKustomization()
			for name, contents := range rendered.Resources {
				appKust.AddResources(name)
				out, err := r.Writer(path.Join(appDir, name))
				if err != nil {
					return "", err
				}
				defer out.Close()
				if _, err := out.Write(contents); err != nil {
					return "", err
				}
			}
			if err := WriteYaml(r, path.Join(appDir, "kustomization.yaml"), appKust); err != nil {
				return "", err
			}
		}
		return fmt.Sprintf("install: %s", app.Name()), nil
	})
}

func (m *AppManager) Install(app App, appDir string, namespace string, values map[string]any) error {
	appDir = filepath.Clean(appDir)
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	globalConfig, err := m.Config()
	if err != nil {
		return err
	}
	derivedValues, err := deriveValues(values, app.Schema(), CreateNetworks(globalConfig))
	if err != nil {
		return err
	}
	derived := Derived{
		Global: globalConfig.Values,
		Values: derivedValues,
	}
	return InstallApp(m.repoIO, m.nsCreator, app, appDir, namespace, values, derived)
}

func (m *AppManager) Update(app App, instanceId string, config map[string]any) error {
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	globalConfig, err := m.Config()
	if err != nil {
		return err
	}
	instanceDir := filepath.Join(appDir, instanceId)
	instanceConfigPath := filepath.Join(instanceDir, configFileName)
	appConfig, err := m.appConfig(instanceConfigPath)
	if err != nil {
		return err
	}
	derivedValues, err := deriveValues(config, app.Schema(), CreateNetworks(globalConfig))
	if err != nil {
		return err
	}
	derived := Derived{
		Global:  globalConfig.Values,
		Release: appConfig.Derived.Release,
		Values:  derivedValues,
	}
	return InstallApp(m.repoIO, m.nsCreator, app, instanceDir, appConfig.Derived.Release.Namespace, config, derived)
}

func (m *AppManager) Remove(instanceId string) error {
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	return m.repoIO.Atomic(func(r RepoFS) (string, error) {
		r.RemoveDir(filepath.Join(appDir, instanceId))
		kustPath := filepath.Join(appDir, "kustomization.yaml")
		kust, err := ReadKustomization(r, kustPath)
		if err != nil {
			return "", err
		}
		kust.RemoveResources(instanceId)
		WriteYaml(r, kustPath, kust)
		return fmt.Sprintf("uninstall: %s", instanceId), nil
	})
}

// TODO(gio): deduplicate with cue definition in app.go, this one should be removed.
func CreateNetworks(global Config) []Network {
	return []Network{
		{
			Name:              "Public",
			IngressClass:      fmt.Sprintf("%s-ingress-public", global.Values.PCloudEnvName),
			CertificateIssuer: fmt.Sprintf("%s-public", global.Values.Id),
			Domain:            global.Values.Domain,
			AllocatePortAddr:  fmt.Sprintf("http://port-allocator.%s-ingress-public/api/allocate", global.Values.PCloudEnvName),
		},
		{
			Name:             "Private",
			IngressClass:     fmt.Sprintf("%s-ingress-private", global.Values.Id),
			Domain:           global.Values.PrivateDomain,
			AllocatePortAddr: fmt.Sprintf("http://port-allocator.%s-ingress-private/api/allocate", global.Values.Id),
		},
	}
}
