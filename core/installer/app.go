package installer

import (
	"embed"
	"fmt"
	"log"
	"text/template"
)

//go:embed values-tmpl
var valuesTmpls embed.FS

type App struct {
	Name      string
	Templates []*template.Template
	Schema    string
	Readme    *template.Template
}

type AppRepository interface {
	GetAll() ([]App, error)
	Find(name string) (*App, error)
}

type InMemoryAppRepository struct {
	apps []App
}

func NewInMemoryAppRepository(apps []App) AppRepository {
	return &InMemoryAppRepository{
		apps,
	}
}

func (r InMemoryAppRepository) Find(name string) (*App, error) {
	for _, a := range r.apps {
		if a.Name == name {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("Application not found: %s", name)
}

func (r InMemoryAppRepository) GetAll() ([]App, error) {
	return r.apps, nil
}

func CreateAllApps() []App {
	tmpls, err := template.ParseFS(valuesTmpls, "values-tmpl/*")
	if err != nil {
		log.Fatal(err)
	}
	return []App{
		// CreateAppIngressPublic(tmpls),
		CreateAppIngressPrivate(valuesTmpls, tmpls),
		CreateAppCoreAuth(valuesTmpls, tmpls),
		CreateAppVaultwarden(valuesTmpls, tmpls),
		CreateAppMatrix(valuesTmpls, tmpls),
		CreateAppPihole(valuesTmpls, tmpls),
		CreateAppMaddy(valuesTmpls, tmpls),
		CreateAppQBittorrent(valuesTmpls, tmpls),
		CreateAppJellyfin(valuesTmpls, tmpls),
		CreateAppRpuppy(valuesTmpls, tmpls),
		CreateAppHeadscale(valuesTmpls, tmpls),
	}
}

func CreateAppIngressPublic(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/ingress-public.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"ingress-public",
		[]*template.Template{
			tmpls.Lookup("ingress-public.yaml"),
		},
		string(schema),
		nil,
	}
}

// TODO(gio): service account needs permission to create/update secret
func CreateAppIngressPrivate(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/ingress-private.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"ingress-private",
		[]*template.Template{
			// tmpls.Lookup("vpn-mesh-config.yaml"),
			tmpls.Lookup("ingress-private.yaml"),
			tmpls.Lookup("certificate-issuer.yaml"),
		},
		string(schema),
		tmpls.Lookup("ingress-private.md"),
	}
}

func CreateAppCoreAuth(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/core-auth.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"core-auth",
		[]*template.Template{
			tmpls.Lookup("core-auth-storage.yaml"),
			tmpls.Lookup("core-auth.yaml"),
		},
		string(schema),
		tmpls.Lookup("core-auth.md"),
	}
}

func CreateAppVaultwarden(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/vaultwarden.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"vaultwarden",
		[]*template.Template{
			tmpls.Lookup("vaultwarden.yaml"),
		},
		string(schema),
		tmpls.Lookup("vaultwarden.md"),
	}
}

func CreateAppMatrix(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/matrix.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"matrix",
		[]*template.Template{
			tmpls.Lookup("matrix-storage.yaml"),
			tmpls.Lookup("matrix.yaml"),
		},
		string(schema),
		nil,
	}
}

func CreateAppPihole(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/pihole.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"pihole",
		[]*template.Template{
			tmpls.Lookup("pihole.yaml"),
		},
		string(schema),
		tmpls.Lookup("pihole.md"),
	}
}

func CreateAppMaddy(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/maddy.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"maddy",
		[]*template.Template{
			tmpls.Lookup("maddy.yaml"),
		},
		string(schema),
		nil,
	}
}

func CreateAppQBittorrent(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/qbittorrent.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"qbittorrent",
		[]*template.Template{
			tmpls.Lookup("qbittorrent.yaml"),
		},
		string(schema),
		nil,
	}
}

func CreateAppJellyfin(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/jellyfin.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"jellyfin",
		[]*template.Template{
			tmpls.Lookup("jellyfin.yaml"),
		},
		string(schema),
		nil,
	}
}

func CreateAppRpuppy(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/rpuppy.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"rpuppy",
		[]*template.Template{
			tmpls.Lookup("rpuppy.yaml"),
		},
		string(schema),
		tmpls.Lookup("rpuppy.md"),
	}
}

func CreateAppHeadscale(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/headscale.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"headscale",
		[]*template.Template{
			tmpls.Lookup("headscale.yaml"),
		},
		string(schema),
		tmpls.Lookup("headscale.md"),
	}
}
