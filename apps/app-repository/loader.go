package apprepo

import (
	"io"
	"io/fs"
	"log"
	"strings"

	"golang.org/x/mod/semver"
)

type Loader interface {
	Load() ([]App, error)
}

type fsApp struct {
	name    string
	version string
	fs      fs.FS
	path    string
}

func (a *fsApp) Name() string {
	return a.name
}

func (a *fsApp) Version() string {
	return a.version
}

func (a *fsApp) Reader() (io.ReadCloser, error) {
	return a.fs.Open(a.path)
}

type fsLoader struct {
	fs fs.FS
}

func NewFSLoader(fs fs.FS) Loader {
	return &fsLoader{fs}
}

func (l *fsLoader) Load() ([]App, error) {
	entries, err := fs.ReadDir(l.fs, ".")
	if err != nil {
		return nil, err
	}
	apps := make([]App, 0)
	for _, e := range entries {
		log.Println(e.Name())
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".tar.gz") {
			items := strings.Split(strings.TrimSuffix(e.Name(), ".tar.gz"), "-")
			if len(items) <= 1 {
				continue
			}
			version := items[len(items)-1]
			if semver.IsValid(version) || semver.IsValid("v"+version) {
				name := strings.Join(items[:len(items)-1], "-")
				apps = append(apps, &fsApp{name, strings.TrimPrefix(version, "v"), l.fs, e.Name()})
			}
		}
	}
	return apps, nil
}
