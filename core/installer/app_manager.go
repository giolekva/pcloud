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

const appDirRoot = "/apps"
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

func (m *AppManager) Config() (AppEnvConfig, error) {
	var cfg AppEnvConfig
	if err := ReadYaml(m.repoIO, configFileName, &cfg); err != nil {
		return AppEnvConfig{}, err
	} else {
		return cfg, nil
	}
}

func (m *AppManager) appConfig(path string) (AppInstanceConfig, error) {
	var cfg AppInstanceConfig
	if err := ReadYaml(m.repoIO, path, &cfg); err != nil {
		return AppInstanceConfig{}, err
	} else {
		return cfg, nil
	}
}

func (m *AppManager) FindAllInstances(name string) ([]AppInstanceConfig, error) {
	kust, err := ReadKustomization(m.repoIO, filepath.Join(appDirRoot, "kustomization.yaml"))
	if err != nil {
		return nil, err
	}
	ret := make([]AppInstanceConfig, 0)
	for _, app := range kust.Resources {
		cfg, err := m.appConfig(filepath.Join(appDirRoot, app, "config.yaml"))
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

func (m *AppManager) FindInstance(id string) (AppInstanceConfig, error) {
	kust, err := ReadKustomization(m.repoIO, filepath.Join(appDirRoot, "kustomization.yaml"))
	if err != nil {
		return AppInstanceConfig{}, err
	}
	for _, app := range kust.Resources {
		if app == id {
			cfg, err := m.appConfig(filepath.Join(appDirRoot, app, "config.yaml"))
			if err != nil {
				return AppInstanceConfig{}, err
			}
			cfg.Id = id
			return cfg, nil
		}
	}
	return AppInstanceConfig{}, nil
}

func (m *AppManager) AppConfig(name string) (AppInstanceConfig, error) {
	configF, err := m.repoIO.Reader(filepath.Join(appDirRoot, name, configFileName))
	if err != nil {
		return AppInstanceConfig{}, err
	}
	defer configF.Close()
	var cfg AppInstanceConfig
	contents, err := ioutil.ReadAll(configF)
	if err != nil {
		return AppInstanceConfig{}, err
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

// TODO(gio): rename to CommitApp
func InstallApp(repo RepoIO, appDir string, rendered Rendered) error {
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
			if err := WriteYaml(r, path.Join(appDir, configFileName), rendered.Config); err != nil {
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
		return fmt.Sprintf("install: %s", rendered.Name), nil
	})
}

// TODO(gio): commit instanceId -> appDir mapping as well
func (m *AppManager) Install(app EnvApp, instanceId string, appDir string, namespace string, values map[string]any) error {
	appDir = filepath.Clean(appDir)
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	if err := m.nsCreator.Create(namespace); err != nil {
		return err
	}
	env, err := m.Config()
	if err != nil {
		return err
	}
	release := Release{
		AppInstanceId: instanceId,
		Namespace:     namespace,
		RepoAddr:      m.repoIO.FullAddress(),
		AppDir:        appDir,
	}
	rendered, err := app.Render(release, env, values)
	if err != nil {
		return err
	}
	return InstallApp(m.repoIO, appDir, rendered)
}

func (m *AppManager) Update(app EnvApp, instanceId string, values map[string]any) error {
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	env, err := m.Config()
	if err != nil {
		return err
	}
	instanceDir := filepath.Join(appDirRoot, instanceId)
	instanceConfigPath := filepath.Join(instanceDir, configFileName)
	config, err := m.appConfig(instanceConfigPath)
	if err != nil {
		return err
	}
	release := Release{
		AppInstanceId: instanceId,
		Namespace:     config.Release.Namespace,
		RepoAddr:      m.repoIO.FullAddress(),
		AppDir:        appDirRoot,
	}
	rendered, err := app.Render(release, env, values)
	if err != nil {
		return err
	}
	return InstallApp(m.repoIO, instanceDir, rendered)
}

func (m *AppManager) Remove(instanceId string) error {
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	return m.repoIO.Atomic(func(r RepoFS) (string, error) {
		r.RemoveDir(filepath.Join(appDirRoot, instanceId))
		kustPath := filepath.Join(appDirRoot, "kustomization.yaml")
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
func CreateNetworks(env AppEnvConfig) []Network {
	return []Network{
		{
			Name:              "Public",
			IngressClass:      fmt.Sprintf("%s-ingress-public", env.InfraName),
			CertificateIssuer: fmt.Sprintf("%s-public", env.Id),
			Domain:            env.Domain,
			AllocatePortAddr:  fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/allocate", env.InfraName),
		},
		{
			Name:             "Private",
			IngressClass:     fmt.Sprintf("%s-ingress-private", env.Id),
			Domain:           env.PrivateDomain,
			AllocatePortAddr: fmt.Sprintf("http://port-allocator.%s-ingress-private.svc.cluster.local/api/allocate", env.Id),
		},
	}
}

// InfraAppmanager

type InfraAppManager struct {
	repoIO    RepoIO
	nsCreator NamespaceCreator
}

func NewInfraAppManager(repoIO RepoIO, nsCreator NamespaceCreator) (*InfraAppManager, error) {
	return &InfraAppManager{
		repoIO,
		nsCreator,
	}, nil
}

func (m *InfraAppManager) Config() (InfraConfig, error) {
	var cfg InfraConfig
	if err := ReadYaml(m.repoIO, configFileName, &cfg); err != nil {
		return InfraConfig{}, err
	} else {
		return cfg, nil
	}
}

func (m *InfraAppManager) Install(app InfraApp, appDir string, namespace string, values map[string]any) error {
	appDir = filepath.Clean(appDir)
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	if err := m.nsCreator.Create(namespace); err != nil {
		return err
	}
	infra, err := m.Config()
	if err != nil {
		return err
	}
	release := Release{
		Namespace: namespace,
		RepoAddr:  m.repoIO.FullAddress(),
		AppDir:    appDir,
	}
	rendered, err := app.Render(release, infra, values)
	if err != nil {
		return err
	}
	return InstallApp(m.repoIO, appDir, rendered)
}
