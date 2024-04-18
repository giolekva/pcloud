package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	template "html/template"
	"net"
	"strings"

	"cuelang.org/go/cue"
	cueyaml "cuelang.org/go/encoding/yaml"
)

// TODO(gio): import
const cueBaseConfig = `
name: string | *""
description: string | *""
readme: string | *""
icon: string | *""
namespace: string | *""

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

_ingressPrivate: "\(global.id)-ingress-private"
_ingressPublic: "\(global.pcloudEnvName)-ingress-public"
_issuerPrivate: "\(global.id)-private"
_issuerPublic: "\(global.id)-public"

_IngressWithAuthProxy: {
	inp: {
		auth: #Auth
		network: #Network
		subdomain: string
		serviceName: string
		port: { name: string } | { number: int & > 0 }
	}

	_domain: "\(inp.subdomain).\(inp.network.domain)"
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
			if inp.auth.enabled {
				"auth-proxy": {
					chart: charts.authProxy
					values: {
						image: {
							repository: images.authProxy.fullName
							tag: images.authProxy.tag
							pullPolicy: images.authProxy.pullPolicy
						}
						upstream: "\(inp.serviceName).\(release.namespace).svc.cluster.local"
						whoAmIAddr: "https://accounts.\(global.domain)/sessions/whoami"
						loginAddr: "https://accounts-ui.\(global.domain)/login"
						membershipAddr: "http://memberships-api.\(global.id)-core-auth-memberships.svc.cluster.local/api/user"
						groups: inp.auth.groups
						portName: _authProxyHTTPPortName
					}
				}
			}
			ingress: {
				chart: charts.ingress
				values: {
					domain: _domain
					ingressClassName: inp.network.ingressClass
					certificateIssuer: inp.network.certificateIssuer
					service: {
						if inp.auth.enabled {
							name: "auth-proxy"
                            port: name: _authProxyHTTPPortName
						}
						if !inp.auth.enabled {
							name: inp.serviceName
							if inp.port.name != _|_ {
								port: name: inp.port.name
							}
							if inp.port.number != _|_ {
								port: number: inp.port.number
							}
						}
					}
				}
			}
		}
	}
}

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
	dependsOn: [...#ResourceReference] | *[]
	...
}

helmValidate: {
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
	for name, r in helmValidate {
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

type Rendered struct {
	Name      string
	Readme    string
	Resources map[string][]byte
	Ports     []PortForward
	Config    AppInstanceConfig
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
	Type() AppType
	Name() string
	Description() string
	Icon() template.HTML
	Schema() Schema
	Namespace() string
}

type InfraConfig struct {
	Name                 string   `json:"pcloudEnvName"` // #TODO(gio): change to name
	PublicIP             []net.IP `json:"publicIP"`
	InfraNamespacePrefix string   `json:"namespacePrefix"`
	InfraAdminPublicKey  []byte   `json:"infraAdminPublicKey"`
}

type InfraApp interface {
	App
	Render(release Release, infra InfraConfig, values map[string]any) (Rendered, error)
}

// TODO(gio): rename to EnvConfig
type AppEnvConfig struct {
	Id              string   `json:"id"`
	InfraName       string   `json:"pcloudEnvName"`
	Domain          string   `json:"domain"`
	PrivateDomain   string   `json:"privateDomain"`
	ContactEmail    string   `json:"contactEmail"`
	PublicIP        []net.IP `json:"publicIP"`
	NamespacePrefix string   `json:"namespacePrefix"`
}

type EnvApp interface {
	App
	Render(release Release, env AppEnvConfig, values map[string]any) (Rendered, error)
}

type cueApp struct {
	name        string
	description string
	icon        template.HTML
	namespace   string
	schema      Schema
	cfg         *cue.Value
}

func newCueApp(config *cue.Value) (cueApp, error) {
	if config == nil {
		return cueApp{}, fmt.Errorf("config not provided")
	}
	cfg := struct {
		Name        string `json:"name"`
		Namespace   string `json:"namespace"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	}{}
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

func (a cueApp) Namespace() string {
	return a.namespace
}

func (a cueApp) render(values map[string]any) (Rendered, error) {
	ret := Rendered{
		Name:      a.Name(),
		Resources: make(map[string][]byte),
		Ports:     make([]PortForward, 0),
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(values); err != nil {
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
	if err := res.LookupPath(cue.ParsePath("portForward")).Decode(&ret.Ports); err != nil {
		return Rendered{}, err
	}
	output := res.LookupPath(cue.ParsePath("output"))
	i, err := output.Fields()
	if err != nil {
		return Rendered{}, err
	}
	for i.Next() {
		if contents, err := cueyaml.Encode(i.Value()); err != nil {
			return Rendered{}, err
		} else {
			name := fmt.Sprintf("%s.yaml", cleanName(i.Selector().String()))
			ret.Resources[name] = contents
		}
	}
	return ret, nil
}

type cueEnvApp struct {
	cueApp
}

func NewCueEnvApp(config *cue.Value) (EnvApp, error) {
	app, err := newCueApp(config)
	if err != nil {
		return nil, err
	}
	return cueEnvApp{app}, nil
}

func (a cueEnvApp) Type() AppType {
	return AppTypeEnv
}

func (a cueEnvApp) Render(release Release, env AppEnvConfig, values map[string]any) (Rendered, error) {
	networks := CreateNetworks(env)
	derived, err := deriveValues(values, a.Schema(), networks)
	if err != nil {
		return Rendered{}, nil
	}
	ret, err := a.cueApp.render(map[string]any{
		"global":  env,
		"release": release,
		"input":   derived,
	})
	if err != nil {
		return Rendered{}, err
	}
	ret.Config = AppInstanceConfig{
		AppId:   a.Name(),
		Env:     env,
		Release: release,
		Values:  values,
		Input:   derived,
	}
	return ret, nil
}

type cueInfraApp struct {
	cueApp
}

func NewCueInfraApp(config *cue.Value) (InfraApp, error) {
	app, err := newCueApp(config)
	if err != nil {
		return nil, err
	}
	return cueInfraApp{app}, nil
}

func (a cueInfraApp) Type() AppType {
	return AppTypeInfra
}

func (a cueInfraApp) Render(release Release, infra InfraConfig, values map[string]any) (Rendered, error) {
	return a.cueApp.render(map[string]any{
		"global":  infra,
		"release": release,
		"input":   values,
	})
}

func cleanName(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\"", ""), "'", "")
}
