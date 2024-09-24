package welcome

import (
	"bytes"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"text/template"

	"github.com/giolekva/pcloud/core/installer"
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
	sort.Slice(ret, func(i, j int) bool {
		a := strings.SplitN(ret[i], ":", 2)
		b := strings.SplitN(ret[j], ":", 2)
		langCmp := strings.Compare(a[0], b[0])
		if langCmp != 0 {
			return langCmp < 0
		}
		// TODO(gio): compare semver?
		return strings.Compare(a[1], b[1]) > 0
	})
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
	Render(schemaAddr string, network installer.Network, subdomain string) (map[string][]byte, error)
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

func (a *appTmplFS) Render(schemaAddr string, network installer.Network, subdomain string) (map[string][]byte, error) {
	ret := map[string][]byte{}
	for path, contents := range a.files {
		ret[path] = contents
	}
	for path, tmpl := range a.tmpls {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, map[string]any{
			"SchemaAddr": schemaAddr,
			"Network":    network,
			"Subdomain":  subdomain,
		}); err != nil {
			return nil, err
		}
		ret[path] = buf.Bytes()
	}
	return ret, nil
}
