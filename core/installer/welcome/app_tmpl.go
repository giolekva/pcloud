package welcome

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
	"text/template"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

const tmplSuffix = ".gotmpl"

type AppTmplStore interface {
	Types() []string
	Find(appType string) (AppTmpl, error)
}

type appTmplStoreFS struct {
	tmpls map[string]AppTmpl
}

func NewAppTmplStoreFS(fsys fs.FS) (AppTmplStore, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	apps := map[string]AppTmpl{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		app, err := NewAppTmplFS(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		apps[e.Name()] = app
	}
	return &appTmplStoreFS{apps}, nil
}

func (s *appTmplStoreFS) Types() []string {
	var ret []string
	for t := range s.tmpls {
		ret = append(ret, t)
	}
	return ret
}

func (s *appTmplStoreFS) Find(appType string) (AppTmpl, error) {
	if app, ok := s.tmpls[appType]; ok {
		return app, nil
	} else {
		return nil, fmt.Errorf("not found")
	}
}

type AppTmpl interface {
	Render(network installer.Network, subdomain string, out soft.RepoFS) error
}

type appTmplFS struct {
	files map[string][]byte
	tmpls map[string]*template.Template
}

func NewAppTmplFS(fsys fs.FS, root string) (AppTmpl, error) {
	files := map[string][]byte{}
	tmpls := map[string]*template.Template{}
	if err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		contents, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		p, _ := strings.CutPrefix(path, root)
		if !strings.HasSuffix(p, tmplSuffix) {
			files[p] = contents
			return nil
		}
		tmpl, err := template.New(path).Parse(string(contents))
		if err != nil {
			return err
		}
		np, _ := strings.CutSuffix(p, tmplSuffix)
		tmpls[np] = tmpl
		return nil
	}); err != nil {
		return nil, err
	}
	return &appTmplFS{files, tmpls}, nil
}

func (a *appTmplFS) Render(network installer.Network, subdomain string, out soft.RepoFS) error {
	for path, tmpl := range a.tmpls {
		f, err := out.Writer(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := tmpl.Execute(f, map[string]any{
			"Network":   network,
			"Subdomain": subdomain,
		}); err != nil {
			return err
		}
	}
	for path, contents := range a.files {
		f, err := out.Writer(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.WriteString(f, string(contents)); err != nil {
			return err
		}
	}
	return nil
}
