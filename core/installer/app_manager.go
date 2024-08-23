package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/giolekva/pcloud/core/installer/cluster"
	gio "github.com/giolekva/pcloud/core/installer/io"
	"github.com/giolekva/pcloud/core/installer/kube"
	"github.com/giolekva/pcloud/core/installer/soft"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"sigs.k8s.io/yaml"
)

const (
	configFileName        = "config.yaml"
	kustomizationFileName = "kustomization.yaml"
	gitIgnoreFileName     = ".gitignore"
	includeEverything     = "!*"
)

var ErrorNotFound = errors.New("not found")

type AppManager struct {
	l            sync.Locker
	repo         soft.RepoIO
	nsc          NamespaceCreator
	jc           JobCreator
	hf           HelmFetcher
	vpnAPIClient VPNAPIClient
	cnc          ClusterNetworkConfigurator
	appDirRoot   string
}

func NewAppManager(
	repo soft.RepoIO,
	nsc NamespaceCreator,
	jc JobCreator,
	hf HelmFetcher,
	vpnKeyGen VPNAPIClient,
	cnc ClusterNetworkConfigurator,
	appDirRoot string,
) (*AppManager, error) {
	return &AppManager{
		&sync.Mutex{},
		repo,
		nsc,
		jc,
		hf,
		vpnKeyGen,
		cnc,
		appDirRoot,
	}, nil
}

func (m *AppManager) Config() (EnvConfig, error) {
	var cfg EnvConfig
	if err := soft.ReadYaml(m.repo, configFileName, &cfg); err != nil {
		return EnvConfig{}, err
	} else {
		return cfg, nil
	}
}

func (m *AppManager) appConfig(path string) (AppInstanceConfig, error) {
	var cfg AppInstanceConfig
	if err := soft.ReadJson(m.repo, path, &cfg); err != nil {
		return AppInstanceConfig{}, err
	} else {
		return cfg, nil
	}
}

func (m *AppManager) GetAllInstances() ([]AppInstanceConfig, error) {
	m.repo.Pull()
	kust, err := soft.ReadKustomization(m.repo, filepath.Join(m.appDirRoot, "kustomization.yaml"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
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

func (m *AppManager) GetAllAppInstances(name string) ([]AppInstanceConfig, error) {
	kust, err := soft.ReadKustomization(m.repo, filepath.Join(m.appDirRoot, "kustomization.yaml"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		} else {
			return nil, err
		}
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

func (m *AppManager) GetInstance(id string) (*AppInstanceConfig, error) {
	appDir := filepath.Clean(filepath.Join(m.appDirRoot, id))
	cfgPath := filepath.Join(appDir, "config.json")
	cfg, err := m.appConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	cfg.Id = id
	return &cfg, err
}

func GetCueAppData(fs soft.RepoFS, dir string) (CueAppData, error) {
	files, err := fs.ListDir(dir)
	if err != nil {
		return nil, err
	}
	cfg := CueAppData{}
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".cue") {
			contents, err := soft.ReadFile(fs, filepath.Join(dir, f.Name()))
			if err != nil {
				return nil, err
			}
			cfg[f.Name()] = contents
		}
	}
	return cfg, nil
}

func (m *AppManager) GetInstanceApp(id string) (EnvApp, error) {
	cfg, err := GetCueAppData(m.repo, filepath.Join(m.appDirRoot, id))
	if err != nil {
		return nil, err
	}
	return NewCueEnvApp(cfg)
}

type allocatePortReq struct {
	Protocol      string `json:"protocol"`
	SourcePort    int    `json:"sourcePort"`
	TargetService string `json:"targetService"`
	TargetPort    int    `json:"targetPort"`
	Secret        string `json:"secret,omitempty"`
}

type removePortReq struct {
	Protocol      string `json:"protocol"`
	SourcePort    int    `json:"sourcePort"`
	TargetService string `json:"targetService"`
	TargetPort    int    `json:"targetPort"`
}

type reservePortResp struct {
	Port   int    `json:"port"`
	Secret string `json:"secret"`
}

func reservePorts(ports map[string]string) (map[string]reservePortResp, error) {
	ret := map[string]reservePortResp{}
	for p, reserveAddr := range ports {
		resp, err := http.Post(reserveAddr, "application/json", nil) // TODO(gio): address
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			var e bytes.Buffer
			io.Copy(&e, resp.Body)
			return nil, fmt.Errorf("Could not reserve port: %s", e.String())
		}
		var r reservePortResp
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return nil, err
		}
		ret[p] = r
	}
	return ret, nil
}

func openPorts(ports []PortForward, reservations map[string]reservePortResp, allocators map[string]string) error {
	for _, p := range ports {
		var buf bytes.Buffer
		req := allocatePortReq{
			Protocol:      p.Protocol,
			SourcePort:    p.SourcePort,
			TargetService: p.TargetService,
			TargetPort:    p.TargetPort,
		}
		allocator := ""
		for n, r := range reservations {
			if p.SourcePort == r.Port {
				allocator = allocators[n]
				req.Secret = r.Secret
				break
			}
		}
		if allocator == "" {
			return fmt.Errorf("Could not find allocator for: %d", p.SourcePort)
		}
		if err := json.NewEncoder(&buf).Encode(req); err != nil {
			return err
		}
		resp, err := http.Post(allocator, "application/json", &buf)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			var r bytes.Buffer
			io.Copy(&r, resp.Body)
			return fmt.Errorf("Could not allocate port %d, status code %d, message: %s", p.SourcePort, resp.StatusCode, r.String())
		}
	}
	return nil
}

func closePorts(ports []PortForward) error {
	var retErr error
	for _, p := range ports {
		var buf bytes.Buffer
		req := removePortReq{
			Protocol:      p.Protocol,
			SourcePort:    p.SourcePort,
			TargetService: p.TargetService,
			TargetPort:    p.TargetPort,
		}
		if err := json.NewEncoder(&buf).Encode(req); err != nil {
			retErr = err
			continue
		}
		resp, err := http.Post(p.RemoveAddr, "application/json", &buf)
		if err != nil {
			retErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			retErr = fmt.Errorf("Could not deallocate port %d, status code: %d", p.SourcePort, resp.StatusCode)
			continue
		}
	}
	return retErr
}

func createKustomizationChain(r soft.RepoFS, path string) error {
	for p := filepath.Clean(path); p != "/"; {
		parent, child := filepath.Split(p)
		kustPath := filepath.Join(parent, "kustomization.yaml")
		kust, err := soft.ReadKustomization(r, kustPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				k := gio.NewKustomization()
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

type Resource struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Info        string            `json:"info"`
	Annotations map[string]string `json:"annotations"`
}

type ReleaseResources struct {
	Release     Release
	Helm        []Resource
	RenderedRaw []byte
}

// TODO(gio): rename to CommitApp
func installApp(
	repo soft.RepoIO,
	appDir string,
	name string,
	config any,
	resources CueAppData,
	data CueAppData,
	opts ...InstallOption,
) error {
	var o installOptions
	for _, i := range opts {
		i(&o)
	}
	dopts := []soft.DoOption{}
	if o.Branch != "" {
		dopts = append(dopts, soft.WithCommitToBranch(o.Branch))
	}
	if o.NoPull {
		dopts = append(dopts, soft.WithNoPull())
	}
	if o.NoPublish {
		dopts = append(dopts, soft.WithNoCommit())
	}
	if o.Force {
		dopts = append(dopts, soft.WithForce())
	}
	if o.NoLock {
		dopts = append(dopts, soft.WithNoLock())
	}
	_, err := repo.Do(func(r soft.RepoFS) (string, error) {
		if err := r.RemoveAll(appDir); err != nil {
			return "", err
		}
		resourcesDir := path.Join(appDir, "resources")
		if err := r.CreateDir(resourcesDir); err != nil {
			return "", err
		}
		if err := func() error {
			if err := soft.WriteFile(r, path.Join(appDir, gitIgnoreFileName), includeEverything); err != nil {
				return err
			}
			if err := soft.WriteYaml(r, path.Join(appDir, configFileName), config); err != nil {
				return err
			}
			if err := soft.WriteJson(r, path.Join(appDir, "config.json"), config); err != nil {
				return err
			}
			for name, contents := range data {
				if name == "config.json" || name == "kustomization.yaml" || name == "resources" {
					return fmt.Errorf("%s is forbidden", name)
				}
				w, err := r.Writer(path.Join(appDir, name))
				if err != nil {
					return err
				}
				defer w.Close()
				if _, err := w.Write(contents); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return "", err
		}
		if err := func() error {
			if err := createKustomizationChain(r, resourcesDir); err != nil {
				return err
			}
			appKust := gio.NewKustomization()
			for name, contents := range resources {
				appKust.AddResources(name)
				w, err := r.Writer(path.Join(resourcesDir, name))
				if err != nil {
					return err
				}
				defer w.Close()
				if _, err := w.Write(contents); err != nil {
					return err
				}
			}
			if err := soft.WriteYaml(r, path.Join(resourcesDir, "kustomization.yaml"), appKust); err != nil {
				return err
			}
			return nil
		}(); err != nil {
			return "", err
		}
		return fmt.Sprintf("install: %s", name), nil
	}, dopts...)
	return err
}

// TODO(gio): commit instanceId -> appDir mapping as well
func (m *AppManager) Install(
	app EnvApp,
	instanceId string,
	appDir string,
	namespace string,
	values map[string]any,
	opts ...InstallOption,
) (ReleaseResources, error) {
	o := &installOptions{}
	for _, i := range opts {
		i(o)
	}
	if !o.NoLock {
		m.l.Lock()
		defer m.l.Unlock()
	}
	portFields := findPortFields(app.Schema())
	fakeReservations := map[string]reservePortResp{}
	for i, f := range portFields {
		fakeReservations[f] = reservePortResp{Port: i}
	}
	if err := setPortFields(values, fakeReservations); err != nil {
		return ReleaseResources{}, err
	}
	appDir = filepath.Clean(appDir)
	if !o.NoPull {
		if err := m.repo.Pull(); err != nil {
			return ReleaseResources{}, err
		}
	}
	opts = append(opts, WithNoPull())
	if err := m.nsc.Create(namespace); err != nil {
		return ReleaseResources{}, err
	}
	var env EnvConfig
	if o.Env != nil {
		env = *o.Env
	} else {
		var err error
		env, err = m.Config()
		if err != nil {
			return ReleaseResources{}, err
		}
	}
	var networks []Network
	if o.Networks != nil {
		networks = o.Networks
	} else {
		var err error
		networks, err = m.CreateNetworks(env)
		if err != nil {
			return ReleaseResources{}, err
		}
	}
	clusters, err := m.GetClusters()
	if err != nil {
		return ReleaseResources{}, err
	}
	var lg LocalChartGenerator
	if o.LG != nil {
		lg = o.LG
	} else {
		lg = GitRepositoryLocalChartGenerator{env.Id, env.Id}
	}
	release := Release{
		AppInstanceId: instanceId,
		Namespace:     namespace,
		RepoAddr:      m.repo.FullAddress(),
		AppDir:        appDir,
	}
	rendered, err := app.Render(release, env, networks, ToAccessConfigs(clusters), values, nil, m.vpnAPIClient)
	if err != nil {
		return ReleaseResources{}, err
	}
	reservators := map[string]string{}
	allocators := map[string]string{}
	for _, pf := range rendered.Ports {
		reservators[portFields[pf.SourcePort]] = pf.ReserveAddr
		allocators[portFields[pf.SourcePort]] = pf.Allocator
	}
	portReservations, err := reservePorts(reservators)
	if err != nil {
		return ReleaseResources{}, err
	}
	if err := setPortFields(values, portReservations); err != nil {
		return ReleaseResources{}, err
	}
	// TODO(gio): env might not have private domain
	imageRegistry := fmt.Sprintf("zot.%s", env.PrivateDomain)
	if o.FetchContainerImages {
		if err := pullContainerImages(instanceId, rendered.ContainerImages, imageRegistry, namespace, m.jc); err != nil {
			return ReleaseResources{}, err
		}
	}
	charts, err := pullHelmCharts(m.hf, rendered.HelmCharts, m.repo, "/helm-charts")
	if err != nil {
		return ReleaseResources{}, err
	}
	localCharts := generateLocalCharts(lg, charts)
	if o.FetchContainerImages {
		release.ImageRegistry = imageRegistry
	}
	rendered, err = app.Render(release, env, networks, ToAccessConfigs(clusters), values, localCharts, m.vpnAPIClient)
	if err != nil {
		return ReleaseResources{}, err
	}
	for _, ns := range rendered.Namespaces {
		if ns.Name == "" {
			return ReleaseResources{}, fmt.Errorf("namespace name missing")
		}
		if ns.Kubeconfig == "" {
			continue
		}
		nsc, err := NewNamespaceCreator(kube.KubeConfigOpts{KubeConfig: ns.Kubeconfig})
		if err != nil {
			return ReleaseResources{}, err
		}
		if err := nsc.Create(ns.Name); err != nil {
			return ReleaseResources{}, err
		}
	}
	if err := installApp(m.repo, appDir, rendered.Name, rendered.Config, rendered.Resources, rendered.Data, opts...); err != nil {
		return ReleaseResources{}, err
	}
	// TODO(gio): add ingress-nginx to release resources
	if err := openPorts(rendered.Ports, portReservations, allocators); err != nil {
		return ReleaseResources{}, err
	}
	for _, p := range rendered.ClusterProxies {
		if err := m.cnc.AddProxy(p.From, p.To); err != nil {
			return ReleaseResources{}, err
		}
	}
	return ReleaseResources{
		Release:     rendered.Config.Release,
		RenderedRaw: rendered.Raw,
		Helm:        extractHelm(rendered.Resources),
	}, nil
}

type helmRelease struct {
	Metadata struct {
		Name        string            `json:"name"`
		Namespace   string            `json:"namespace"`
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata"`
	Kind   string `json:"kind"`
	Status struct {
		Conditions []struct {
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"conditions"`
	} `json:"status,omitempty"`
}

func extractHelm(resources CueAppData) []Resource {
	ret := make([]Resource, 0, len(resources))
	for _, contents := range resources {
		var h helmRelease
		if err := yaml.Unmarshal(contents, &h); err != nil {
			panic(err) // TODO(gio): handle
		}
		if h.Kind == "HelmRelease" {
			res := Resource{
				Name:        h.Metadata.Name,
				Namespace:   h.Metadata.Namespace,
				Info:        fmt.Sprintf("%s/%s", h.Metadata.Namespace, h.Metadata.Name),
				Annotations: nil,
			}
			if h.Metadata.Annotations != nil {
				res.Annotations = h.Metadata.Annotations
				info, ok := h.Metadata.Annotations["dodo.cloud/installer-info"]
				if ok && len(info) != 0 {
					res.Info = info
				}
			}
			ret = append(ret, res)
		}
	}
	return ret
}

// TODO(gio): take app configuration from the repo
func (m *AppManager) Update(
	instanceId string,
	values map[string]any,
	opts ...InstallOption,
) (ReleaseResources, error) {
	m.l.Lock()
	defer m.l.Unlock()
	if err := m.repo.Pull(); err != nil {
		return ReleaseResources{}, err
	}
	env, err := m.Config()
	if err != nil {
		return ReleaseResources{}, err
	}
	instanceDir := filepath.Join(m.appDirRoot, instanceId)
	app, err := m.GetInstanceApp(instanceId)
	if err != nil {
		return ReleaseResources{}, err
	}
	instanceConfigPath := filepath.Join(instanceDir, "config.json")
	config, err := m.appConfig(instanceConfigPath)
	if err != nil {
		return ReleaseResources{}, err
	}
	renderedCfg, err := readRendered(m.repo, filepath.Join(instanceDir, "rendered.json"))
	if err != nil {
		return ReleaseResources{}, err
	}
	networks, err := m.CreateNetworks(env)
	if err != nil {
		return ReleaseResources{}, err
	}
	clusters, err := m.GetClusters()
	if err != nil {
		return ReleaseResources{}, err
	}
	rendered, err := app.Render(config.Release, env, networks, ToAccessConfigs(clusters), values, renderedCfg.LocalCharts, m.vpnAPIClient)
	if err != nil {
		return ReleaseResources{}, err
	}
	for _, ns := range rendered.Namespaces {
		if ns.Name == "" {
			return ReleaseResources{}, fmt.Errorf("namespace name missing")
		}
		if ns.Kubeconfig == "" {
			continue
		}
		nsc, err := NewNamespaceCreator(kube.KubeConfigOpts{KubeConfig: ns.Kubeconfig})
		if err != nil {
			return ReleaseResources{}, err
		}
		if err := nsc.Create(ns.Name); err != nil {
			return ReleaseResources{}, err
		}
	}
	if err := installApp(m.repo, instanceDir, rendered.Name, rendered.Config, rendered.Resources, rendered.Data, opts...); err != nil {
		return ReleaseResources{}, err
	}
	for _, ocp := range renderedCfg.Out.ClusterProxy {
		found := false
		for _, ncp := range rendered.ClusterProxies {
			if ocp == ncp {
				found = true
				break
			}
		}
		if !found {
			if err := m.cnc.RemoveProxy(ocp.From, ocp.To); err != nil {
				return ReleaseResources{}, err
			}
		}
	}
	for _, ncp := range rendered.ClusterProxies {
		found := false
		for _, ocp := range renderedCfg.Out.ClusterProxy {
			if ocp == ncp {
				found = true
				break
			}
		}
		if !found {
			if err := m.cnc.AddProxy(ncp.From, ncp.To); err != nil {
				return ReleaseResources{}, err
			}
		}
	}
	return ReleaseResources{
		Release:     rendered.Config.Release,
		RenderedRaw: rendered.Raw,
		Helm:        extractHelm(rendered.Resources),
	}, nil
}

func (m *AppManager) Remove(instanceId string) error {
	m.l.Lock()
	defer m.l.Unlock()
	if err := m.repo.Pull(); err != nil {
		return err
	}
	var cfg renderedInstance
	if _, err := m.repo.Do(func(r soft.RepoFS) (string, error) {
		instanceDir := filepath.Join(m.appDirRoot, instanceId)
		renderedCfg, err := readRendered(m.repo, filepath.Join(instanceDir, "rendered.json"))
		if err != nil {
			return "", err
		}
		cfg = renderedCfg
		r.RemoveAll(instanceDir)
		kustPath := filepath.Join(m.appDirRoot, "kustomization.yaml")
		kust, err := soft.ReadKustomization(r, kustPath)
		if err != nil {
			return "", err
		}
		kust.RemoveResources(instanceId)
		soft.WriteYaml(r, kustPath, kust)
		return fmt.Sprintf("uninstall: %s", instanceId), nil
	}); err != nil {
		return err
	}
	if err := closePorts(cfg.PortForward); err != nil {
		return err
	}
	for _, cp := range cfg.Out.ClusterProxy {
		if err := m.cnc.RemoveProxy(cp.From, cp.To); err != nil {
			return err
		}
	}
	for vmName, vmCfg := range cfg.Out.VM {
		if vmCfg.VPN.Enabled {
			if err := m.vpnAPIClient.ExpireNode(vmCfg.Username, vmName); err != nil {
				return err
			}
			if err := m.vpnAPIClient.ExpireKey(vmCfg.Username, vmCfg.VPN.AuthKey); err != nil {
				return err
			}
			if err := m.vpnAPIClient.RemoveNode(vmCfg.Username, vmName); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *AppManager) CreateNetworks(env EnvConfig) ([]Network, error) {
	ret := []Network{
		{
			Name:               "Public",
			IngressClass:       fmt.Sprintf("%s-ingress-public", env.InfraName),
			CertificateIssuer:  fmt.Sprintf("%s-public", env.Id),
			Domain:             env.Domain,
			AllocatePortAddr:   fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/allocate", env.InfraName),
			ReservePortAddr:    fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/reserve", env.InfraName),
			DeallocatePortAddr: fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/remove", env.InfraName),
		},
	}
	if env.PrivateDomain != "" {
		ret = append(ret, Network{
			Name:               "Private",
			IngressClass:       fmt.Sprintf("%s-ingress-private", env.Id),
			Domain:             env.PrivateDomain,
			AllocatePortAddr:   fmt.Sprintf("http://port-allocator.%s-ingress-private.svc.cluster.local/api/allocate", env.Id),
			ReservePortAddr:    fmt.Sprintf("http://port-allocator.%s-ingress-private.svc.cluster.local/api/reserve", env.Id),
			DeallocatePortAddr: fmt.Sprintf("http://port-allocator.%s-ingress-private.svc.cluster.local/api/remove", env.Id),
		})
	}
	n, err := m.GetAllAppInstances("network")
	if err != nil {
		return nil, err
	}
	for _, a := range n {
		ret = append(ret, Network{
			Name:               a.Input["name"].(string),
			IngressClass:       fmt.Sprintf("%s-ingress-public", env.InfraName),
			CertificateIssuer:  fmt.Sprintf("%s-public", env.Id),
			Domain:             a.Input["domain"].(string),
			AllocatePortAddr:   fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/allocate", env.InfraName),
			ReservePortAddr:    fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/reserve", env.InfraName),
			DeallocatePortAddr: fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/remove", env.InfraName),
		})
	}
	return ret, nil
}

func (m *AppManager) GetClusters() ([]cluster.State, error) {
	ret := []cluster.State{
		{
			Name: "default",
		},
	}
	files, err := m.repo.ListDir("/clusters")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ret, nil
		}
		return nil, err
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		cfgPath := filepath.Clean(filepath.Join("/clusters", f.Name(), "config.json"))
		var c cluster.State
		if err := soft.ReadJson(m.repo, cfgPath, &c); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		ret = append(ret, c)
	}
	return ret, nil
}

type installOptions struct {
	NoPull               bool
	NoPublish            bool
	Env                  *EnvConfig
	Networks             []Network
	Branch               string
	LG                   LocalChartGenerator
	FetchContainerImages bool
	Force                bool
	NoLock               bool
}

type InstallOption func(*installOptions)

func WithConfig(env *EnvConfig) InstallOption {
	return func(o *installOptions) {
		o.Env = env
	}
}

func WithNetworks(networks []Network) InstallOption {
	return func(o *installOptions) {
		o.Networks = networks
	}
}

func WithNoNetworks() InstallOption {
	return WithNetworks([]Network{})
}

func WithBranch(branch string) InstallOption {
	return func(o *installOptions) {
		o.Branch = branch
	}
}

func WithForce() InstallOption {
	return func(o *installOptions) {
		o.Force = true
	}
}

func WithLocalChartGenerator(lg LocalChartGenerator) InstallOption {
	return func(o *installOptions) {
		o.LG = lg
	}
}

func WithFetchContainerImages() InstallOption {
	return func(o *installOptions) {
		o.FetchContainerImages = true
	}
}

func WithNoPublish() InstallOption {
	return func(o *installOptions) {
		o.NoPublish = true
	}
}

func WithNoPull() InstallOption {
	return func(o *installOptions) {
		o.NoPull = true
	}
}

func WithNoLock() InstallOption {
	return func(o *installOptions) {
		o.NoLock = true
	}
}

// InfraAppmanager

type InfraAppManager struct {
	repoIO soft.RepoIO
	nsc    NamespaceCreator
	hf     HelmFetcher
	lg     LocalChartGenerator
}

func NewInfraAppManager(
	repoIO soft.RepoIO,
	nsc NamespaceCreator,
	hf HelmFetcher,
	lg LocalChartGenerator,
) (*InfraAppManager, error) {
	return &InfraAppManager{
		repoIO,
		nsc,
		hf,
		lg,
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

func (m *InfraAppManager) Install(app InfraApp, appDir string, namespace string, values map[string]any) (ReleaseResources, error) {
	appDir = filepath.Clean(appDir)
	if err := m.repoIO.Pull(); err != nil {
		return ReleaseResources{}, err
	}
	if err := m.nsc.Create(namespace); err != nil {
		return ReleaseResources{}, err
	}
	infra, err := m.Config()
	if err != nil {
		return ReleaseResources{}, err
	}
	release := Release{
		Namespace: namespace,
		RepoAddr:  m.repoIO.FullAddress(),
		AppDir:    appDir,
	}
	networks := m.CreateNetworks(infra)
	rendered, err := app.Render(release, infra, networks, values, nil)
	if err != nil {
		return ReleaseResources{}, err
	}
	charts, err := pullHelmCharts(m.hf, rendered.HelmCharts, m.repoIO, "/helm-charts")
	if err != nil {
		return ReleaseResources{}, err
	}
	localCharts := generateLocalCharts(m.lg, charts)
	rendered, err = app.Render(release, infra, networks, values, localCharts)
	if err != nil {
		return ReleaseResources{}, err
	}
	if err := installApp(m.repoIO, appDir, rendered.Name, rendered.Config, rendered.Resources, rendered.Data); err != nil {
		return ReleaseResources{}, err
	}
	return ReleaseResources{
		Release:     rendered.Config.Release,
		RenderedRaw: rendered.Raw,
		Helm:        extractHelm(rendered.Resources),
	}, nil
}

// TODO(gio): take app configuration from the repo
func (m *InfraAppManager) Update(
	instanceId string,
	values map[string]any,
	opts ...InstallOption,
) (ReleaseResources, error) {
	if err := m.repoIO.Pull(); err != nil {
		return ReleaseResources{}, err
	}
	infra, err := m.Config()
	if err != nil {
		return ReleaseResources{}, err
	}
	instanceDir := filepath.Join("/infrastructure", instanceId)
	appCfg, err := GetCueAppData(m.repoIO, instanceDir)
	if err != nil {
		return ReleaseResources{}, err
	}
	app, err := NewCueInfraApp(appCfg)
	if err != nil {
		return ReleaseResources{}, err
	}
	instanceConfigPath := filepath.Join(instanceDir, "config.json")
	config, err := m.appConfig(instanceConfigPath)
	if err != nil {
		return ReleaseResources{}, err
	}
	renderedCfg, err := readRendered(m.repoIO, filepath.Join(instanceDir, "rendered.json"))
	if err != nil {
		return ReleaseResources{}, err
	}
	networks := m.CreateNetworks(infra)
	rendered, err := app.Render(config.Release, infra, networks, values, renderedCfg.LocalCharts)
	if err != nil {
		return ReleaseResources{}, err
	}
	if err := installApp(m.repoIO, instanceDir, rendered.Name, rendered.Config, rendered.Resources, rendered.Data, opts...); err != nil {
		return ReleaseResources{}, err
	}
	return ReleaseResources{
		Release:     rendered.Config.Release,
		RenderedRaw: rendered.Raw,
		Helm:        extractHelm(rendered.Resources),
	}, nil
}

func (m *InfraAppManager) CreateNetworks(infra InfraConfig) []InfraNetwork {
	return []InfraNetwork{
		{
			Name:               "Public",
			IngressClass:       fmt.Sprintf("%s-ingress-public", infra.Name),
			CertificateIssuer:  fmt.Sprintf("%s-public", infra.Name),
			AllocatePortAddr:   fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/allocate", infra.Name),
			ReservePortAddr:    fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/reserve", infra.Name),
			DeallocatePortAddr: fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/remove", infra.Name),
		},
	}
}

func pullHelmCharts(hf HelmFetcher, charts HelmCharts, rfs soft.RepoFS, root string) (map[string]string, error) {
	ret := make(map[string]string)
	for name, chart := range charts.Git {
		chartRoot := filepath.Join(root, name)
		ret[name] = chartRoot
		if err := hf.Pull(chart, rfs, chartRoot); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func generateLocalCharts(g LocalChartGenerator, charts map[string]string) map[string]helmv2.HelmChartTemplateSpec {
	ret := make(map[string]helmv2.HelmChartTemplateSpec)
	for name, path := range charts {
		ret[name] = g.Generate(path)
	}
	return ret
}

func pullContainerImages(appName string, imgs map[string]ContainerImage, registry, namespace string, jc JobCreator) error {
	for _, img := range imgs {
		name := fmt.Sprintf("copy-image-%s-%s-%s-%s", appName, img.Repository, img.Name, img.Tag)
		if err := jc.Create(name, namespace, "giolekva/skopeo:latest", []string{
			"skopeo",
			"--insecure-policy",
			"copy",
			"--dest-tls-verify=false", // TODO(gio): enable
			"--multi-arch=all",
			fmt.Sprintf("docker://%s/%s/%s:%s", img.Registry, img.Repository, img.Name, img.Tag),
			fmt.Sprintf("docker://%s/%s/%s:%s", registry, img.Repository, img.Name, img.Tag),
		}); err != nil {
			return err
		}
	}
	return nil
}

type renderedInstance struct {
	LocalCharts map[string]helmv2.HelmChartTemplateSpec `json:"localCharts"`
	PortForward []PortForward                           `json:"portForward"`
	Out         outRendered                             `json:"out"`
}

type outRendered struct {
	ClusterProxy map[string]ClusterProxy
	VM           map[string]vmRendered `json:"vm"`
}

type vmRendered struct {
	Username string `json:"username"`
	VPN      struct {
		Enabled bool   `json:"enabled"`
		AuthKey string `json:"authKey"`
	} `json:"vpn"`
}

func readRendered(fs soft.RepoFS, path string) (renderedInstance, error) {
	r, err := fs.Reader(path)
	if err != nil {
		return renderedInstance{}, err
	}
	defer r.Close()
	var cfg renderedInstance
	if err := json.NewDecoder(r).Decode(&cfg); err != nil {
		return renderedInstance{}, err
	}
	return cfg, nil
}

func findPortFields(scm Schema) []string {
	switch scm.Kind() {
	case KindBoolean:
		return []string{}
	case KindInt:
		return []string{}
	case KindString:
		return []string{}
	case KindStruct:
		ret := []string{}
		for _, f := range scm.Fields() {
			for _, p := range findPortFields(f.Schema) {
				if p == "" {
					ret = append(ret, f.Name)
				} else {
					ret = append(ret, fmt.Sprintf("%s.%s", f.Name, p))
				}
			}
		}
		return ret
	case KindNetwork:
		return []string{}
	case KindMultiNetwork:
		return []string{}
	case KindAuth:
		return []string{}
	case KindSSHKey:
		return []string{}
	case KindNumber:
		return []string{}
	case KindArrayString:
		return []string{}
	case KindPort:
		return []string{""}
	case KindVPNAuthKey:
		return []string{}
	case KindCluster:
		return []string{}
	default:
		panic("MUST NOT REACH!")
	}
}

func setPortFields(values map[string]any, ports map[string]reservePortResp) error {
	for p, r := range ports {
		if err := setPortField(values, p, r.Port); err != nil {
			return err
		}
	}
	return nil
}

func setPortField(values map[string]any, field string, port int) error {
	f := strings.SplitN(field, ".", 2)
	if len(f) == 2 {
		var sub map[string]any
		if s, ok := values[f[0]]; ok {
			sub, ok = s.(map[string]any)
			if !ok {
				return fmt.Errorf("expected map")
			}
		} else {
			sub = map[string]any{}
			values[f[0]] = sub
		}
		if err := setPortField(sub, f[1], port); err != nil {
			return err
		}
	} else {
		values[f[0]] = port
	}
	return nil
}

type Cluster struct {
	Name             string `json:"name"`
	Kubeconfig       string `json:"kubeconfig"`
	IngressClassName string `json:"ingressClassName"`
}

func ClusterStateToAccessConfig(c cluster.State) Cluster {
	return Cluster{
		Name:             c.Name,
		Kubeconfig:       c.Kubeconfig,
		IngressClassName: c.IngressClassName,
	}
}

func ToAccessConfigs(clusters []cluster.State) []Cluster {
	ret := make([]Cluster, 0, len(clusters))
	for _, c := range clusters {
		ret = append(ret, ClusterStateToAccessConfig(c))
	}
	return ret
}
