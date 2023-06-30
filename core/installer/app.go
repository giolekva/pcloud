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

type Named interface {
	Nam() string
}

type App struct {
	Name       string
	Namespaces []string
	Templates  []*template.Template
	Schema     string
	Readme     *template.Template
}

type StoreApp struct {
	App
	Icon             string
	ShortDescription string
}

func (a App) Nam() string {
	return a.Name
}

func (a StoreApp) Nam() string {
	return a.Name
}

type AppRepository[A Named] interface {
	GetAll() ([]A, error)
	Find(name string) (*A, error)
}

type InMemoryAppRepository[A Named] struct {
	apps []A
}

func NewInMemoryAppRepository[A Named](apps []A) AppRepository[A] {
	return &InMemoryAppRepository[A]{
		apps,
	}
}

func (r InMemoryAppRepository[A]) Find(name string) (*A, error) {
	for _, a := range r.apps {
		if a.Nam() == name {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("Application not found: %s", name)
}

func (r InMemoryAppRepository[A]) GetAll() ([]A, error) {
	return r.apps, nil
}

func CreateAllApps() []App {
	tmpls, err := template.New("root").Funcs(template.FuncMap(sprig.FuncMap())).ParseFS(valuesTmpls, "values-tmpl/*")
	if err != nil {
		log.Fatal(err)
	}
	ret := []App{
		CreateAppIngressPrivate(valuesTmpls, tmpls),
		CreateCertificateIssuerPublic(valuesTmpls, tmpls),
		CreateCertificateIssuerPrivate(valuesTmpls, tmpls),
		CreateAppCoreAuth(valuesTmpls, tmpls),
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
	for _, a := range CreateStoreApps() {
		ret = append(ret, a.App)
	}
	return ret
}

func CreateStoreApps() []StoreApp {
	tmpls, err := template.New("root").Funcs(template.FuncMap(sprig.FuncMap())).ParseFS(valuesTmpls, "values-tmpl/*")
	if err != nil {
		log.Fatal(err)
	}
	return []StoreApp{
		CreateAppVaultwarden(valuesTmpls, tmpls),
		CreateAppMatrix(valuesTmpls, tmpls),
		CreateAppPihole(valuesTmpls, tmpls),
		CreateAppMaddy(valuesTmpls, tmpls),
		CreateAppQBittorrent(valuesTmpls, tmpls),
		CreateAppJellyfin(valuesTmpls, tmpls),
		CreateAppRpuppy(valuesTmpls, tmpls),
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
		[]string{"ingress-private"},
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
		[]string{},
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
		[]string{},
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
		[]string{"core-auth"},
		[]*template.Template{
			tmpls.Lookup("core-auth-storage.yaml"),
			tmpls.Lookup("core-auth.yaml"),
		},
		string(schema),
		tmpls.Lookup("core-auth.md"),
	}
}

func CreateAppVaultwarden(fs embed.FS, tmpls *template.Template) StoreApp {
	schema, err := fs.ReadFile("values-tmpl/vaultwarden.jsonschema")
	if err != nil {
		panic(err)
	}
	return StoreApp{
		App: App{
			"vaultwarden",
			[]string{"app-vaultwarden"},
			[]*template.Template{
				tmpls.Lookup("vaultwarden.yaml"),
			},
			string(schema),
			tmpls.Lookup("vaultwarden.md"),
		},
		Icon:             "arcticons:bitwarden",
		ShortDescription: "Open source implementation of Bitwarden password manager. Can be used with official client applications.",
	}
}

func CreateAppMatrix(fs embed.FS, tmpls *template.Template) StoreApp {
	schema, err := fs.ReadFile("values-tmpl/matrix.jsonschema")
	if err != nil {
		panic(err)
	}
	return StoreApp{
		App{
			"matrix",
			[]string{"app-matrix"},
			[]*template.Template{
				tmpls.Lookup("matrix-storage.yaml"),
				tmpls.Lookup("matrix.yaml"),
			},
			string(schema),
			nil,
		},
		"simple-icons:matrix",
		"An open network for secure, decentralised communication",
	}
}

func CreateAppPihole(fs embed.FS, tmpls *template.Template) StoreApp {
	schema, err := fs.ReadFile("values-tmpl/pihole.jsonschema")
	if err != nil {
		panic(err)
	}
	return StoreApp{
		App{
			"pihole",
			[]string{"app-pihole"},
			[]*template.Template{
				tmpls.Lookup("pihole.yaml"),
			},
			string(schema),
			tmpls.Lookup("pihole.md"),
		},
		"simple-icons:pihole",
		"Pi-hole is a Linux network-level advertisement and Internet tracker blocking application which acts as a DNS sinkhole and optionally a DHCP server, intended for use on a private network.",
	}
}

func CreateAppMaddy(fs embed.FS, tmpls *template.Template) StoreApp {
	schema, err := fs.ReadFile("values-tmpl/maddy.jsonschema")
	if err != nil {
		panic(err)
	}
	return StoreApp{
		App{
			"maddy",
			[]string{"app-maddy"},
			[]*template.Template{
				tmpls.Lookup("maddy.yaml"),
			},
			string(schema),
			nil,
		},
		"arcticons:huawei-email",
		"SMPT/IMAP server to communicate via email.",
	}
}

func CreateAppQBittorrent(fs embed.FS, tmpls *template.Template) StoreApp {
	schema, err := fs.ReadFile("values-tmpl/qbittorrent.jsonschema")
	if err != nil {
		panic(err)
	}
	return StoreApp{
		App{
			"qbittorrent",
			[]string{"app-qbittorrent"},
			[]*template.Template{
				tmpls.Lookup("qbittorrent.yaml"),
			},
			string(schema),
			tmpls.Lookup("qbittorrent.md"),
		},
		"arcticons:qbittorrent-remote",
		"qBittorrent is a cross-platform free and open-source BitTorrent client written in native C++. It relies on Boost, Qt 6 toolkit and the libtorrent-rasterbar library, with an optional search engine written in Python.",
	}
}

func CreateAppJellyfin(fs embed.FS, tmpls *template.Template) StoreApp {
	schema, err := fs.ReadFile("values-tmpl/jellyfin.jsonschema")
	if err != nil {
		panic(err)
	}
	return StoreApp{
		App{
			"jellyfin",
			[]string{"app-jellyfin"},
			[]*template.Template{
				tmpls.Lookup("jellyfin.yaml"),
			},
			string(schema),
			nil,
		},
		"arcticons:jellyfin",
		"Jellyfin is a free and open-source media server and suite of multimedia applications designed to organize, manage, and share digital media files to networked devices.",
	}
}

func CreateAppRpuppy(fs embed.FS, tmpls *template.Template) StoreApp {
	schema, err := fs.ReadFile("values-tmpl/rpuppy.jsonschema")
	if err != nil {
		panic(err)
	}
	return StoreApp{
		App{
			"rpuppy",
			[]string{"app-rpuppy"},
			[]*template.Template{
				tmpls.Lookup("rpuppy.yaml"),
			},
			string(schema),
			tmpls.Lookup("rpuppy.md"),
		},
		"ph:dog-thin",
		"Delights users with randomly generate puppy pictures. Can be configured to be reachable only from private network or publicly.",
	}
}

func CreateAppHeadscale(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/headscale.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"headscale",
		[]string{"app-headscale"},
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
		[]string{"tailscale-proxy"},
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
		[]string{"metallb-config"},
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
		[]string{"env-manager"},
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
		[]string{"app-welcome"},
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
		[]string{"ingress-public"},
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
		[]string{"cert-manager"},
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
		[]string{},
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
		[]string{},
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
		[]string{"csi-driver-smb"},
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
		[]string{"rr-controller"},
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
		[]string{"headscale-controller"},
		[]*template.Template{
			tmpls.Lookup("headscale-controller.yaml"),
		},
		string(schema),
		tmpls.Lookup("headscale-controller.md"),
	}
}
