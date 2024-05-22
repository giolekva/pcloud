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
	Pull(chart HelmChartGitRepo, rfs soft.RepoFS, root string) error
}

type gitHelmFetcher struct{}

func NewGitHelmFetcher() *gitHelmFetcher {
	return &gitHelmFetcher{}
}

func (f *gitHelmFetcher) Pull(chart HelmChartGitRepo, rfs soft.RepoFS, root string) error {
	ref := fmt.Sprintf("refs/heads/%s", chart.Branch)
	r, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:           chart.Address,
		ReferenceName: plumbing.ReferenceName(ref),
		SingleBranch:  true,
		Depth:         1,
	})
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
