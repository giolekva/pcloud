package installer

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	fluxcd "github.com/fluxcd/source-controller/api/v1beta2"
	"helm.sh/helm/v3/pkg/registry"
	// "github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"helm.sh/helm/v3/pkg/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//go:embed values-tmpl/rpuppy.cue
var rpuppyConfig []byte

type ContainerImage struct {
	Repository string
	Tag        string
	PullPolicy string
}

type Chart struct {
	Source ChartSource
	Chart  string
}

type ChartSource struct {
	Kind    string
	Address string
}

type ApplicationConfig struct {
	Images map[string]ContainerImage
	Charts map[string]Chart
}

type client struct {
	clientset dynamic.Interface
}

func (c *client) CreateHelmChart(chart fluxcd.HelmChart) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(chart); err != nil {
		return nil
	}
	var u unstructured.Unstructured
	if err := json.NewDecoder(&buf).Decode(&u.Object); err != nil {
		return err
	}
	_, err := c.clientset.Resource(schema.GroupVersionResource{Group: fluxcd.GroupVersion.Group, Version: fluxcd.GroupVersion.Version, Resource: "helmcharts"}).Namespace(chart.Namespace).Create(context.TODO(), &u, metav1.CreateOptions{})
	return err
}

func NewClient(kubeconfig string) (*client, error) {
	if kubeconfig == "" {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		c, err := dynamic.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		return &client{c}, nil

	} else {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		c, err := dynamic.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		return &client{c}, nil
	}
}

// const networkSchema = `
// #Network: {
// 	IngressClass: string
// 	CertificateIssuer: string
// 	Domain: string
// }

// value: %s

// valid: #Network & value
// `

type StringFormatter struct {
	s strings.Builder
}

func (f *StringFormatter) Write(b []byte) (n int, err error) {
	return f.s.Write(b)
}

func (f *StringFormatter) Width() (wid int, ok bool) {
	return 4, true
}

func (f *StringFormatter) Precision() (prec int, ok bool) {
	return 4, true
}

func (f *StringFormatter) Flag(c int) bool {
	return false
}

func IsNetwork(v cue.Value) bool {
	if v.Value().Kind() != cue.StructKind {
		return false
	}
	value := fmt.Sprintf("%#v", v)
	s := fmt.Sprintf(networkSchema, value)
	c := cuecontext.New()
	u := c.CompileString(s)
	return u.Err() == nil && u.Validate() == nil
}

func PrintSchema(v cue.Value) {
	f, _ := v.Fields()
	for f.Next() {
		fmt.Printf("%s\n", f.Selector())
		if IsNetwork(f.Value()) {
			fmt.Println("network")
		}
		PrintSchema(f.Value())
	}
}

func TestInput(t *testing.T) {
	return
	c := cuecontext.New()
	cfg := c.CompileBytes(rpuppyConfig)
	input := c.CompileString(`
global: {
  id: "foo"
}
input: {
  network: {
    name: "public"
    ingressClass: "dodo-ingress-public"
    certificateIssuer: "rpuppu-public"
    domain: "lekva.me"
  }
  subdomain: "rpuppy"
}
`)
	if cfg.Err() != nil {
		panic(cfg.Err())
	}
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	PrintSchema(cfg.Eval().LookupPath(cue.ParsePath("input")))
	out := cfg.Unify(input)
	if out.Err() != nil {
		panic(out.Err())
	}
	if err := out.Validate(); err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", out)
	e := out.Eval()
	if e.Err() != nil {
		panic(out.Err())
	}
	if err := e.Validate(); err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", e)
	fmt.Println(e.IsConcrete())
}

func TestParseApplicationConfig(t *testing.T) {
	return
	var r cue.Runtime
	i, err := r.Compile("rpuppy", rpuppyConfig)
	if err != nil {
		panic(err)
	}
	var cfg ApplicationConfig
	if err := i.Value().Decode(&cfg); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", cfg)
	_, err = NewClient("/Users/lekva/dev/src/pcloud/priv/kubeconfig-hetzner")
	if err != nil {
		panic(err)
	}

	for name, c := range cfg.Charts {
		chart := fluxcd.HelmChart{
			TypeMeta: metav1.TypeMeta{
				APIVersion: fluxcd.GroupVersion.String(),
				Kind:       "HelmChart",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "dodo",
			},
			Spec: fluxcd.HelmChartSpec{
				Chart: c.Chart,
				SourceRef: fluxcd.LocalHelmChartSourceReference{
					Kind: c.Source.Kind,
					Name: c.Source.Address,
				},
				Interval: metav1.Duration{time.Hour},
			},
		}
		fmt.Printf("%+v\n", chart)
		// if err := client.CreateHelmChart(chart); err != nil {
		// 	panic(err)
		// }
	}
}

type downloader struct {
	client *http.Client
}

func NewDownloader() *downloader {
	return &downloader{
		client: http.DefaultClient,
	}
}

func (d *downloader) Download(addr string, out io.Writer) error {
	resp, err := d.client.Get(addr)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}

func TestDownload(t *testing.T) {
	return
	// fs := memfs.New()
	fs := osfs.New("/tmp")
	func() {
		f, err := fs.Create("/chart")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		d := NewDownloader()
		if err := d.Download("http://localhost:9090/helmchart/dodo/rpuppy/rpuppy-0.0.1.tgz", f); err != nil {
			panic(err)
		}
	}()
	client, err := registry.NewClient()
	if err != nil {
		panic(err)
	}
	if err := client.Login("https://harbor.t46.lekva.me", registry.LoginOptBasicAuth("admin", "Harbor12345")); err != nil {
		panic(err)
	}
	defer client.Logout("https://harbor.t46.lekva.me")
	push := action.NewPushWithOpts(action.WithPushConfig(&action.Configuration{
		RegistryClient: client,
	}))
	fmt.Printf("%+v\n", push)
	res, err := push.Run("/tmp/chart", "oci://harbor.t46.lekva.me/library/charts")
	fmt.Println(res)
	if err != nil {
		panic(err)
	}
	// cfg, err := ActionConfigFactory{"/Users/lekva/dev/src/pcloud/priv/kubeconfig-hetzner"}.New("")
	// installer := action.NewInstall(config)
	// installer.Namespace = env.Name
	// installer.ReleaseName = "metallb-ns"
	// installer.Wait = true
	// installer.WaitForJobs = true
}

func TestAppManager(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	apps, err := r.GetAll()
	fmt.Println(apps)
	fmt.Println(err)
	for _, app := range apps {
		fmt.Println(app.Name())
	}
}
