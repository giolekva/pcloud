package installer

import (
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

func (m *AppManager) FindInstance(name string) (AppConfig, error) {
	return m.repoIO.FindInstance(appDir, name)
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
	// if err := m.repoIO.Fetch(); err != nil {
	// 	return err
	// }
	suffix, err := suffixGen.Generate()
	if err != nil {
		return err
	}
	namespaces := make([]string, len(app.Namespaces))
	for i, n := range app.Namespaces {
		ns, err := ns.Generate(n)
		if err != nil {
			return err
		}
		namespaces[i] = ns + suffix
	}
	for _, n := range namespaces {
		if err := m.nsCreator.Create(n); err != nil {
			return err
		}
	}
	globalConfig, err := m.repoIO.ReadConfig()
	if err != nil {
		return err
	}
	all := map[string]any{
		"Global": globalConfig.Values,
		"Values": config,
	}
	if len(namespaces) > 0 {
		all["Release"] = map[string]any{
			"Namespace": namespaces[0],
		}
	}
	return m.repoIO.InstallApp(
		app,
		filepath.Join(appDir, app.Name+suffix),
		all)
}

func (m *AppManager) Update(app App, instanceId string, config map[string]any) error {
	// if err := m.repoIO.Fetch(); err != nil {
	// 	return err
	// }
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
	all := map[string]any{
		"Global":  globalConfig.Values,
		"Values":  config,
		"Release": appConfig.Config["Release"],
	}
	return m.repoIO.InstallApp(app, instanceDir, all)
}
