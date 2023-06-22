package installer

import (
	"embed"
	"fmt"
	"log"
	"text/template"

	"github.com/Masterminds/sprig/v3"
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
	tmpls, err := template.New("root").Funcs(template.FuncMap(sprig.FuncMap())).ParseFS(valuesTmpls, "values-tmpl/*")
	if err != nil {
		log.Fatal(err)
	}
	return []App{
		CreateAppIngressPrivate(valuesTmpls, tmpls),
		CreateCertificateIssuerPublic(valuesTmpls, tmpls),
		CreateCertificateIssuerPrivate(valuesTmpls, tmpls),
		CreateAppCoreAuth(valuesTmpls, tmpls),
		CreateAppVaultwarden(valuesTmpls, tmpls),
		CreateAppMatrix(valuesTmpls, tmpls),
		CreateAppPihole(valuesTmpls, tmpls),
		CreateAppMaddy(valuesTmpls, tmpls),
		CreateAppQBittorrent(valuesTmpls, tmpls),
		CreateAppJellyfin(valuesTmpls, tmpls),
		CreateAppRpuppy(valuesTmpls, tmpls),
		CreateAppHeadscale(valuesTmpls, tmpls),
		CreateAppTailscaleProxy(valuesTmpls, tmpls),
		CreateMetallbConfigEnv(valuesTmpls, tmpls),
		CreateEnvManager(valuesTmpls, tmpls),
		CreateWelcome(valuesTmpls, tmpls),
		CreateIngressPublic(valuesTmpls, tmpls),
		CreateCertManager(valuesTmpls, tmpls),
		CreateCertManagerWebhookGandi(valuesTmpls, tmpls),
		CreateCertManagerWebhookGandiRole(valuesTmpls, tmpls),
		CreateCSIDriverSMB(valuesTmpls, tmpls),
		CreateResourceRendererController(valuesTmpls, tmpls),
		CreateHeadscaleController(valuesTmpls, tmpls),
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
			tmpls.Lookup("ingress-private.yaml"),
		},
		string(schema),
		tmpls.Lookup("ingress-private.md"),
	}
}

func CreateCertificateIssuerPrivate(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/certificate-issuer-private.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"certificate-issuer-private",
		[]*template.Template{
			tmpls.Lookup("certificate-issuer-private.yaml"),
		},
		string(schema),
		tmpls.Lookup("certificate-issuer-private.md"),
	}
}

func CreateCertificateIssuerPublic(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/certificate-issuer-public.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"certificate-issuer-public",
		[]*template.Template{
			tmpls.Lookup("certificate-issuer-public.yaml"),
		},
		string(schema),
		tmpls.Lookup("certificate-issuer-public.md"),
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
		tmpls.Lookup("qbittorrent.md"),
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

func CreateAppTailscaleProxy(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/tailscale-proxy.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"tailscale-proxy",
		[]*template.Template{
			tmpls.Lookup("tailscale-proxy.yaml"),
		},
		string(schema),
		tmpls.Lookup("tailscale-proxy.md"),
	}
}

func CreateMetallbConfigEnv(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/metallb-config-env.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"metallb-config-env",
		[]*template.Template{
			tmpls.Lookup("metallb-config-env.yaml"),
		},
		string(schema),
		tmpls.Lookup("metallb-config-env.md"),
	}
}

func CreateEnvManager(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/env-manager.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"env-manager",
		[]*template.Template{
			tmpls.Lookup("env-manager.yaml"),
		},
		string(schema),
		tmpls.Lookup("env-manager.md"),
	}
}

func CreateWelcome(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/welcome.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"welcome",
		[]*template.Template{
			tmpls.Lookup("welcome.yaml"),
		},
		string(schema),
		tmpls.Lookup("welcome.md"),
	}
}

func CreateIngressPublic(fs embed.FS, tmpls *template.Template) App {
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
		tmpls.Lookup("ingress-public.md"),
	}
}

func CreateCertManager(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/cert-manager.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"cert-manager",
		[]*template.Template{
			tmpls.Lookup("cert-manager.yaml"),
		},
		string(schema),
		tmpls.Lookup("cert-manager.md"),
	}
}

func CreateCertManagerWebhookGandi(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/cert-manager-webhook-gandi.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"cert-manager-webhook-gandi",
		[]*template.Template{
			tmpls.Lookup("cert-manager-webhook-gandi.yaml"),
		},
		string(schema),
		tmpls.Lookup("cert-manager-webhook-gandi.md"),
	}
}

func CreateCertManagerWebhookGandiRole(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/cert-manager-webhook-gandi-role.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"cert-manager-webhook-gandi-role",
		[]*template.Template{
			tmpls.Lookup("cert-manager-webhook-gandi-role.yaml"),
		},
		string(schema),
		tmpls.Lookup("cert-manager-webhook-gandi-role.md"),
	}
}

func CreateCSIDriverSMB(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/csi-driver-smb.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"csi-driver-smb",
		[]*template.Template{
			tmpls.Lookup("csi-driver-smb.yaml"),
		},
		string(schema),
		tmpls.Lookup("csi-driver-smb.md"),
	}
}

func CreateResourceRendererController(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/resource-renderer-controller.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"resource-renderer-controller",
		[]*template.Template{
			tmpls.Lookup("resource-renderer-controller.yaml"),
		},
		string(schema),
		tmpls.Lookup("resource-renderer-controller.md"),
	}
}

func CreateHeadscaleController(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/headscale-controller.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"headscale-controller",
		[]*template.Template{
			tmpls.Lookup("headscale-controller.yaml"),
		},
		string(schema),
		tmpls.Lookup("headscale-controller.md"),
	}
}
