package installer

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	template "html/template"
	"net"
	"net/netip"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	cueyaml "cuelang.org/go/encoding/yaml"
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
)

//go:embed app_configs/dodo_app.cue
var dodoAppCue []byte

//go:embed app_configs/app_base.cue
var cueBaseConfig []byte

//go:embed app_configs/app_global_env.cue
var cueEnvAppGlobal []byte

//go:embed app_configs/app_global_infra.cue
var cueInfraAppGlobal []byte

type rendered struct {
	Name            string
	Readme          string
	Resources       CueAppData
	HelmCharts      HelmCharts
	ContainerImages map[string]ContainerImage
	Ports           []PortForward
	Data            CueAppData
	URL             string
	Help            []HelpDocument
	Icon            string
	Raw             []byte
}

type HelpDocument struct {
	Title    string
	Contents string
	Children []HelpDocument
}

type ContainerImage struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Name       string `json:"name"`
	Tag        string `json:"tag"`
}

type helmChartRef struct {
	Kind string `json:"kind"`
}

type HelmCharts struct {
	Git map[string]HelmChartGitRepo
}

type HelmChartGitRepo struct {
	Address string `json:"address"`
	Branch  string `json:"branch"`
	Path    string `json:"path"`
}

type EnvAppRendered struct {
	rendered
	Config AppInstanceConfig
}

type InfraAppRendered struct {
	rendered
	Config InfraAppInstanceConfig
}

type PortForward struct {
	Allocator     string `json:"allocator"`
	ReserveAddr   string `json:"reservator"`
	RemoveAddr    string `json:"deallocator"`
	Protocol      string `json:"protocol"`
	SourcePort    int    `json:"sourcePort"`
	TargetService string `json:"targetService"`
	TargetPort    int    `json:"targetPort"`
}

type AppType int

const (
	AppTypeInfra AppType = iota
	AppTypeEnv
)

type App interface {
	Name() string
	Type() AppType
	Slug() string
	Description() string
	Icon() template.HTML
	Schema() Schema
	Namespace() string
}

type InfraConfig struct {
	Name                 string   `json:"pcloudEnvName,omitempty"` // #TODO(gio): change to name
	PublicIP             []net.IP `json:"publicIP,omitempty"`
	InfraNamespacePrefix string   `json:"namespacePrefix,omitempty"`
	InfraAdminPublicKey  []byte   `json:"infraAdminPublicKey,omitempty"`
}

type InfraNetwork struct {
	Name               string `json:"name,omitempty"`
	IngressClass       string `json:"ingressClass,omitempty"`
	CertificateIssuer  string `json:"certificateIssuer,omitempty"`
	AllocatePortAddr   string `json:"allocatePortAddr,omitempty"`
	ReservePortAddr    string `json:"reservePortAddr,omitempty"`
	DeallocatePortAddr string `json:"deallocatePortAddr,omitempty"`
}

type InfraApp interface {
	App
	Render(release Release, infra InfraConfig, networks []InfraNetwork, values map[string]any, charts map[string]helmv2.HelmChartTemplateSpec) (InfraAppRendered, error)
}

type EnvNetwork struct {
	DNS            net.IP `json:"dns,omitempty"`
	DNSInClusterIP net.IP `json:"dnsInClusterIP,omitempty"`
	Ingress        net.IP `json:"ingress,omitempty"`
	Headscale      net.IP `json:"headscale,omitempty"`
	ServicesFrom   net.IP `json:"servicesFrom,omitempty"`
	ServicesTo     net.IP `json:"servicesTo,omitempty"`
}

func NewEnvNetwork(subnet net.IP) (EnvNetwork, error) {
	addr, err := netip.ParseAddr(subnet.String())
	if err != nil {
		return EnvNetwork{}, err
	}
	if !addr.Is4() {
		return EnvNetwork{}, fmt.Errorf("Expected IPv4, got %s instead", addr)
	}
	dns := addr.Next()
	ingress := dns.Next()
	headscale := ingress.Next()
	b := addr.AsSlice()
	if b[3] != 0 {
		return EnvNetwork{}, fmt.Errorf("Expected last byte to be zero, got %d instead", b[3])
	}
	b[3] = 10
	servicesFrom, ok := netip.AddrFromSlice(b)
	if !ok {
		return EnvNetwork{}, fmt.Errorf("Must not reach")
	}
	b[3] = 254
	servicesTo, ok := netip.AddrFromSlice(b)
	if !ok {
		return EnvNetwork{}, fmt.Errorf("Must not reach")
	}
	b[3] = b[2]
	b[2] = b[1]
	b[0] = 10
	b[1] = 44
	dnsInClusterIP, ok := netip.AddrFromSlice(b)
	if !ok {
		return EnvNetwork{}, fmt.Errorf("Must not reach")
	}
	return EnvNetwork{
		DNS:            net.ParseIP(dns.String()),
		DNSInClusterIP: net.ParseIP(dnsInClusterIP.String()),
		Ingress:        net.ParseIP(ingress.String()),
		Headscale:      net.ParseIP(headscale.String()),
		ServicesFrom:   net.ParseIP(servicesFrom.String()),
		ServicesTo:     net.ParseIP(servicesTo.String()),
	}, nil
}

type EnvConfig struct {
	Id              string     `json:"id,omitempty"`
	InfraName       string     `json:"pcloudEnvName,omitempty"`
	Domain          string     `json:"domain,omitempty"`
	PrivateDomain   string     `json:"privateDomain,omitempty"`
	ContactEmail    string     `json:"contactEmail,omitempty"`
	AdminPublicKey  string     `json:"adminPublicKey,omitempty"`
	PublicIP        []net.IP   `json:"publicIP,omitempty"`
	NameserverIP    []net.IP   `json:"nameserverIP,omitempty"`
	NamespacePrefix string     `json:"namespacePrefix,omitempty"`
	Network         EnvNetwork `json:"network,omitempty"`
}

type EnvApp interface {
	App
	Render(
		release Release,
		env EnvConfig,
		networks []Network,
		values map[string]any,
		charts map[string]helmv2.HelmChartTemplateSpec,
		vpnKeyGen VPNAuthKeyGenerator,
	) (EnvAppRendered, error)
}

type cueApp struct {
	name        string
	description string
	icon        template.HTML
	namespace   string
	schema      Schema
	cfg         cue.Value
	data        CueAppData
}

type CueAppData map[string][]byte

func ParseCueAppConfig(data CueAppData) (cue.Value, error) {
	ctx := cuecontext.New()
	buildCtx := build.NewContext()
	cfg := &load.Config{
		Context: buildCtx,
		Overlay: map[string]load.Source{},
	}
	names := make([]string, 0)
	for n, b := range data {
		a := fmt.Sprintf("/%s", n)
		names = append(names, a)
		cfg.Overlay[a] = load.FromString("package main\n\n" + string(b))
	}
	instances := load.Instances(names, cfg)
	for _, inst := range instances {
		if inst.Err != nil {
			return cue.Value{}, inst.Err
		}
	}
	if len(instances) != 1 {
		return cue.Value{}, fmt.Errorf("invalid")
	}
	ret := ctx.BuildInstance(instances[0])
	if ret.Err() != nil {
		return cue.Value{}, ret.Err()
	}
	if err := ret.Validate(); err != nil {
		return cue.Value{}, err
	}
	return ret, nil
}

func newCueApp(config cue.Value, data CueAppData) (cueApp, error) {
	cfg := struct {
		Name        string `json:"name"`
		Namespace   string `json:"namespace"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	}{}
	if err := config.Decode(&cfg); err != nil {
		return cueApp{}, err
	}
	schema, err := NewCueSchema("input", config.LookupPath(cue.ParsePath("input")))
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
		data:        data,
	}, nil
}

func ParseAndCreateNewCueApp(data CueAppData) (cueApp, error) {
	config, err := ParseCueAppConfig(data)
	if err != nil {
		return cueApp{}, err
	}
	return newCueApp(config, data)
}

func (a cueApp) Name() string {
	return a.name
}

func (a cueApp) Slug() string {
	return strings.ReplaceAll(strings.ToLower(a.name), " ", "-")
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

func (a cueApp) Namespace() string {
	return a.namespace
}

func (a cueApp) render(values map[string]any) (rendered, error) {
	ret := rendered{
		Name:      a.Slug(),
		Resources: make(CueAppData),
		HelmCharts: HelmCharts{
			Git: make(map[string]HelmChartGitRepo),
		},
		ContainerImages: make(map[string]ContainerImage),
		Ports:           make([]PortForward, 0),
		Data:            a.data,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(values); err != nil {
		return rendered{}, err
	}
	ctx := a.cfg.Context()
	d := ctx.CompileBytes(buf.Bytes())
	res := a.cfg.Unify(d).Eval()
	if err := res.Err(); err != nil {
		return rendered{}, err
	}
	if err := res.Validate(); err != nil {
		return rendered{}, err
	}
	full, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		return rendered{}, err
	}
	ret.Raw = full
	ret.Data["rendered.json"] = full
	readme, err := res.LookupPath(cue.ParsePath("readme")).String()
	if err != nil {
		return rendered{}, err
	}
	ret.Readme = readme
	if err := res.LookupPath(cue.ParsePath("portForward")).Decode(&ret.Ports); err != nil {
		return rendered{}, err
	}
	{
		charts := res.LookupPath(cue.ParsePath("output.charts"))
		i, err := charts.Fields()
		if err != nil {
			return rendered{}, err
		}
		for i.Next() {
			var chartRef helmChartRef
			if err := i.Value().Decode(&chartRef); err != nil {
				return rendered{}, err
			}
			if chartRef.Kind == "GitRepository" {
				var chart HelmChartGitRepo
				if err := i.Value().Decode(&chart); err != nil {
					return rendered{}, err
				}
				ret.HelmCharts.Git[cleanName(i.Selector().String())] = chart
			}
		}
	}
	{
		images := res.LookupPath(cue.ParsePath("output.images"))
		i, err := images.Fields()
		if err != nil {
			return rendered{}, err
		}
		for i.Next() {
			var img ContainerImage
			if err := i.Value().Decode(&img); err != nil {
				return rendered{}, err
			}
			ret.ContainerImages[cleanName(i.Selector().String())] = img
		}
	}
	{
		helm := res.LookupPath(cue.ParsePath("output.helm"))
		i, err := helm.Fields()
		if err != nil {
			return rendered{}, err
		}
		for i.Next() {
			if contents, err := cueyaml.Encode(i.Value()); err != nil {
				return rendered{}, err
			} else {
				name := fmt.Sprintf("%s.yaml", cleanName(i.Selector().String()))
				ret.Resources[name] = contents
			}
		}
	}
	{
		resources := res.LookupPath(cue.ParsePath("resources"))
		i, err := resources.Fields()
		if err != nil {
			return rendered{}, err
		}
		for i.Next() {
			if contents, err := cueyaml.Encode(i.Value()); err != nil {
				return rendered{}, err
			} else {
				name := fmt.Sprintf("%s.yaml", cleanName(i.Selector().String()))
				ret.Resources[name] = contents
			}
		}
	}
	helpValue := res.LookupPath(cue.ParsePath("help"))
	if helpValue.Exists() {
		if err := helpValue.Decode(&ret.Help); err != nil {
			return rendered{}, err
		}
	}
	url, err := res.LookupPath(cue.ParsePath("url")).String()
	if err != nil {
		return rendered{}, err
	}
	ret.URL = url
	icon, err := res.LookupPath(cue.ParsePath("icon")).String()
	if err != nil {
		return rendered{}, err
	}
	ret.Icon = icon
	return ret, nil
}

type cueEnvApp struct {
	cueApp
}

func NewCueEnvApp(data CueAppData) (EnvApp, error) {
	app, err := ParseAndCreateNewCueApp(data)
	if err != nil {
		return nil, err
	}
	return cueEnvApp{app}, nil
}

func NewDodoApp(appCfg []byte) (EnvApp, error) {
	return NewCueEnvApp(CueAppData{
		"app.cue":  appCfg,
		"base.cue": []byte(cueBaseConfig),
		"dodo.cue": dodoAppCue,
		"env.cue":  []byte(cueEnvAppGlobal),
	})
}

func (a cueEnvApp) Type() AppType {
	return AppTypeEnv
}

func (a cueEnvApp) Render(
	release Release,
	env EnvConfig,
	networks []Network,
	values map[string]any,
	charts map[string]helmv2.HelmChartTemplateSpec,
	vpnKeyGen VPNAuthKeyGenerator,
) (EnvAppRendered, error) {
	derived, err := deriveValues(values, values, a.Schema(), networks, vpnKeyGen)
	if err != nil {
		return EnvAppRendered{}, err
	}
	if charts == nil {
		charts = make(map[string]helmv2.HelmChartTemplateSpec)
	}
	ret, err := a.cueApp.render(map[string]any{
		"global":      env,
		"release":     release,
		"input":       derived,
		"localCharts": charts,
		"networks":    NetworkMap(networks),
	})
	if err != nil {
		return EnvAppRendered{}, err
	}
	return EnvAppRendered{
		rendered: ret,
		Config: AppInstanceConfig{
			AppId:   a.Slug(),
			Env:     env,
			Release: release,
			Values:  values,
			Input:   derived,
			URL:     ret.URL,
			Help:    ret.Help,
			Icon:    ret.Icon,
		},
	}, nil
}

type cueInfraApp struct {
	cueApp
}

func NewCueInfraApp(data CueAppData) (InfraApp, error) {
	app, err := ParseAndCreateNewCueApp(data)
	if err != nil {
		return nil, err
	}
	return cueInfraApp{app}, nil
}

func (a cueInfraApp) Type() AppType {
	return AppTypeInfra
}

func (a cueInfraApp) Render(release Release, infra InfraConfig, networks []InfraNetwork, values map[string]any, charts map[string]helmv2.HelmChartTemplateSpec) (InfraAppRendered, error) {
	if charts == nil {
		charts = make(map[string]helmv2.HelmChartTemplateSpec)
	}
	ret, err := a.cueApp.render(map[string]any{
		"global":      infra,
		"release":     release,
		"input":       values,
		"localCharts": charts,
		"networks":    InfraNetworkMap(networks),
	})
	if err != nil {
		return InfraAppRendered{}, err
	}
	return InfraAppRendered{
		rendered: ret,
		Config: InfraAppInstanceConfig{
			AppId:   a.Slug(),
			Infra:   infra,
			Release: release,
			Values:  values,
			Input:   values,
			URL:     ret.URL,
			Help:    ret.Help,
		},
	}, nil
}

func cleanName(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\"", ""), "'", "")
}

func join[T fmt.Stringer](items []T, sep string) string {
	var tmp []string
	for _, i := range items {
		tmp = append(tmp, i.String())
	}
	return strings.Join(tmp, ",")
}

func NetworkMap(networks []Network) map[string]Network {
	ret := make(map[string]Network)
	for _, n := range networks {
		ret[strings.ToLower(n.Name)] = n
	}
	return ret
}

func InfraNetworkMap(networks []InfraNetwork) map[string]InfraNetwork {
	ret := make(map[string]InfraNetwork)
	for _, n := range networks {
		ret[strings.ToLower(n.Name)] = n
	}
	return ret
}
