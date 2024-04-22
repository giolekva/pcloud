package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"

	"github.com/giolekva/pcloud/core/installer/io"
	"github.com/giolekva/pcloud/core/installer/soft"
)

const configFileName = "config.yaml"
const kustomizationFileName = "kustomization.yaml"

type AppManager struct {
	repoIO     soft.RepoIO
	nsCreator  NamespaceCreator
	appDirRoot string
}

func NewAppManager(repoIO soft.RepoIO, nsCreator NamespaceCreator, appDirRoot string) (*AppManager, error) {
	return &AppManager{
		repoIO,
		nsCreator,
		appDirRoot,
	}, nil
}

func (m *AppManager) Config() (EnvConfig, error) {
	var cfg EnvConfig
	if err := soft.ReadYaml(m.repoIO, configFileName, &cfg); err != nil {
		return EnvConfig{}, err
	} else {
		return cfg, nil
	}
}

func (m *AppManager) appConfig(path string) (AppInstanceConfig, error) {
	var cfg AppInstanceConfig
	if err := soft.ReadJson(m.repoIO, path, &cfg); err != nil {
		return AppInstanceConfig{}, err
	} else {
		return cfg, nil
	}
}

func (m *AppManager) FindAllInstances() ([]AppInstanceConfig, error) {
	kust, err := soft.ReadKustomization(m.repoIO, filepath.Join(m.appDirRoot, "kustomization.yaml"))
	if err != nil {
		return nil, err
	}
	ret := make([]AppInstanceConfig, 0)
	for _, app := range kust.Resources {
		cfg, err := m.appConfig(filepath.Join(m.appDirRoot, app, "config.json"))
		if err != nil {
			return nil, err
		}
		cfg.Id = app
		ret = append(ret, cfg)
	}
	return ret, nil
}

func (m *AppManager) FindAllAppInstances(name string) ([]AppInstanceConfig, error) {
	kust, err := soft.ReadKustomization(m.repoIO, filepath.Join(m.appDirRoot, "kustomization.yaml"))
	if err != nil {
		return nil, err
	}
	ret := make([]AppInstanceConfig, 0)
	for _, app := range kust.Resources {
		cfg, err := m.appConfig(filepath.Join(m.appDirRoot, app, "config.json"))
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
	kust, err := soft.ReadKustomization(m.repoIO, filepath.Join(m.appDirRoot, "kustomization.yaml"))
	if err != nil {
		return AppInstanceConfig{}, err
	}
	for _, app := range kust.Resources {
		if app == id {
			cfg, err := m.appConfig(filepath.Join(m.appDirRoot, app, "config.json"))
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
	var cfg AppInstanceConfig
	if err := soft.ReadJson(m.repoIO, filepath.Join(m.appDirRoot, name, "config.json"), &cfg); err != nil {
		return AppInstanceConfig{}, err
	}
	return cfg, nil
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

func createKustomizationChain(r soft.RepoFS, path string) error {
	for p := filepath.Clean(path); p != "/"; {
		parent, child := filepath.Split(p)
		kustPath := filepath.Join(parent, "kustomization.yaml")
		kust, err := soft.ReadKustomization(r, kustPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				k := io.NewKustomization()
				kust = &k
			} else {
				return err
			}
		}
		kust.AddResources(child)
		if err := soft.WriteYaml(r, kustPath, kust); err != nil {
			return err
		}
		p = filepath.Clean(parent)
	}
	return nil
}

// TODO(gio): rename to CommitApp
func InstallApp(
	repo soft.RepoIO,
	appDir string,
	name string,
	config any,
	ports []PortForward,
	resources CueAppData,
	data CueAppData,
	opts ...soft.DoOption) error {
	// if err := openPorts(rendered.Ports); err != nil {
	// 	return err
	// }
	return repo.Do(func(r soft.RepoFS) (string, error) {
		if err := r.RemoveDir(appDir); err != nil {
			return "", err
		}
		resourcesDir := path.Join(appDir, "resources")
		if err := r.CreateDir(resourcesDir); err != nil {
			return "", err
		}
		{
			if err := soft.WriteYaml(r, path.Join(appDir, configFileName), config); err != nil {
				return "", err
			}
			if err := soft.WriteJson(r, path.Join(appDir, "config.json"), config); err != nil {
				return "", err
			}
			for name, contents := range data {
				if name == "config.json" || name == "kustomization.yaml" || name == "resources" {
					return "", fmt.Errorf("%s is forbidden", name)
				}
				w, err := r.Writer(path.Join(appDir, name))
				if err != nil {
					return "", err
				}
				defer w.Close()
				if _, err := w.Write(contents); err != nil {
					return "", err
				}
			}
		}
		{
			if err := createKustomizationChain(r, resourcesDir); err != nil {
				return "", err
			}
			appKust := io.NewKustomization()
			for name, contents := range resources {
				appKust.AddResources(name)
				w, err := r.Writer(path.Join(resourcesDir, name))
				if err != nil {
					return "", err
				}
				defer w.Close()
				if _, err := w.Write(contents); err != nil {
					return "", err
				}
			}
			if err := soft.WriteYaml(r, path.Join(resourcesDir, "kustomization.yaml"), appKust); err != nil {
				return "", err
			}
		}
		return fmt.Sprintf("install: %s", name), nil
	}, opts...)
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
	return InstallApp(m.repoIO, appDir, rendered.Name, rendered.Config, rendered.Ports, rendered.Resources, rendered.Data)
}

func (m *AppManager) Update(app EnvApp, instanceId string, values map[string]any, opts ...soft.DoOption) error {
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	env, err := m.Config()
	if err != nil {
		return err
	}
	instanceDir := filepath.Join(m.appDirRoot, instanceId)
	instanceConfigPath := filepath.Join(instanceDir, "config.json")
	config, err := m.appConfig(instanceConfigPath)
	if err != nil {
		return err
	}
	release := Release{
		AppInstanceId: instanceId,
		Namespace:     config.Release.Namespace,
		RepoAddr:      m.repoIO.FullAddress(),
		AppDir:        instanceDir,
	}
	rendered, err := app.Render(release, env, values)
	if err != nil {
		return err
	}
	return InstallApp(m.repoIO, instanceDir, rendered.Name, rendered.Config, rendered.Ports, rendered.Resources, rendered.Data, opts...)
}

func (m *AppManager) Remove(instanceId string) error {
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	return m.repoIO.Do(func(r soft.RepoFS) (string, error) {
		r.RemoveDir(filepath.Join(m.appDirRoot, instanceId))
		kustPath := filepath.Join(m.appDirRoot, "kustomization.yaml")
		kust, err := soft.ReadKustomization(r, kustPath)
		if err != nil {
			return "", err
		}
		kust.RemoveResources(instanceId)
		soft.WriteYaml(r, kustPath, kust)
		return fmt.Sprintf("uninstall: %s", instanceId), nil
	})
}

// TODO(gio): deduplicate with cue definition in app.go, this one should be removed.
func CreateNetworks(env EnvConfig) []Network {
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
	repoIO    soft.RepoIO
	nsCreator NamespaceCreator
}

func NewInfraAppManager(repoIO soft.RepoIO, nsCreator NamespaceCreator) (*InfraAppManager, error) {
	return &InfraAppManager{
		repoIO,
		nsCreator,
	}, nil
}

func (m *InfraAppManager) Config() (InfraConfig, error) {
	var cfg InfraConfig
	if err := soft.ReadYaml(m.repoIO, configFileName, &cfg); err != nil {
		return InfraConfig{}, err
	} else {
		return cfg, nil
	}
}

func (m *InfraAppManager) appConfig(path string) (InfraAppInstanceConfig, error) {
	var cfg InfraAppInstanceConfig
	if err := soft.ReadJson(m.repoIO, path, &cfg); err != nil {
		return InfraAppInstanceConfig{}, err
	} else {
		return cfg, nil
	}
}

func (m *InfraAppManager) FindInstance(id string) (InfraAppInstanceConfig, error) {
	kust, err := soft.ReadKustomization(m.repoIO, filepath.Join("/infrastructure", "kustomization.yaml"))
	if err != nil {
		return InfraAppInstanceConfig{}, err
	}
	for _, app := range kust.Resources {
		if app == id {
			cfg, err := m.appConfig(filepath.Join("/infrastructure", app, "config.json"))
			if err != nil {
				return InfraAppInstanceConfig{}, err
			}
			cfg.Id = id
			return cfg, nil
		}
	}
	return InfraAppInstanceConfig{}, nil
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
	return InstallApp(m.repoIO, appDir, rendered.Name, rendered.Config, rendered.Ports, rendered.Resources, rendered.Data)
}

func (m *InfraAppManager) Update(app InfraApp, instanceId string, values map[string]any, opts ...soft.DoOption) error {
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	env, err := m.Config()
	if err != nil {
		return err
	}
	instanceDir := filepath.Join("/infrastructure", instanceId)
	instanceConfigPath := filepath.Join(instanceDir, "config.json")
	config, err := m.appConfig(instanceConfigPath)
	if err != nil {
		return err
	}
	release := Release{
		AppInstanceId: instanceId,
		Namespace:     config.Release.Namespace,
		RepoAddr:      m.repoIO.FullAddress(),
		AppDir:        instanceDir,
	}
	rendered, err := app.Render(release, env, values)
	if err != nil {
		return err
	}
	return InstallApp(m.repoIO, instanceDir, rendered.Name, rendered.Config, rendered.Ports, rendered.Resources, rendered.Data, opts...)
}
