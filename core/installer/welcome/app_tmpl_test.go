package welcome

import (
	"embed"
	"fmt"
	"io/fs"
	"testing"

	"github.com/giolekva/pcloud/core/installer"
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
	if _, err := a.Render("schema.json", network, "testapp"); err != nil {
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
	if _, err := a.Render("schema.json", network, "testapp"); err != nil {
		t.Fatal(err)
	}
}

func TestAppTmplPHP82(t *testing.T) {
	d, err := fs.Sub(appTmpl, "app-tmpl")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewAppTmplStoreFS(d)
	if err != nil {
		t.Fatal(err)
	}
	a, err := store.Find("php-8.2-apache")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Render("schema.json", network, "testapp"); err != nil {
		t.Fatal(err)
	}
}
