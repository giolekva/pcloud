package installer

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"path"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"
)

type RepoIO interface {
	Fetch() error
	ReadKustomization(path string) (*Kustomization, error)
	WriteKustomization(path string, kust Kustomization) error
	WriteYaml(path string, data any) error
	CommitAndPush(message string) error
	Reader(path string) (io.ReadCloser, error)
	Writer(path string) (io.WriteCloser, error)
	CreateDir(path string) error
	RemoveDir(path string) error
	InstallApp(app App, path string, values map[string]any) error
}

type repoIO struct {
	repo   *git.Repository
	signer ssh.Signer
}

func NewRepoIO(repo *git.Repository, signer ssh.Signer) RepoIO {
	return &repoIO{
		repo,
		signer,
	}
}

func (r *repoIO) Fetch() error {
	err := r.repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth(r.signer),
		Force:      true,
	})
	if err == nil || err == git.NoErrAlreadyUpToDate {
		return nil
	}
	return err
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

func (r *repoIO) InstallApp(app App, root string, values map[string]any) error {
	{
		appsKustPath := path.Join(root, "kustomization.yaml")
		appsKust, err := r.ReadKustomization(appsKustPath)
		if err != nil {
			return err
		}
		appsKust.AddResources(app.Name)
		if err := r.WriteKustomization(appsKustPath, *appsKust); err != nil {
			return err
		}
	}
	appRootDir := path.Join(root, app.Name)
	{
		if err := r.RemoveDir(appRootDir); err != nil {
			return err
		}
		if err := r.CreateDir(appRootDir); err != nil {
			return err
		}
		if err := r.WriteYaml(path.Join(appRootDir, configFileName), values); err != nil {
			return err
		}
	}
	{
		appKust := NewKustomization()
		for _, t := range app.Templates {
			appKust.AddResources(t.Name())
			out, err := r.Writer(path.Join(appRootDir, t.Name()))
			if err != nil {
				return err
			}
			defer out.Close()
			if err := t.Execute(out, values); err != nil {
				return err
			}
		}
		if err := r.WriteKustomization(path.Join(appRootDir, "kustomization.yaml"), appKust); err != nil {
			return err
		}
	}
	return r.CommitAndPush(fmt.Sprintf("install: %s", app.Name))
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
