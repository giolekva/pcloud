package welcome

import (
	"embed"
	"fmt"
	"io/fs"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

//go:embed app-tmpl
var appTmpl embed.FS

var network = installer.Network{
	Name:               "Public",
	IngressClass:       fmt.Sprintf("%s-ingress-public", "dodo"),
	CertificateIssuer:  fmt.Sprintf("%s-public", "io"),
	Domain:             "example.com",
	AllocatePortAddr:   fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/allocate", "dodo"),
	ReservePortAddr:    fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/reserve", "dodo"),
	DeallocatePortAddr: fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/remove", "dodo"),
}

func TestAppTmplGolang1200(t *testing.T) {
	d, err := fs.Sub(appTmpl, "app-tmpl")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewAppTmplStoreFS(d)
	if err != nil {
		t.Fatal(err)
	}
	a, err := store.Find("golang-1.20.0")
	if err != nil {
		t.Fatal(err)
	}
	out := soft.NewBillyRepoFS(memfs.New())
	if err := a.Render(network, "testapp", out); err != nil {
		t.Fatal(err)
	}
}

func TestAppTmplGolang1220(t *testing.T) {
	d, err := fs.Sub(appTmpl, "app-tmpl")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewAppTmplStoreFS(d)
	if err != nil {
		t.Fatal(err)
	}
	a, err := store.Find("golang-1.22.0")
	if err != nil {
		t.Fatal(err)
	}
	out := soft.NewBillyRepoFS(memfs.New())
	if err := a.Render(network, "testapp", out); err != nil {
		t.Fatal(err)
	}
}

func TestAppTmplHugoLatest(t *testing.T) {
	d, err := fs.Sub(appTmpl, "app-tmpl")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewAppTmplStoreFS(d)
	if err != nil {
		t.Fatal(err)
	}
	a, err := store.Find("hugo-latest")
	if err != nil {
		t.Fatal(err)
	}
	out := soft.NewBillyRepoFS(memfs.New())
	if err := a.Render(network, "testapp", out); err != nil {
		t.Fatal(err)
	}
}
