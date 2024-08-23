package soft

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	pio "github.com/giolekva/pcloud/core/installer/io"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"
)

type RepoFS interface {
	Reader(path string) (io.ReadCloser, error)
	Writer(path string) (io.WriteCloser, error)
	CreateDir(path string) error
	RemoveAll(path string) error
	ListDir(path string) ([]os.FileInfo, error)
}

type DoFn func(r RepoFS) (string, error)

type doOptions struct {
	NoPull   bool
	NoCommit bool
	Force    bool
	ToBranch string
	NoLock   bool
}

type DoOption func(*doOptions)

func WithNoLock() DoOption {
	return func(o *doOptions) {
		o.NoLock = true
	}
}

func WithNoPull() DoOption {
	return func(o *doOptions) {
		o.NoPull = true
	}
}

func WithNoCommit() DoOption {
	return func(o *doOptions) {
		o.NoCommit = true
	}
}

func WithForce() DoOption {
	return func(o *doOptions) {
		o.Force = true
	}
}

func WithCommitToBranch(branch string) DoOption {
	return func(o *doOptions) {
		o.ToBranch = branch
	}
}

type pushOptions struct {
	ToBranch string
	Force    bool
}

type PushOption func(*pushOptions)

func WithToBranch(branch string) PushOption {
	return func(o *pushOptions) {
		o.ToBranch = branch
	}
}

func PushWithForce() PushOption {
	return func(o *pushOptions) {
		o.Force = true
	}
}

type RepoIO interface {
	RepoFS
	FullAddress() string
	Pull() error
	CommitAndPush(message string, opts ...PushOption) (string, error)
	Do(op DoFn, opts ...DoOption) (string, error)
}

type repoFS struct {
	fs billy.Filesystem
}

func NewBillyRepoFS(fs billy.Filesystem) RepoFS {
	return &repoFS{fs}
}

func (r *repoFS) Reader(path string) (io.ReadCloser, error) {
	return r.fs.Open(path)
}

func (r *repoFS) Writer(path string) (io.WriteCloser, error) {
	if err := r.CreateDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	return r.fs.Create(path)
}

func (r *repoFS) CreateDir(path string) error {
	return r.fs.MkdirAll(path, fs.ModePerm)
}

func (r *repoFS) RemoveAll(path string) error {
	if err := util.RemoveAll(r.fs, path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

func (r *repoFS) ListDir(path string) ([]os.FileInfo, error) {
	return r.fs.ReadDir(path)
}

type repoIO struct {
	*repoFS
	repo   *Repository
	signer ssh.Signer
	l      sync.Locker
}

func NewRepoIO(repo *Repository, signer ssh.Signer) (RepoIO, error) {
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
		Auth:     auth(r.signer),
		Force:    true,
		Progress: os.Stdout,
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}
	// TODO(gio): check `remote repository is empty`
	fmt.Printf("-- GIT PULL: %s\n", err.Error())
	return nil
}

func (r *repoIO) CommitAndPush(message string, opts ...PushOption) (string, error) {
	var o pushOptions
	for _, i := range opts {
		i(&o)
	}
	wt, err := r.repo.Worktree()
	if err != nil {
		return "", err
	}
	if err := wt.AddGlob("*"); err != nil {
		return "", err
	}
	st, err := wt.Status()
	if err != nil {
		return "", err
	}
	if len(st) == 0 {
		return "", nil // TODO(gio): maybe return ErrorNothingToCommit
	}
	fmt.Printf("@@@ %+v\n", st)
	hash, err := wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name: "pcloud-installer",
			When: time.Now(),
		},
	})
	if err != nil {
		return "", err
	}
	gopts := &git.PushOptions{
		RemoteName: "origin",
		Auth:       auth(r.signer),
	}
	if o.ToBranch != "" {
		gopts.RefSpecs = []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:refs/heads/%s", r.repo.Ref, o.ToBranch))}
	}
	if o.Force {
		gopts.Force = true
	}
	fmt.Println(3333)
	return hash.String(), r.repo.Push(gopts)
}

func (r *repoIO) Do(op DoFn, opts ...DoOption) (string, error) {
	o := &doOptions{}
	for _, i := range opts {
		i(o)
	}
	if o.NoLock {
		r.l.Lock()
		defer r.l.Unlock()
	}
	if !o.NoPull {
		if err := r.pullWithoutLock(); err != nil {
			return "", err
		}
	}
	msg, err := op(r)
	if err != nil {
		return "", err
	}
	if o.NoCommit {
		return "", nil
	}
	popts := []PushOption{}
	if o.Force {
		popts = append(popts, PushWithForce())
	}
	if o.ToBranch != "" {
		popts = append(popts, WithToBranch(o.ToBranch))
	}
	fmt.Println(2222)
	return r.CommitAndPush(msg, popts...)
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
	if d, ok := data.(*pio.Kustomization); ok {
		data = d
	}
	out, err := repo.Writer(path)
	if err != nil {
		return err
	}
	defer out.Close()
	serialized, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := out.Write(serialized); err != nil {
		return err
	}
	return nil
}

func ReadJson[T any](repo RepoFS, path string, o *T) error {
	r, err := repo.Reader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	return json.NewDecoder(r).Decode(o)
}

func WriteJson(repo RepoFS, path string, data any) error {
	if d, ok := data.(*pio.Kustomization); ok {
		data = d
	}
	w, err := repo.Writer(path)
	if err != nil {
		return err
	}
	e := json.NewEncoder(w)
	e.SetIndent("", "\t")
	return e.Encode(data)
}

func ReadKustomization(repo RepoFS, path string) (*pio.Kustomization, error) {
	ret := &pio.Kustomization{}
	if err := ReadYaml(repo, path, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func ReadFile(repo RepoFS, path string) ([]byte, error) {
	r, err := repo.Reader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func WriteFile(repo RepoFS, path, contents string) error {
	w, err := repo.Writer(path)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.WriteString(w, contents)
	return err
}
