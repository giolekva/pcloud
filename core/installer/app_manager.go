package installer

import (
	"io/fs"

	"golang.org/x/exp/slices"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
)

const kustomizationFileName = "kustomization.yaml"

type AppRepository interface {
	Find(name string) (*App, error)
}

type AppManager struct {
	fs       billy.Filesystem
	config   Config
	appRepo  AppRepository
	rootKust *Kustomization
}

func NewAppManager(fs billy.Filesystem, config Config, appRepo AppRepository) (*AppManager, error) {
	rootKustF, err := fs.Open(kustomizationFileName)
	if err != nil {
		return nil, err
	}
	defer rootKustF.Close()
	rootKust, err := ReadKustomization(rootKustF)
	if err != nil {
		return nil, err
	}
	return &AppManager{
		fs,
		config,
		appRepo,
		rootKust,
	}, nil
}

func (m *AppManager) Install(name string) error {
	app, err := m.appRepo.Find(name)
	if err != nil {
		return nil
	}
	if err := util.RemoveAll(m.fs, name); err != nil {
		return err
	}
	if err := m.fs.MkdirAll(name, fs.ModePerm); err != nil {
		return nil
	}
	appRoot, err := m.fs.Chroot(name)
	if err != nil {
		return err
	}
	appKust := NewKustomization()
	for _, t := range app.Templates {
		out, err := appRoot.Create(t.Name())
		if err != nil {
			return err
		}
		defer out.Close()
		if err := t.Execute(out, m.config); err != nil {
			return err
		}
		appKust.Resources = append(appKust.Resources, t.Name())
	}
	appKustF, err := appRoot.Create(kustomizationFileName)
	if err != nil {
		return err
	}
	defer appKustF.Close()
	if err := appKust.Write(appKustF); err != nil {
		return err
	}
	if slices.Contains(m.rootKust.Resources, name) {
		return nil
	}
	m.rootKust.Resources = append(m.rootKust.Resources, name)
	rootKustF, err := m.fs.Create(kustomizationFileName)
	if err != nil {
		return err
	}
	defer rootKustF.Close()
	return m.rootKust.Write(rootKustF)
}
