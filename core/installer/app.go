package installer

import (
	"bytes"
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
)

// TODO(gio): import
const cueEnvAppGlobal = `
#Global: {
	id: string | *""
	pcloudEnvName: string | *""
	domain: string | *""
    privateDomain: string | *""
    contactEmail: string | *""
    adminPublicKey: string | *""
    publicIP: [...string] | *[]
    nameserverIP: [...string] | *[]
	namespacePrefix: string | *""
	network: #EnvNetwork
}

networks: {
	public: #Network & {
		name: "Public"
		ingressClass: "\(global.pcloudEnvName)-ingress-public"
		certificateIssuer: "\(global.id)-public"
		domain: global.domain
		allocatePortAddr: "http://port-allocator.\(global.pcloudEnvName)-ingress-public.svc.cluster.local/api/allocate"
	}
	private: #Network & {
		name: "Private"
		ingressClass: "\(global.id)-ingress-private"
		domain: global.privateDomain
		allocatePortAddr: "http://port-allocator.\(global.id)-ingress-private.svc.cluster.local/api/allocate"
	}
}

// TODO(gio): remove
ingressPrivate: "\(global.id)-ingress-private"
ingressPublic: "\(global.pcloudEnvName)-ingress-public"
issuerPrivate: "\(global.id)-private"
issuerPublic: "\(global.id)-public"

#Ingress: {
	auth: #Auth
	network: #Network
	subdomain: string
	service: close({
		name: string
		port: close({ name: string }) | close({ number: int & > 0 })
	})

	_domain: "\(subdomain).\(network.domain)"
    _authProxyHTTPPortName: "http"

	out: {
		images: {
			authProxy: #Image & {
				repository: "giolekva"
				name: "auth-proxy"
				tag: "latest"
				pullPolicy: "Always"
			}
		}
		charts: {
			ingress: #Chart & {
				chart: "charts/ingress"
				sourceRef: {
					kind: "GitRepository"
					name: "pcloud"
					namespace: global.id
				}
			}
			authProxy: #Chart & {
				chart: "charts/auth-proxy"
				sourceRef: {
					kind: "GitRepository"
					name: "pcloud"
					namespace: global.id
				}
			}
		}
		helm: {
			if auth.enabled {
				"auth-proxy": {
					chart: charts.authProxy
					values: {
						image: {
							repository: images.authProxy.fullName
							tag: images.authProxy.tag
							pullPolicy: images.authProxy.pullPolicy
						}
						upstream: "\(service.name).\(release.namespace).svc.cluster.local"
						whoAmIAddr: "https://accounts.\(global.domain)/sessions/whoami"
						loginAddr: "https://accounts-ui.\(global.domain)/login"
						membershipAddr: "http://memberships-api.\(global.id)-core-auth-memberships.svc.cluster.local/api/user"
						groups: auth.groups
						portName: _authProxyHTTPPortName
					}
				}
			}
			ingress: {
				chart: charts.ingress
				_service: service
				values: {
					domain: _domain
					ingressClassName: network.ingressClass
					certificateIssuer: network.certificateIssuer
					service: {
						if auth.enabled {
							name: "auth-proxy"
                            port: name: _authProxyHTTPPortName
						}
						if !auth.enabled {
							name: _service.name
							if _service.port.name != _|_ {
								port: name: _service.port.name
							}
							if _service.port.number != _|_ {
								port: number: _service.port.number
							}
						}
					}
				}
			}
		}
	}
}

ingress: {}

_ingressValidate: {
	for key, value in ingress {
		"\(key)": #Ingress & value
	}
}
`

const cueInfraAppGlobal = `
#Global: {
	pcloudEnvName: string | *""
    publicIP: [...string] | *[]
	namespacePrefix: string | *""
    infraAdminPublicKey: string | *""
}

// TODO(gio): remove
ingressPublic: "\(global.pcloudEnvName)-ingress-public"

ingress: {}
_ingressValidate: {}
`

const cueBaseConfig = `
import (
  "net"
)

name: string | *""
description: string | *""
readme: string | *""
icon: string | *""
namespace: string | *""

help: [...#HelpDocument] | *[]

#HelpDocument: {
	title: string
	contents: string
	children: [...#HelpDocument] | *[]
}

url: string | *""

#AppType: "infra" | "env"
appType: #AppType | *"env"

#Auth: {
  enabled: bool | *false // TODO(gio): enabled by default?
  groups: string | *"" // TODO(gio): []string
}

#Network: {
	name: string
	ingressClass: string
	certificateIssuer: string | *""
	domain: string
	allocatePortAddr: string
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

#EnvNetwork: {
	dns: net.IPv4
	dnsInClusterIP: net.IPv4
	ingress: net.IPv4
	headscale: net.IPv4
	servicesFrom: net.IPv4
	servicesTo: net.IPv4
}

#Release: {
	appInstanceId: string
	namespace: string
	repoAddr: string
	appDir: string
}

#PortForward: {
	allocator: string
	protocol: "TCP" | "UDP" | *"TCP"
	sourcePort: int
	targetService: string
	targetPort: int
}

portForward: [...#PortForward] | *[]

global: #Global
release: #Release

images: {
	for key, value in images {
		"\(key)": #Image & value
	}
    for _, value in _ingressValidate {
        for name, image in value.out.images {
            "\(name)": #Image & image
        }
    }
}

charts: {
	for key, value in charts {
		"\(key)": #Chart & value
	}
    for _, value in _ingressValidate {
        for name, chart in value.out.charts {
            "\(name)": #Chart & chart
        }
    }
}

#ResourceReference: {
    name: string
    namespace: string
}

#Helm: {
	name: string
	dependsOn: [...#ResourceReference] | *[]
	...
}

_helmValidate: {
	for key, value in helm {
		"\(key)": #Helm & value & {
			name: key
		}
	}
	for key, value in _ingressValidate {
		for ing, ingValue in value.out.helm {
            // TODO(gio): support multiple ingresses
			// "\(key)-\(ing)": #Helm & ingValue & {
			"\(ing)": #Helm & ingValue & {
				// name: "\(key)-\(ing)"
				name: ing
			}
		}
	}
}

#HelmRelease: {
	_name: string
	_chart: #Chart
	_values: _
	_dependencies: [...#ResourceReference] | *[]

	apiVersion: "helm.toolkit.fluxcd.io/v2beta1"
	kind: "HelmRelease"
	metadata: {
		name: _name
   		namespace: release.namespace
	}
	spec: {
		interval: "1m0s"
		dependsOn: _dependencies
		chart: {
			spec: _chart
		}
		values: _values
	}
}

output: {
	for name, r in _helmValidate {
		"\(name)": #HelmRelease & {
			_name: name
			_chart: r.chart
			_values: r.values
			_dependencies: r.dependsOn
		}
	}
}

#SSHKey: {
	public: string
	private: string
}
`

type rendered struct {
	Name      string
	Readme    string
	Resources CueAppData
	Ports     []PortForward
	Data      CueAppData
	Help      []HelpDocument
	Url       string
	Icon      string
}

type HelpDocument struct {
	Title    string
	Contents string
	Children []HelpDocument
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

type InfraApp interface {
	App
	Render(release Release, infra InfraConfig, values map[string]any) (InfraAppRendered, error)
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

// TODO(gio): rename to EnvConfig
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
	Render(release Release, env EnvConfig, values map[string]any) (EnvAppRendered, error)
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
		Ports:     make([]PortForward, 0),
		Data:      a.data,
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
	ret.Data["rendered.json"] = full
	readme, err := res.LookupPath(cue.ParsePath("readme")).String()
	if err != nil {
		return rendered{}, err
	}
	ret.Readme = readme
	if err := res.LookupPath(cue.ParsePath("portForward")).Decode(&ret.Ports); err != nil {
		return rendered{}, err
	}
	output := res.LookupPath(cue.ParsePath("output"))
	i, err := output.Fields()
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
	ret.Url = url
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

func (a cueEnvApp) Type() AppType {
	return AppTypeEnv
}

func (a cueEnvApp) Render(release Release, env EnvConfig, values map[string]any) (EnvAppRendered, error) {
	networks := CreateNetworks(env)
	derived, err := deriveValues(values, a.Schema(), networks)
	if err != nil {
		return EnvAppRendered{}, nil
	}
	ret, err := a.cueApp.render(map[string]any{
		"global":  env,
		"release": release,
		"input":   derived,
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
			Help:    ret.Help,
			Url:     ret.Url,
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

func (a cueInfraApp) Render(release Release, infra InfraConfig, values map[string]any) (InfraAppRendered, error) {
	ret, err := a.cueApp.render(map[string]any{
		"global":  infra,
		"release": release,
		"input":   values,
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
