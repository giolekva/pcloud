package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	template "html/template"
	"io"
	"log"
	"net/http"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueyaml "cuelang.org/go/encoding/yaml"
	"github.com/go-git/go-billy/v5"
	"sigs.k8s.io/yaml"
)

//go:embed values-tmpl
var valuesTmpls embed.FS

var storeAppConfigs = []string{
	"values-tmpl/jellyfin.cue",
	// "values-tmpl/maddy.cue",
	"values-tmpl/matrix.cue",
	"values-tmpl/penpot.cue",
	"values-tmpl/pihole.cue",
	"values-tmpl/qbittorrent.cue",
	"values-tmpl/rpuppy.cue",
	"values-tmpl/soft-serve.cue",
	"values-tmpl/vaultwarden.cue",
	"values-tmpl/url-shortener.cue",
}

var infraAppConfigs = []string{
	"values-tmpl/appmanager.cue",
	"values-tmpl/cert-manager.cue",
	"values-tmpl/certificate-issuer-private.cue",
	"values-tmpl/certificate-issuer-public.cue",
	"values-tmpl/config-repo.cue",
	"values-tmpl/core-auth.cue",
	"values-tmpl/csi-driver-smb.cue",
	"values-tmpl/dns-zone-manager.cue",
	"values-tmpl/env-manager.cue",
	"values-tmpl/fluxcd-reconciler.cue",
	"values-tmpl/headscale-controller.cue",
	"values-tmpl/headscale-user.cue",
	"values-tmpl/headscale.cue",
	"values-tmpl/ingress-public.cue",
	"values-tmpl/metallb-ipaddresspool.cue",
	"values-tmpl/private-network.cue",
	"values-tmpl/resource-renderer-controller.cue",
	"values-tmpl/welcome.cue",
}

const cueBaseConfigImports = `
import (
    "list"
)
`

// TODO(gio): import
const cueBaseConfig = `
name: string | *""
description: string | *""
readme: string | *""
icon: string | *""
namespace: string | *""

#Network: {
	name: string
	ingressClass: string
	certificateIssuer: string | *""
	domain: string
}

#Image: {
	registry: string | *"docker.io"
	repository: string
	name: string
	tag: string
	pullPolicy: string | *"IfNotPresent"
	imageName: "\(repository)/\(name)"
	fullName: "\(registry)/\(imageName)"
	fullNameWithTag: "\(fullName):\(tag)"
}

#Chart: {
	chart: string
	sourceRef: #SourceRef
}

#SourceRef: {
	kind: "GitRepository" | "HelmRepository"
	name: string
	namespace: string // TODO(gio): default global.id
}

#Global: {
	id: string | *""
	pcloudEnvName: string | *""
	domain: string | *""
    privateDomain: string | *""
	namespacePrefix: string | *""
	...
}

#Release: {
	namespace: string
}

global: #Global
release: #Release

_ingressPrivate: "\(global.id)-ingress-private"
_ingressPublic: "\(global.pcloudEnvName)-ingress-public"
_issuerPrivate: "\(global.id)-private"
_issuerPublic: "\(global.id)-public"

images: {
	for key, value in images {
		"\(key)": #Image & value
	}
}

charts: {
	for key, value in charts {
		"\(key)": #Chart & value
	}
}

#ResourceReference: {
    name: string
    namespace: string
}

#Helm: {
	name: string
	dependsOn: [...#Helm] | *[]
    dependsOnExternal: [...#ResourceReference] | *[]
	...
}

helm: {
	for key, value in helm {
		"\(key)": #Helm & value & {
			name: key
		}
	}
}

#HelmRelease: {
	_name: string
	_chart: #Chart
	_values: _
	_dependencies: [...#Helm] | *[]
	_externalDependencies: [...#ResourceReference] | *[]

	apiVersion: "helm.toolkit.fluxcd.io/v2beta1"
	kind: "HelmRelease"
	metadata: {
		name: _name
   		namespace: release.namespace
	}
	spec: {
		interval: "1m0s"
		dependsOn: list.Concat([_externalDependencies, [
			for d in _dependencies {
				name: d.name
				namespace: release.namespace
			}
    	]])
		chart: {
			spec: _chart
		}
		values: _values
	}
}

output: {
	for name, r in helm {
		"\(name)": #HelmRelease & {
			_name: name
			_chart: r.chart
			_values: r.values
			_dependencies: r.dependsOn
            _externalDependencies: r.dependsOnExternal
		}
	}
}
`

type appConfig struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Namespaces  []string      `json:"namespaces"`
	Icon        template.HTML `json:"icon"`
}

type Rendered struct {
	Readme    string
	Resources map[string][]byte
}

type App interface {
	Name() string
	Description() string
	Icon() template.HTML
	Schema() Schema
	Namespaces() []string
	Render(derived Derived) (Rendered, error)
}

type cueApp struct {
	name        string
	description string
	icon        template.HTML
	namespace   string
	schema      Schema
	cfg         *cue.Value
}

type cueAppConfig struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func newCueApp(config *cue.Value) (cueApp, error) {
	if config == nil {
		return cueApp{}, fmt.Errorf("config not provided")
	}
	var cfg cueAppConfig
	if err := config.Decode(&cfg); err != nil {
		return cueApp{}, err
	}
	schema, err := NewCueSchema(config.LookupPath(cue.ParsePath("input")))
	if err != nil {
		return cueApp{}, err
	}
	return cueApp{
		name:        cfg.Name,
		description: cfg.Description,
		icon:        template.HTML(cfg.Icon),
		namespace:   cfg.Namespace,
		schema:      schema,
		cfg:         config,
	}, nil
}

func (a cueApp) Name() string {
	return a.name
}

func (a cueApp) Description() string {
	return a.description
}

func (a cueApp) Icon() template.HTML {
	return a.icon
}

func (a cueApp) Schema() Schema {
	return a.schema
}

func (a cueApp) Namespaces() []string {
	return []string{a.namespace}
}

func (a cueApp) Render(derived Derived) (Rendered, error) {
	ret := Rendered{
		Resources: make(map[string][]byte),
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(derived); err != nil {
		return Rendered{}, err
	}
	ctx := a.cfg.Context()
	d := ctx.CompileBytes(buf.Bytes())
	res := a.cfg.Unify(d).Eval()
	if err := res.Err(); err != nil {
		return Rendered{}, err
	}
	if err := res.Validate(); err != nil {
		return Rendered{}, err
	}
	readme, err := res.LookupPath(cue.ParsePath("readme")).String()
	if err != nil {
		return Rendered{}, err
	}
	ret.Readme = readme
	output := res.LookupPath(cue.ParsePath("output"))
	i, err := output.Fields()
	if err != nil {
		return Rendered{}, err
	}
	for i.Next() {
		name := fmt.Sprintf("%s.yaml", cleanName(i.Selector().String()))
		contents, err := cueyaml.Encode(i.Value())
		if err != nil {
			return Rendered{}, err
		}
		ret.Resources[name] = contents
	}
	return ret, nil
}

type AppRepository interface {
	GetAll() ([]App, error)
	Find(name string) (App, error)
}

type InMemoryAppRepository struct {
	apps []App
}

func NewInMemoryAppRepository(apps []App) InMemoryAppRepository {
	return InMemoryAppRepository{apps}
}

func (r InMemoryAppRepository) Find(name string) (App, error) {
	for _, a := range r.apps {
		if a.Name() == name {
			return a, nil
		}
	}
	return nil, fmt.Errorf("Application not found: %s", name)
}

func (r InMemoryAppRepository) GetAll() ([]App, error) {
	return r.apps, nil
}

func CreateAllApps() []App {
	return append(
		createApps(infraAppConfigs),
		CreateStoreApps()...,
	)
}

func CreateStoreApps() []App {
	return createApps(storeAppConfigs)
}

func createApps(configs []string) []App {
	ret := make([]App, 0)
	for _, cfgFile := range configs {
		cfg, err := readCueConfigFromFile(valuesTmpls, cfgFile)
		if err != nil {
			panic(err)
		}
		if app, err := newCueApp(cfg); err != nil {
			panic(err)
		} else {
			ret = append(ret, app)
		}
	}
	return ret
}

// func CreateAppMaddy(fs embed.FS, tmpls *template.Template) App {
// 	schema, err := readJSONSchemaFromFile(fs, "values-tmpl/maddy.jsonschema")
// 	if err != nil {
// 		panic(err)
// 	}
// 	return StoreApp{
// 		App{
// 			"maddy",
// 			[]string{"app-maddy"},
// 			[]*template.Template{
// 				tmpls.Lookup("maddy.yaml"),
// 			},
// 			schema,
// 			nil,
// 			nil,
// 		},
// 		`<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 48 48"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" d="M9.5 13c13.687 13.574 14.825 13.09 29 0"/><rect width="37" height="31" x="5.5" y="8.5" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" rx="2"/></svg>`,
// 		"SMPT/IMAP server to communicate via email.",
// 	}
// }

type httpAppRepository struct {
	apps []App
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
	InMemoryAppRepository
	fs billy.Filesystem
}

func NewFSAppRepository(fs billy.Filesystem) (AppRepository, error) {
	all, err := fs.ReadDir(".")
	if err != nil {
		return nil, err
	}
	apps := make([]App, 0)
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
		NewInMemoryAppRepository(apps),
		fs,
	}, nil
}

func loadApp(fs billy.Filesystem) (App, error) {
	items, err := fs.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var contents bytes.Buffer
	for _, i := range items {
		if i.IsDir() {
			continue
		}
		f, err := fs.Open(i.Name())
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if _, err := io.Copy(&contents, f); err != nil {
			return nil, err
		}
	}
	cfg, err := processCueConfig(contents.String())
	if err != nil {
		return nil, err
	}
	return newCueApp(cfg)
}

func cleanName(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\"", ""), "'", "")
}

func processCueConfig(contents string) (*cue.Value, error) {
	ctx := cuecontext.New()
	cfg := ctx.CompileString(cueBaseConfigImports + contents + cueBaseConfig)
	if err := cfg.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func readCueConfigFromFile(fs embed.FS, f string) (*cue.Value, error) {
	contents, err := fs.ReadFile(f)
	if err != nil {
		return nil, err
	}
	return processCueConfig(string(contents))
}

func createApp(fs embed.FS, configFile string) App {
	cfg, err := readCueConfigFromFile(fs, configFile)
	if err != nil {
		panic(err)
	}
	if app, err := newCueApp(cfg); err != nil {
		panic(err)
	} else {
		return app
	}
}
