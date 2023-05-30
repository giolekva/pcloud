package installer

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/slices"

	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"sigs.k8s.io/yaml"
)

const appDirName = "apps"
const configFileName = "config.yaml"
const kustomizationFileName = "kustomization.yaml"

type AppManager struct {
	repo   *git.Repository
	signer ssh.Signer
}

// func NewAppManager(repo *git.Repository, fs billy.Filesystem, config Config, appRepo AppRepository) (*AppManager, error) {
func NewAppManager(repo *git.Repository, signer ssh.Signer) (*AppManager, error) {
	return &AppManager{
		repo,
		signer,
	}, nil
}

func (m *AppManager) Config() (Config, error) {
	wt, err := m.repo.Worktree()
	if err != nil {
		return Config{}, err
	}
	configF, err := wt.Filesystem.Open(configFileName)
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
	wt, err := m.repo.Worktree()
	if err != nil {
		return nil, err
	}
	configF, err := wt.Filesystem.Open(wt.Filesystem.Join(appDirName, name, configFileName))
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
	wt, err := m.repo.Worktree()
	if err != nil {
		return err
	}
	globalConfig, err := m.Config()
	if err != nil {
		return err
	}
	all := map[string]any{
		"Global": globalConfig.Values,
		"Values": config,
	}
	appsRoot, err := wt.Filesystem.Chroot(appDirName)
	if err != nil {
		return err
	}
	rootKustF, err := appsRoot.Open(kustomizationFileName)
	if err != nil {
		return err
	}
	defer rootKustF.Close()
	rootKust, err := ReadKustomization(rootKustF)
	if err != nil {
		return err
	}
	appRoot, err := appsRoot.Chroot(app.Name)
	if err != nil {
		return err
	}
	if err := util.RemoveAll(appRoot, app.Name); err != nil {
		return err
	}
	if err := appRoot.MkdirAll(app.Name, fs.ModePerm); err != nil {
		return nil
	}
	appKust := NewKustomization()
	for _, t := range app.Templates {
		out, err := appRoot.Create(t.Name())
		if err != nil {
			return err
		}
		defer out.Close()
		if err := t.Execute(out, all); err != nil {
			return err
		}
		appKust.Resources = append(appKust.Resources, t.Name())
	}
	{
		out, err := appRoot.Create(configFileName)
		if err != nil {
			return err
		}
		defer out.Close()
		configBytes, err := yaml.Marshal(config)
		if err != nil {
			return err
		}
		if _, err := out.Write(configBytes); err != nil {
			return err
		}
	}
	appKustF, err := appRoot.Create(kustomizationFileName)
	if err != nil {
		return err
	}
	defer appKustF.Close()
	if err := appKust.Write(appKustF); err != nil {
		return err
	}
	if !slices.Contains(rootKust.Resources, app.Name) {
		rootKust.Resources = append(rootKust.Resources, app.Name)
		rootKustFW, err := appsRoot.Create(kustomizationFileName)
		if err != nil {
			return err
		}
		defer rootKustFW.Close()
		if err := rootKust.Write(rootKustFW); err != nil {
			return err
		}
	}
	// Commit and push
	if err := wt.AddGlob("*"); err != nil {
		return err
	}
	if _, err := wt.Commit(fmt.Sprintf("install: %s", app.Name), &git.CommitOptions{
		Author: &object.Signature{
			Name: "pcloud-appmanager",
			When: time.Now(),
		},
	}); err != nil {
		return err
	}
	return m.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth(m.signer),
	})
}

func auth(signer ssh.Signer) *gitssh.PublicKeys {
	return &gitssh.PublicKeys{
		Signer: signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// TODO(giolekva): verify server public key
				// fmt.Printf("## %s || %s -- \n", serverPubKey, ssh.MarshalAuthorizedKey(key))
				return nil
			},
		},
	}
}
