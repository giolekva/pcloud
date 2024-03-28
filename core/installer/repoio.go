package installer

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"

	"github.com/giolekva/pcloud/core/installer/soft"
)

type RepoIO interface {
	Addr() string
	Pull() error
	ReadConfig() (Config, error)
	ReadAppConfig(path string) (AppConfig, error)
	ReadKustomization(path string) (*Kustomization, error)
	WriteKustomization(path string, kust Kustomization) error
	ReadYaml(path string) (any, error)
	WriteYaml(path string, data any) error
	CommitAndPush(message string) error
	WriteCommitAndPush(path, contents, message string) error
	Reader(path string) (io.ReadCloser, error)
	Writer(path string) (io.WriteCloser, error)
	CreateDir(path string) error
	RemoveDir(path string) error
	InstallApp(app App, path string, values map[string]any, derived Derived) error
	RemoveApp(path string) error
	FindAllInstances(root string, appId string) ([]AppConfig, error)
	FindInstance(root string, id string) (AppConfig, error)
}

type repoIO struct {
	repo   *soft.Repository
	signer ssh.Signer
	l      sync.Locker
}

func NewRepoIO(repo *soft.Repository, signer ssh.Signer) RepoIO {
	return &repoIO{
		repo,
		signer,
		&sync.Mutex{},
	}
}

func (r *repoIO) Addr() string {
	return r.repo.Addr.Addr
}

func (r *repoIO) Pull() error {
	r.l.Lock()
	defer r.l.Unlock()
	return r.pullWithoutLock()
}

func (r *repoIO) pullWithoutLock() error {
	wt, err := r.repo.Worktree()
	if err != nil {
		fmt.Printf("EEEER wt: %s\b", err)
		return nil
	}
	err = wt.Pull(&git.PullOptions{
		Auth:  auth(r.signer),
		Force: true,
	})
	// TODO(gio): propagate error
	if err != nil {
		fmt.Printf("EEEER: %s\b", err)
	}
	return nil
}

func (r *repoIO) ReadConfig() (Config, error) {
	configF, err := r.Reader(configFileName)
	if err != nil {
		return Config{}, err
	}
	defer configF.Close()
	var cfg Config
	if err := ReadYaml(configF, &cfg); err != nil {
		return Config{}, err
	} else {
		return cfg, nil
	}
}

func (r *repoIO) ReadAppConfig(path string) (AppConfig, error) {
	configF, err := r.Reader(path)
	if err != nil {
		return AppConfig{}, err
	}
	defer configF.Close()
	var cfg AppConfig
	if err := ReadYaml(configF, &cfg); err != nil {
		return AppConfig{}, err
	} else {
		return cfg, nil
	}
}

func (r *repoIO) ReadKustomization(path string) (*Kustomization, error) {
	inp, err := r.Reader(path)
	if err != nil {
		return nil, err
	}
	defer inp.Close()
	return ReadKustomization(inp)
}

func (r *repoIO) Reader(path string) (io.ReadCloser, error) {
	wt, err := r.repo.Worktree()
	if err != nil {
		return nil, err
	}
	return wt.Filesystem.Open(path)
}

func (r *repoIO) Writer(path string) (io.WriteCloser, error) {
	wt, err := r.repo.Worktree()
	if err != nil {
		return nil, err
	}
	if err := wt.Filesystem.MkdirAll(filepath.Dir(path), fs.ModePerm); err != nil {
		return nil, err
	}
	return wt.Filesystem.Create(path)
}

func (r *repoIO) WriteKustomization(path string, kust Kustomization) error {
	out, err := r.Writer(path)
	if err != nil {
		return err
	}
	return kust.Write(out)
}

func (r *repoIO) WriteYaml(path string, data any) error {
	out, err := r.Writer(path)
	if err != nil {
		return err
	}
	serialized, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := out.Write(serialized); err != nil {
		return err
	}
	return nil
}

func (r *repoIO) ReadYaml(path string) (any, error) {
	inp, err := r.Reader(path)
	if err != nil {
		return nil, err
	}
	data := make(map[string]any)
	if err := ReadYaml(inp, &data); err != nil {
		return nil, err
	}
	return data, err
}

func (r *repoIO) WriteCommitAndPush(path, contents, message string) error {
	w, err := r.Writer(path)
	if err != nil {
		return err
	}
	defer w.Close()
	if _, err := io.WriteString(w, contents); err != nil {
		return err
	}
	return r.CommitAndPush(message)
}

func (r *repoIO) CommitAndPush(message string) error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	if err := wt.AddGlob("*"); err != nil {
		return err
	}
	if _, err := wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name: "pcloud-installer",
			When: time.Now(),
		},
	}); err != nil {
		return err
	}
	return r.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth(r.signer),
	})
}

func (r *repoIO) CreateDir(path string) error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	return wt.Filesystem.MkdirAll(path, fs.ModePerm)
}

func (r *repoIO) RemoveDir(path string) error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return err
	}
	err = util.RemoveAll(wt.Filesystem, path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

type Release struct {
	Namespace string `json:"namespace"`
}

type Derived struct {
	Release Release        `json:"release"`
	Global  Values         `json:"global"`
	Values  map[string]any `json:"input"` // TODO(gio): rename to input
}

type AppConfig struct {
	Id      string         `json:"id"`
	AppId   string         `json:"appId"`
	Config  map[string]any `json:"config"`
	Derived Derived        `json:"derived"`
}

func (r *repoIO) InstallApp(app App, appRootDir string, values map[string]any, derived Derived) error {
	r.l.Lock()
	defer r.l.Unlock()
	if err := r.pullWithoutLock(); err != nil {
		return err
	}
	if !filepath.IsAbs(appRootDir) {
		return fmt.Errorf("Expected absolute path: %s", appRootDir)
	}
	appRootDir = filepath.Clean(appRootDir)
	for p := appRootDir; p != "/"; {
		parent, child := filepath.Split(p)
		kustPath := filepath.Join(parent, "kustomization.yaml")
		kust, err := r.ReadKustomization(kustPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				k := NewKustomization()
				kust = &k
			} else {
				return err
			}
		}
		kust.AddResources(child)
		if err := r.WriteKustomization(kustPath, *kust); err != nil {
			return err
		}
		p = filepath.Clean(parent)
	}
	{
		if err := r.RemoveDir(appRootDir); err != nil {
			return err
		}
		if err := r.CreateDir(appRootDir); err != nil {
			return err
		}
		cfg := AppConfig{
			AppId:   app.Name(),
			Config:  values,
			Derived: derived,
		}
		if err := r.WriteYaml(path.Join(appRootDir, configFileName), cfg); err != nil {
			return err
		}
	}
	{
		appKust := NewKustomization()
		rendered, err := app.Render(derived)
		if err != nil {
			return err
		}
		for name, contents := range rendered.Resources {
			appKust.AddResources(name)
			out, err := r.Writer(path.Join(appRootDir, name))
			if err != nil {
				return err
			}
			defer out.Close()
			if _, err := out.Write(contents); err != nil {
				return err
			}
		}
		if err := r.WriteKustomization(path.Join(appRootDir, "kustomization.yaml"), appKust); err != nil {
			return err
		}
	}
	return r.CommitAndPush(fmt.Sprintf("install: %s", app.Name()))
}

func (r *repoIO) RemoveApp(appRootDir string) error {
	r.l.Lock()
	defer r.l.Unlock()
	r.RemoveDir(appRootDir)
	parent, child := filepath.Split(appRootDir)
	kustPath := filepath.Join(parent, "kustomization.yaml")
	kust, err := r.ReadKustomization(kustPath)
	if err != nil {
		return err
	}
	kust.RemoveResources(child)
	r.WriteKustomization(kustPath, *kust)
	return r.CommitAndPush(fmt.Sprintf("uninstall: %s", child))
}

func (r *repoIO) FindAllInstances(root string, name string) ([]AppConfig, error) {
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("Expected absolute path: %s", root)
	}
	kust, err := r.ReadKustomization(filepath.Join(root, "kustomization.yaml"))
	if err != nil {
		return nil, err
	}
	ret := make([]AppConfig, 0)
	for _, app := range kust.Resources {
		cfg, err := r.ReadAppConfig(filepath.Join(root, app, "config.yaml"))
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

func (r *repoIO) FindInstance(root string, id string) (AppConfig, error) {
	if !filepath.IsAbs(root) {
		return AppConfig{}, fmt.Errorf("Expected absolute path: %s", root)
	}
	kust, err := r.ReadKustomization(filepath.Join(root, "kustomization.yaml"))
	if err != nil {
		return AppConfig{}, err
	}
	for _, app := range kust.Resources {
		if app == id {
			cfg, err := r.ReadAppConfig(filepath.Join(root, app, "config.yaml"))
			if err != nil {
				return AppConfig{}, err
			}
			cfg.Id = id
			return cfg, nil
		}
	}
	return AppConfig{}, nil
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

func ReadYaml[T any](r io.Reader, o *T) error {
	if contents, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		return yaml.UnmarshalStrict(contents, o)
	}
}

func deriveValues(values any, schema Schema, networks []Network) (map[string]any, error) {
	ret := make(map[string]any)
	for k, v := range values.(map[string]any) { // TODO(giolekva): validate
		def, ok := schema.Fields()[k]
		if !ok {
			return nil, fmt.Errorf("Field not found: %s", k)
		}
		switch def.Kind() {
		case KindBoolean:
			ret[k] = v
		case KindString:
			ret[k] = v
		case KindNetwork:
			n, err := findNetwork(networks, v.(string)) // TODO(giolekva): validate
			if err != nil {
				return nil, err
			}
			ret[k] = n
		case KindAuth:
			r, err := deriveValues(v, AuthSchema, networks)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		case KindStruct:
			r, err := deriveValues(v, def, networks)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		default:
			return nil, fmt.Errorf("Should not reach!")
		}
	}
	return ret, nil
}

func findNetwork(networks []Network, name string) (Network, error) {
	for _, n := range networks {
		if n.Name == name {
			return n, nil
		}
	}
	return Network{}, fmt.Errorf("Network not found: %s", name)
}

type Network struct {
	Name              string `json:"name,omitempty"`
	IngressClass      string `json:"ingressClass,omitempty"`
	CertificateIssuer string `json:"certificateIssuer,omitempty"`
	Domain            string `json:"domain,omitempty"`
}
