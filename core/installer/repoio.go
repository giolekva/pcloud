package installer

import (
	"io"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"golang.org/x/crypto/ssh"
)

type RepoIO interface {
	ReadKustomization(path string) (*Kustomization, error)
	WriteKustomization(path string, kust Kustomization) error
	CommitAndPush(message string) error
	Writer(path string) (io.WriteCloser, error)
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

func (r *repoIO) ReadKustomization(path string) (*Kustomization, error) {
	wt, err := r.repo.Worktree()
	if err != nil {
		return nil, err
	}
	inp, err := wt.Filesystem.Open(path)
	if err != nil {
		return nil, err
	}
	defer inp.Close()
	return ReadKustomization(inp)
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
