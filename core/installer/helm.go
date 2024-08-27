package installer

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/giolekva/pcloud/core/installer/soft"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
)

type ActionConfigFactory struct {
	kubeConfigPath string
}

func NewActionConfigFactory(kubeConfigPath string) ActionConfigFactory {
	return ActionConfigFactory{kubeConfigPath}
}

func (f ActionConfigFactory) New(namespace string) (*action.Configuration, error) {
	config := new(action.Configuration)
	if err := config.Init(
		kube.GetConfig(f.kubeConfigPath, "", namespace),
		namespace,
		"",
		func(fmtString string, args ...any) {
			fmt.Printf(fmtString, args...)
			fmt.Println()
		},
	); err != nil {
		return nil, err
	}
	return config, nil
}

type HelmFetcher interface {
	// TODO(gio): implement integrity check
	Pull(chart HelmChartGitRepo, rfs soft.RepoFS, root string) error
}

type RepoCloner interface {
	Clone(addr, ref string) (*git.Repository, error)
}

type cachingRepoCloner struct {
	cache map[string]*git.Repository
}

func NewCachingRepoCloner() RepoCloner {
	return &cachingRepoCloner{make(map[string]*git.Repository)}
}

func (rc *cachingRepoCloner) Clone(addr, ref string) (*git.Repository, error) {
	key := fmt.Sprintf("%s:%s", addr, ref)
	if ret, ok := rc.cache[key]; ok {
		return ret, nil
	}
	r, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:           addr,
		ReferenceName: plumbing.ReferenceName(ref),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return nil, err
	}
	// TODO(gio): enable
	// rc.cache[key] = r
	return r, nil
}

type gitHelmFetcher struct {
	rc RepoCloner
}

func NewGitHelmFetcher() *gitHelmFetcher {
	// TODO(gio): take cloner as an argument
	return &gitHelmFetcher{NewCachingRepoCloner()}
}

func (f *gitHelmFetcher) Pull(chart HelmChartGitRepo, rfs soft.RepoFS, root string) error {
	ref := fmt.Sprintf("refs/heads/%s", chart.Branch)
	r, err := f.rc.Clone(chart.Address, ref)
	if err != nil {
		return err
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	wtFS, err := wt.Filesystem.Chroot(chart.Path)
	if err != nil {
		return err
	}
	if err := rfs.RemoveDir(root); err != nil {
		return err
	}
	return util.Walk(wtFS, "/", func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		inp, err := wtFS.Open(path)
		if err != nil {
			return err
		}
		out, err := rfs.Writer(filepath.Join(root, path))
		if err != nil {
			return err
		}
		_, err = io.Copy(out, inp)
		return err
	})
}
