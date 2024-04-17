package installer

import (
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"net"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"

	"github.com/giolekva/pcloud/core/installer/soft"
)

type RepoFS interface {
	Reader(path string) (io.ReadCloser, error)
	Writer(path string) (io.WriteCloser, error)
	CreateDir(path string) error
	RemoveDir(path string) error
}

type AtomicOp func(r RepoFS) (string, error)

type RepoIO interface {
	RepoFS
	FullAddress() string
	Pull() error
	CommitAndPush(message string) error
	Atomic(op AtomicOp) error
}

type repoFS struct {
	fs billy.Filesystem
}

func (r *repoFS) Reader(path string) (io.ReadCloser, error) {
	return r.fs.Open(path)
}

func (r *repoFS) Writer(path string) (io.WriteCloser, error) {
	if err := r.fs.MkdirAll(filepath.Dir(path), fs.ModePerm); err != nil {
		return nil, err
	}
	return r.fs.Create(path)
}

func (r *repoFS) CreateDir(path string) error {
	return r.fs.MkdirAll(path, fs.ModePerm)
}

func (r *repoFS) RemoveDir(path string) error {
	if err := util.RemoveAll(r.fs, path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

type repoIO struct {
	*repoFS
	repo   *soft.Repository
	signer ssh.Signer
	l      sync.Locker
}

func NewRepoIO(repo *soft.Repository, signer ssh.Signer) (RepoIO, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	return &repoIO{
		&repoFS{wt.Filesystem},
		repo,
		signer,
		&sync.Mutex{},
	}, nil
}

func (r *repoIO) FullAddress() string {
	return r.repo.Addr.FullAddress()
}

func (r *repoIO) Pull() error {
	r.l.Lock()
	defer r.l.Unlock()
	return r.pullWithoutLock()
}

func (r *repoIO) pullWithoutLock() error {
	wt, err := r.repo.Worktree()
	if err != nil {
		return nil
	}
	err = wt.Pull(&git.PullOptions{
		Auth:  auth(r.signer),
		Force: true,
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}
	// TODO(gio): check `remote repository is empty`
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

func (r *repoIO) Atomic(op AtomicOp) error {
	r.l.Lock()
	defer r.l.Unlock()
	if err := r.pullWithoutLock(); err != nil {
		return err
	}
	if msg, err := op(r); err != nil {
		return err
	} else {
		return r.CommitAndPush(msg)
	}
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

func ReadYaml[T any](repo RepoFS, path string, o *T) error {
	r, err := repo.Reader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	if contents, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		return yaml.UnmarshalStrict(contents, o)
	}
}

func WriteYaml(repo RepoFS, path string, data any) error {
	if d, ok := data.(*Kustomization); ok {
		data = d
	}
	out, err := repo.Writer(path)
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

func ReadKustomization(repo RepoFS, path string) (*Kustomization, error) {
	ret := &Kustomization{}
	if err := ReadYaml(repo, path, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}
