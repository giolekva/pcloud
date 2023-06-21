package installer

import (
	"fmt"
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

const appDirName = "apps"
const configFileName = "config.yaml"
const kustomizationFileName = "kustomization.yaml"

type AppManager struct {
	repoIO RepoIO
}

func NewAppManager(repoIO RepoIO) (*AppManager, error) {
	return &AppManager{
		repoIO,
	}, nil
}

func (m *AppManager) Config() (Config, error) {
	configF, err := m.repoIO.Reader(configFileName)
	if err != nil {
		return Config{}, err
	}
	defer configF.Close()
	config, err := ReadConfig(configF)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

func (m *AppManager) AppConfig(name string) (map[string]any, error) {
	configF, err := m.repoIO.Reader(fmt.Sprintf("%s/%s/%s", appDirName, name, configFileName))
	if err != nil {
		return nil, err
	}
	defer configF.Close()
	var cfg map[string]any
	contents, err := ioutil.ReadAll(configF)
	if err != nil {
		return cfg, err
	}
	err = yaml.UnmarshalStrict(contents, &cfg)
	return cfg, err
}

func (m *AppManager) Install(app App, config map[string]any) error {
	// if err := m.repoIO.Fetch(); err != nil {
	// 	return err
	// }
	globalConfig, err := m.Config()
	if err != nil {
		return err
	}
	all := map[string]any{
		"Global": globalConfig.Values,
		"Values": config,
	}
	return m.repoIO.InstallApp(app, "apps", all)
}
