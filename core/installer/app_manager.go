package installer

import (
	"fmt"
	"io/ioutil"
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
	return m.repoIO.ReadConfig()
}

func (m *AppManager) FindAllInstances(name string) ([]AppConfig, error) {
	return m.repoIO.FindAllInstances(appDir, name)
}

func (m *AppManager) FindInstance(id string) (AppConfig, error) {
	return m.repoIO.FindInstance(appDir, id)
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

func (m *AppManager) Install(app App, ns NamespaceGenerator, suffixGen SuffixGenerator, config map[string]any) error {
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	suffix, err := suffixGen.Generate()
	if err != nil {
		return err
	}
	nms, err := ns.Generate(app.Namespace())
	if err != nil {
		return err
	}
	nms = nms + suffix
	if err := m.nsCreator.Create(nms); err != nil {
		return err
	}
	globalConfig, err := m.repoIO.ReadConfig()
	if err != nil {
		return err
	}
	derivedValues, err := deriveValues(config, app.Schema(), CreateNetworks(globalConfig))
	if err != nil {
		fmt.Println(err)
		return err
	}
	derived := Derived{
		Global: globalConfig.Values,
		Values: derivedValues,
	}
	derived.Release.Namespace = nms
	fmt.Printf("%+v\n", derived)
	err = m.repoIO.InstallApp(
		app,
		filepath.Join(appDir, app.Name()+suffix),
		config,
		derived,
	)
	fmt.Println(err)
	return err
}

func (m *AppManager) Update(app App, instanceId string, config map[string]any) error {
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	globalConfig, err := m.repoIO.ReadConfig()
	if err != nil {
		return err
	}
	instanceDir := filepath.Join(appDir, instanceId)
	instanceConfigPath := filepath.Join(instanceDir, configFileName)
	appConfig, err := m.repoIO.ReadAppConfig(instanceConfigPath)
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
	return m.repoIO.InstallApp(app, instanceDir, config, derived)
}

func (m *AppManager) Remove(instanceId string) error {
	if err := m.repoIO.Pull(); err != nil {
		return err
	}
	return m.repoIO.RemoveApp(filepath.Join(appDir, instanceId))
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
