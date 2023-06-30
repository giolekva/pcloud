package installer

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

const appDirName = "apps"
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
	// TODO(giolekva): use ns suffix for app directory
	return m.repoIO.InstallApp(
		app,
		filepath.Join("/apps", app.Name+suffix),
		all)
}
