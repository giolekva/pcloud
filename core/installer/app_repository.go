package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-git/go-billy/v5"
	"sigs.k8s.io/yaml"
)

//go:embed values-tmpl
var valuesTmpls embed.FS

var storeEnvAppConfigs = []string{
	"values-tmpl/dodo-app.cue",
	"values-tmpl/virtual-machine.cue",
	// "values-tmpl/coder.cue",
	"values-tmpl/url-shortener.cue",
	"values-tmpl/matrix.cue",
	"values-tmpl/vaultwarden.cue",
	// "values-tmpl/open-project.cue",
	"values-tmpl/gerrit.cue",
	"values-tmpl/jenkins.cue",
	"values-tmpl/zot.cue",
	// "values-tmpl/penpot.cue",
	"values-tmpl/soft-serve.cue",
	"values-tmpl/pihole.cue",
	// "values-tmpl/maddy.cue",
	// "values-tmpl/qbittorrent.cue",
	// "values-tmpl/jellyfin.cue",
	"values-tmpl/rpuppy.cue",
	"values-tmpl/certificate-issuer-custom.cue",
}

var envAppConfigs = []string{
	"values-tmpl/dodo-app-instance.cue",
	"values-tmpl/dodo-app-instance-status.cue",
	"values-tmpl/certificate-issuer-private.cue",
	"values-tmpl/certificate-issuer-public.cue",
	"values-tmpl/appmanager.cue",
	"values-tmpl/core-auth.cue",
	"values-tmpl/metallb-ipaddresspool.cue",
	"values-tmpl/private-network.cue",
	"values-tmpl/welcome.cue",
	"values-tmpl/memberships.cue",
	"values-tmpl/headscale.cue",
	"values-tmpl/launcher.cue",
	"values-tmpl/env-dns.cue",
	"values-tmpl/launcher.cue",
	"values-tmpl/cluster-network.cue",
	"values-tmpl/longhorn.cue",
}

var infraAppConfigs = []string{
	"values-tmpl/cert-manager.cue",
	"values-tmpl/config-repo.cue",
	"values-tmpl/csi-driver-smb.cue",
	"values-tmpl/dns-gateway.cue",
	"values-tmpl/env-manager.cue",
	"values-tmpl/fluxcd-reconciler.cue",
	"values-tmpl/headscale-controller.cue",
	"values-tmpl/ingress-public.cue",
	"values-tmpl/resource-renderer-controller.cue",
	"values-tmpl/hydra-maester.cue",
}

type AppRepository interface {
	GetAll() ([]App, error)
	Find(name string) (App, error)
	Filter(query string) ([]App, error)
}

type InMemoryAppRepository struct {
	apps []App
}

func NewInMemoryAppRepository(apps []App) InMemoryAppRepository {
	return InMemoryAppRepository{apps}
}

func (r InMemoryAppRepository) Find(name string) (App, error) {
	for _, a := range r.apps {
		if a.Slug() == name {
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
		createInfraApps(),
		CreateAllEnvApps()...,
	)
}

func (r InMemoryAppRepository) Filter(query string) ([]App, error) {
	var filteredApps []App
	if query == "" {
		return r.GetAll()
	}
	for _, a := range r.apps {
		if strings.Contains(strings.ToLower(a.Name()), strings.ToLower(query)) {
			filteredApps = append(filteredApps, a)
		}
	}
	return filteredApps, nil
}

func CreateStoreApps() []App {
	return CreateEnvApps(storeEnvAppConfigs)
}

func CreateAllEnvApps() []App {
	return append(
		CreateStoreApps(),
		CreateEnvApps(envAppConfigs)...,
	)
}

func CreateEnvApps(configs []string) []App {
	ret := make([]App, 0)
	for _, cfgFile := range configs {
		contents, err := valuesTmpls.ReadFile(cfgFile)
		if err != nil {
			panic(err)
		}
		if app, err := NewCueEnvApp(CueAppData{
			"base.cue":   []byte(cueBaseConfig),
			"global.cue": []byte(cueEnvAppGlobal),
			"app.cue":    contents,
		}); err != nil {
			fmt.Println(cfgFile)
			panic(err)
		} else {
			ret = append(ret, app)
		}
	}
	return ret
}

func createInfraApps() []App {
	ret := make([]App, 0)
	for _, cfgFile := range infraAppConfigs {
		contents, err := valuesTmpls.ReadFile(cfgFile)
		if err != nil {
			panic(err)
		}
		if app, err := NewCueInfraApp(CueAppData{
			"base.cue":   []byte(cueBaseConfig),
			"global.cue": []byte(cueInfraAppGlobal),
			"app.cue":    contents,
		}); err != nil {
			fmt.Println(cfgFile)
			panic(err)
		} else {
			ret = append(ret, app)
		}
	}
	return ret
}

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
	return NewCueEnvApp(CueAppData{
		"base.cue": []byte(cueBaseConfig),
		"app.cue":  contents.Bytes(),
	})
}

// func readCueConfigFromFile(fs embed.FS, f string) (*cue.Value, error) {
// 	contents, err := fs.ReadFile(f)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return processCueConfig(string(contents))
// }

// func processCueConfig(contents string) (*cue.Value, error) {
// 	ctx := cuecontext.New()
// 	cfg := ctx.CompileString(contents + cueBaseConfig)
// 	if err := cfg.Err(); err != nil {
// 		return nil, err
// 	}
// 	if err := cfg.Validate(); err != nil {
// 		return nil, err
// 	}
// 	return &cfg, nil
// }

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

func FindEnvApp(r AppRepository, name string) (EnvApp, error) {
	app, err := r.Find(name)
	if err != nil {
		return nil, err
	}
	if a, ok := app.(EnvApp); ok {
		return a, nil
	} else {
		return nil, fmt.Errorf("not found")
	}
}

func FindInfraApp(r AppRepository, name string) (InfraApp, error) {
	app, err := r.Find(name)
	if err != nil {
		return nil, err
	}
	if a, ok := app.(InfraApp); ok {
		return a, nil
	} else {
		return nil, fmt.Errorf("not found")
	}
}
