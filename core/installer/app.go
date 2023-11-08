package installer

import (
	"archive/tar"
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	htemplate "html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-git/go-billy/v5"
	"sigs.k8s.io/yaml"
)

//go:embed values-tmpl
var valuesTmpls embed.FS

type Named interface {
	Nam() string
}

type appConfig struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Namespaces  []string       `json:"namespaces"`
	Icon        htemplate.HTML `json:"icon"`
}

type App struct {
	Name       string
	Namespaces []string
	Templates  []*template.Template
	Schema     string
	Readme     *template.Template
}

func (a App) ConfigSchema() map[string]any {
	ret := make(map[string]any)
	if err := json.NewDecoder(strings.NewReader(a.Schema)).Decode(&ret); err != nil {
		panic(err) // TODO(giolekva): prevalidate
	}
	return ret
}

type StoreApp struct {
	App
	Icon             htemplate.HTML
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

func NewInMemoryAppRepository[A Named](apps []A) InMemoryAppRepository[A] {
	return InMemoryAppRepository[A]{
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
		CreateMetallbIPAddressPool(valuesTmpls, tmpls),
		CreateEnvManager(valuesTmpls, tmpls),
		CreateWelcome(valuesTmpls, tmpls),
		CreateAppManager(valuesTmpls, tmpls),
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
		CreateAppSoftServe(valuesTmpls, tmpls),
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
		Icon:             `<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 48 48"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" d="M35.38 25.63V9.37H24v28.87a34.93 34.93 0 0 0 5.41-3.48q6-4.66 6-9.14Zm4.87-19.5v19.5A11.58 11.58 0 0 1 39.4 30a16.22 16.22 0 0 1-2.11 3.81a23.52 23.52 0 0 1-3 3.24a34.87 34.87 0 0 1-3.22 2.62c-1 .69-2 1.35-3.07 2s-1.82 1-2.27 1.26l-1.08.51a1.53 1.53 0 0 1-1.32 0l-1.08-.51c-.45-.22-1.21-.64-2.27-1.26s-2.09-1.27-3.07-2A34.87 34.87 0 0 1 13.7 37a23.52 23.52 0 0 1-3-3.24A16.22 16.22 0 0 1 8.6 30a11.58 11.58 0 0 1-.85-4.32V6.13A1.64 1.64 0 0 1 9.38 4.5h29.24a1.64 1.64 0 0 1 1.63 1.63Z"/></svg>`,
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
		`<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 24 24"><path fill="currentColor" d="M.632.55v22.9H2.28V24H0V0h2.28v.55zm7.043 7.26v1.157h.033a3.312 3.312 0 0 1 1.117-1.024c.433-.245.936-.365 1.5-.365c.54 0 1.033.107 1.481.314c.448.208.785.582 1.02 1.108c.254-.374.6-.706 1.034-.992c.434-.287.95-.43 1.546-.43c.453 0 .872.056 1.26.167c.388.11.716.286.993.53c.276.245.489.559.646.951c.152.392.23.863.23 1.417v5.728h-2.349V11.52c0-.286-.01-.559-.032-.812a1.755 1.755 0 0 0-.18-.66a1.106 1.106 0 0 0-.438-.448c-.194-.11-.457-.166-.785-.166c-.332 0-.6.064-.803.189a1.38 1.38 0 0 0-.48.499a1.946 1.946 0 0 0-.231.696a5.56 5.56 0 0 0-.06.785v4.768h-2.35v-4.8c0-.254-.004-.503-.018-.752a2.074 2.074 0 0 0-.143-.688a1.052 1.052 0 0 0-.415-.503c-.194-.125-.476-.19-.854-.19c-.111 0-.259.024-.439.074c-.18.051-.36.143-.53.282a1.637 1.637 0 0 0-.439.595c-.12.259-.18.6-.18 1.02v4.966H5.46V7.81zm15.693 15.64V.55H21.72V0H24v24h-2.28v-.55z"/></svg>`,
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
		// "simple-icons:pihole",
		`<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 24 24"><path fill="currentColor" d="M4.344 0c.238 4.792 3.256 7.056 6.252 7.376c.165-1.692-4.319-5.6-4.319-5.6c-.008-.011.009-.025.019-.014c0 0 4.648 4.01 5.423 5.645c2.762-.15 5.196-1.947 5-4.912c0 0-4.12-.613-5 4.618C11.48 2.753 8.993 0 4.344 0zM12 7.682v.002a3.68 3.68 0 0 0-2.591 1.077L4.94 13.227a3.683 3.683 0 0 0-.86 1.356a3.31 3.31 0 0 0-.237 1.255A3.681 3.681 0 0 0 4.92 18.45l4.464 4.466a3.69 3.69 0 0 0 2.251 1.06l.002.001c.093.01.187.015.28.017l-.1-.008c.06.003.117.009.177.009l-.077-.001L12 24l-.004-.005a3.68 3.68 0 0 0 2.61-1.077l4.469-4.465a3.683 3.683 0 0 0 1.006-1.888l.012-.063a3.682 3.682 0 0 0 .057-.541l.003-.061c0-.017.003-.05.004-.06h-.002a3.683 3.683 0 0 0-1.077-2.607l-4.466-4.468a3.694 3.694 0 0 0-1.564-.927l-.07-.02a3.43 3.43 0 0 0-.946-.133L12 7.682zm3.165 3.357c.023 1.748-1.33 3.078-1.33 4.806c.164 2.227 1.733 3.207 3.266 3.146c-.035.003-.068.007-.104.009c-1.847.135-3.209-1.326-5.002-1.326c-2.23.164-3.21 1.736-3.147 3.27l-.008-.104c-.133-1.847 1.328-3.21 1.328-5.002c-.173-2.32-1.867-3.284-3.46-3.132c.1-.011.203-.021.31-.027c1.847-.133 3.209 1.328 5.002 1.328c2.082-.155 3.074-1.536 3.145-2.968zM4.344 0c.238 4.792 3.256 7.056 6.252 7.376c.165-1.692-4.319-5.6-4.319-5.6c-.008-.011.009-.025.019-.014c0 0 4.648 4.01 5.423 5.645c2.762-.15 5.196-1.947 5-4.912c0 0-4.12-.613-5 4.618C11.48 2.753 8.993 0 4.344 0zM12 7.682v.002a3.68 3.68 0 0 0-2.591 1.077L4.94 13.227a3.683 3.683 0 0 0-.86 1.356a3.31 3.31 0 0 0-.237 1.255A3.681 3.681 0 0 0 4.92 18.45l4.464 4.466a3.69 3.69 0 0 0 2.251 1.06l.002.001c.093.01.187.015.28.017l-.1-.008c.06.003.117.009.177.009l-.077-.001L12 24l-.004-.005a3.68 3.68 0 0 0 2.61-1.077l4.469-4.465a3.683 3.683 0 0 0 1.006-1.888l.012-.063a3.682 3.682 0 0 0 .057-.541l.003-.061c0-.017.003-.05.004-.06h-.002a3.683 3.683 0 0 0-1.077-2.607l-4.466-4.468a3.694 3.694 0 0 0-1.564-.927l-.07-.02a3.43 3.43 0 0 0-.946-.133L12 7.682zm3.165 3.357c.023 1.748-1.33 3.078-1.33 4.806c.164 2.227 1.733 3.207 3.266 3.146c-.035.003-.068.007-.104.009c-1.847.135-3.209-1.326-5.002-1.326c-2.23.164-3.21 1.736-3.147 3.27l-.008-.104c-.133-1.847 1.328-3.21 1.328-5.002c-.173-2.32-1.867-3.284-3.46-3.132c.1-.011.203-.021.31-.027c1.847-.133 3.209 1.328 5.002 1.328c2.082-.155 3.074-1.536 3.145-2.968z"/></svg>`,
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
		`<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 48 48"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" d="M9.5 13c13.687 13.574 14.825 13.09 29 0"/><rect width="37" height="31" x="5.5" y="8.5" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" rx="2"/></svg>`,
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
		`<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 48 48"><circle cx="24" cy="24" r="21.5" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"/><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" d="M26.651 22.364a5.034 5.034 0 0 1 5.035-5.035h0a5.034 5.034 0 0 1 5.034 5.035v3.272a5.034 5.034 0 0 1-5.034 5.035h0a5.034 5.034 0 0 1-5.035-5.035m0 5.035V10.533m-5.302 15.103a5.034 5.034 0 0 1-5.035 5.035h0a5.034 5.034 0 0 1-5.034-5.035v-3.272a5.034 5.034 0 0 1 5.034-5.035h0a5.034 5.034 0 0 1 5.035 5.035m0-5.035v20.138"/></svg>`,
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
		`<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 48 48"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" d="M24 20c-1.62 0-6.85 9.48-6.06 11.08s11.33 1.59 12.12 0S25.63 20 24 20Z"/><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" d="M24 5.5c-4.89 0-20.66 28.58-18.25 33.4s34.13 4.77 36.51 0S28.9 5.5 24 5.5Zm12 29.21c-1.56 3.13-22.35 3.17-23.93 0S20.8 12.83 24 12.83s13.52 18.76 12 21.88Z"/></svg>`,
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
		`<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 256 256"><path fill="currentColor" d="M100 140a8 8 0 1 1-8-8a8 8 0 0 1 8 8Zm64 8a8 8 0 1 0-8-8a8 8 0 0 0 8 8Zm64.94-9.11a12.12 12.12 0 0 1-5 1.11a11.83 11.83 0 0 1-9.35-4.62l-2.59-3.29V184a36 36 0 0 1-36 36H80a36 36 0 0 1-36-36v-51.91l-2.53 3.27A11.88 11.88 0 0 1 32.1 140a12.08 12.08 0 0 1-5-1.11a11.82 11.82 0 0 1-6.84-13.14l16.42-88a12 12 0 0 1 14.7-9.43h.16L104.58 44h46.84l53.08-15.6h.16a12 12 0 0 1 14.7 9.43l16.42 88a11.81 11.81 0 0 1-6.84 13.06ZM97.25 50.18L49.34 36.1a4.18 4.18 0 0 0-.92-.1a4 4 0 0 0-3.92 3.26l-16.42 88a4 4 0 0 0 7.08 3.22ZM204 121.75L150 52h-44l-54 69.75V184a28 28 0 0 0 28 28h44v-18.34l-14.83-14.83a4 4 0 0 1 5.66-5.66L128 186.34l13.17-13.17a4 4 0 0 1 5.66 5.66L132 193.66V212h44a28 28 0 0 0 28-28Zm23.92 5.48l-16.42-88a4 4 0 0 0-4.84-3.16l-47.91 14.11l62.11 80.28a4 4 0 0 0 7.06-3.23Z"/></svg>`,
		"Delights users with randomly generate puppy pictures. Can be configured to be reachable only from private network or publicly.",
	}
}

func CreateAppSoftServe(fs embed.FS, tmpls *template.Template) StoreApp {
	schema, err := fs.ReadFile("values-tmpl/soft-serve.jsonschema")
	if err != nil {
		panic(err)
	}
	return StoreApp{
		App{
			"soft-serve",
			[]string{"app-soft-serve"},
			[]*template.Template{
				tmpls.Lookup("soft-serve.yaml"),
			},
			string(schema),
			tmpls.Lookup("soft-serve.md"),
		},
		`<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 48 48"><g fill="none" stroke="currentColor" stroke-linecap="round" stroke-width="4"><path stroke-linejoin="round" d="M15.34 22.5L21 37l3 6l3-6l5.66-14.5"/><path d="M19 32h10"/><path stroke-linejoin="round" d="M24 3c-6 0-8 6-8 6s-6 2-6 7s5 7 5 7s3.5-2 9-2s9 2 9 2s5-2 5-7s-6-7-6-7s-2-6-8-6Z"/></g></svg>`,
		"A tasty, self-hostable Git server for the command line. 🍦",
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

func CreateMetallbIPAddressPool(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/metallb-ipaddresspool.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"metallb-ipaddresspool",
		[]string{"metallb-ipaddresspool"},
		[]*template.Template{
			tmpls.Lookup("metallb-ipaddresspool.yaml"),
		},
		string(schema),
		tmpls.Lookup("metallb-ipaddresspool.md"),
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

func CreateAppManager(fs embed.FS, tmpls *template.Template) App {
	schema, err := fs.ReadFile("values-tmpl/appmanager.jsonschema")
	if err != nil {
		panic(err)
	}
	return App{
		"app-manager",
		[]string{"core-appmanager"},
		[]*template.Template{
			tmpls.Lookup("appmanager.yaml"),
		},
		string(schema),
		tmpls.Lookup("appmanager.md"),
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

type httpAppRepository struct {
	apps []StoreApp
}

type appVersion struct {
	Version string   `json:"version"`
	Urls    []string `json:"urls"`
}

type allAppsResp struct {
	ApiVersion string                  `json:"apiVersion"`
	Entries    map[string][]appVersion `json:"entries"`
}

func FetchAppsFromHTTPRepository(addr string, fs billy.Filesystem) error {
	resp, err := http.Get(addr)
	if err != nil {
		return err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var apps allAppsResp
	if err := yaml.Unmarshal(b, &apps); err != nil {
		return err
	}
	for name, conf := range apps.Entries {
		for _, version := range conf {
			resp, err := http.Get(version.Urls[0])
			if err != nil {
				return err
			}
			nameVersion := fmt.Sprintf("%s-%s", name, version.Version)
			if err := fs.MkdirAll(nameVersion, 0700); err != nil {
				return err
			}
			sub, err := fs.Chroot(nameVersion)
			if err != nil {
				return err
			}
			if err := extractApp(resp.Body, sub); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractApp(archive io.Reader, fs billy.Filesystem) error {
	uncompressed, err := gzip.NewReader(archive)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(uncompressed)
	for true {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(header.Name, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			out, err := fs.Create(header.Name)
			if err != nil {
				return err
			}
			defer out.Close()
			if _, err := io.Copy(out, tarReader); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Uknown type: %s", header.Name)
		}
	}
	return nil
}

type fsAppRepository struct {
	InMemoryAppRepository[StoreApp]
	fs billy.Filesystem
}

func NewFSAppRepository(fs billy.Filesystem) (AppRepository[StoreApp], error) {
	all, err := fs.ReadDir(".")
	if err != nil {
		return nil, err
	}
	apps := make([]StoreApp, 0)
	for _, e := range all {
		if !e.IsDir() {
			continue
		}
		appFS, err := fs.Chroot(e.Name())
		if err != nil {
			return nil, err
		}
		app, err := loadApp(appFS)
		if err != nil {
			log.Printf("Ignoring directory %s: %s", e.Name(), err)
			continue
		}
		apps = append(apps, app)
	}
	return &fsAppRepository{
		NewInMemoryAppRepository[StoreApp](apps),
		fs,
	}, nil
}

func loadApp(fs billy.Filesystem) (StoreApp, error) {
	cfg, err := fs.Open("Chart.yaml")
	if err != nil {
		return StoreApp{}, err
	}
	defer cfg.Close()
	b, err := io.ReadAll(cfg)
	if err != nil {
		return StoreApp{}, err
	}
	var appCfg appConfig
	if err := yaml.Unmarshal(b, &appCfg); err != nil {
		return StoreApp{}, err
	}
	rb, err := fs.Open("README.md")
	if err != nil {
		return StoreApp{}, err
	}
	defer rb.Close()
	readme, err := io.ReadAll(rb)
	if err != nil {
		return StoreApp{}, err
	}
	readmeTmpl, err := template.New("README.md").Parse(string(readme))
	if err != nil {
		return StoreApp{}, err
	}
	sb, err := fs.Open("schema.json")
	if err != nil {
		return StoreApp{}, err
	}
	defer sb.Close()
	schema, err := io.ReadAll(sb)
	if err != nil {
		return StoreApp{}, err
	}
	tFiles, err := fs.ReadDir("templates")
	if err != nil {
		return StoreApp{}, err
	}
	tmpls := make([]*template.Template, 0)
	for _, t := range tFiles {
		if !strings.HasSuffix(t.Name(), ".yaml") {
			continue
		}
		inp, err := fs.Open(fs.Join("templates", t.Name()))
		if err != nil {
			return StoreApp{}, err
		}
		b, err := io.ReadAll(inp)
		if err != nil {
			return StoreApp{}, err
		}
		tmpl, err := template.New(t.Name()).Parse(string(b))
		if err != nil {
			return StoreApp{}, err
		}
		tmpls = append(tmpls, tmpl)
	}
	return StoreApp{
		App: App{
			Name:       appCfg.Name,
			Readme:     readmeTmpl,
			Schema:     string(schema),
			Namespaces: appCfg.Namespaces,
			Templates:  tmpls,
		},
		ShortDescription: appCfg.Description,
		Icon:             appCfg.Icon,
	}, nil
}
