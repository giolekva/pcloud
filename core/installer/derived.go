package installer

import (
	"fmt"
	"html/template"
	"strings"
)

const defaultClusterName = "default"

type Release struct {
	AppInstanceId string `json:"appInstanceId"`
	Namespace     string `json:"namespace"`
	RepoAddr      string `json:"repoAddr"`
	AppDir        string `json:"appDir"`
	ImageRegistry string `json:"imageRegistry,omitempty"`
}

type Network struct {
	Name               string `json:"name,omitempty"`
	IngressClass       string `json:"ingressClass,omitempty"`
	CertificateIssuer  string `json:"certificateIssuer,omitempty"`
	Domain             string `json:"domain,omitempty"`
	AllocatePortAddr   string `json:"allocatePortAddr,omitempty"`
	ReservePortAddr    string `json:"reservePortAddr,omitempty"`
	DeallocatePortAddr string `json:"deallocatePortAddr,omitempty"`
}

type InfraAppInstanceConfig struct {
	Id      string         `json:"id"`
	AppId   string         `json:"appId"`
	Infra   InfraConfig    `json:"infra"`
	Release Release        `json:"release"`
	Values  map[string]any `json:"values"`
	Input   map[string]any `json:"input"`
	URL     string         `json:"url"`
	Help    []HelpDocument `json:"help"`
	Icon    template.HTML  `json:"icon"`
}

type AppInstanceConfig struct {
	Id      string         `json:"id"`
	AppId   string         `json:"appId"`
	Env     EnvConfig      `json:"env"`
	Release Release        `json:"release"`
	Values  map[string]any `json:"values"`
	Input   map[string]any `json:"input"`
	URL     string         `json:"url"`
	Help    []HelpDocument `json:"help"`
	Icon    string         `json:"icon"`
}

func (a AppInstanceConfig) InputToValues(schema Schema) map[string]any {
	ret, err := derivedToConfig(a.Input, schema)
	if err != nil {
		panic(err)
	}
	return ret
}

func getField(v any, f string) any {
	for _, i := range strings.Split(f, ".") {
		vm := v.(map[string]any)
		v = vm[i]
	}
	return v
}

func deriveValues(
	root any,
	values any,
	schema Schema,
	networks []Network,
	clusters []Cluster,
	vpnKeyGen VPNAPIClient,
) (map[string]any, error) {
	ret := make(map[string]any)
	for _, f := range schema.Fields() {
		k := f.Name
		def := f.Schema
		// TODO(gio): validate that it is map
		v, ok := values.(map[string]any)[k]
		// TODO(gio): if missing use default value
		if !ok {
			if def.Kind() == KindSSHKey {
				key, err := NewECDSASSHKeyPair("tmp")
				if err != nil {
					return nil, err
				}
				ret[k] = map[string]string{
					"public":  string(key.RawAuthorizedKey()),
					"private": string(key.RawPrivateKey()),
				}
			}
			if def.Kind() == KindVPNAuthKey {
				enabled := true
				if v, ok := def.Meta()["enabledField"]; ok {
					// TODO(gio): Improve getField
					enabled, ok = getField(root, v).(bool)
					if !ok {
						enabled = false
						// TODO(gio): validate that enabled field exists in the schema
						// return nil, fmt.Errorf("could not resolve enabled: %+v %s %+v", def.Meta(), v, root)
					}
				}
				if !enabled {
					continue
				}
				var username string
				if v, ok := def.Meta()["username"]; ok {
					username = v
				} else if v, ok := def.Meta()["usernameField"]; ok {
					// TODO(gio): Improve getField
					username, ok = getField(root, v).(string)
					if !ok {
						return nil, fmt.Errorf("could not resolve username: %+v %s %+v", def.Meta(), v, root)
					}
				}
				authKey, err := vpnKeyGen.GenerateAuthKey(username)
				if err != nil {
					return nil, err
				}
				ret[k] = authKey
			}
			continue
		}
		switch def.Kind() {
		case KindBoolean:
			ret[k] = v
		case KindString:
			ret[k] = v
		case KindInt:
			ret[k] = v
		case KindPort:
			ret[k] = v
		case KindVPNAuthKey:
			ret[k] = v
		case KindArrayString:
			a, ok := v.([]string)
			if !ok {
				return nil, fmt.Errorf("expected string array")
			}
			ret[k] = a
		case KindNetwork:
			name, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("not a string")
			}
			n, err := findNetwork(networks, name)
			if err != nil {
				return nil, err
			}
			ret[k] = n
		case KindMultiNetwork:
			vv, ok := v.([]any)
			if !ok {
				return nil, fmt.Errorf("not an array")
			}
			picked := []Network{}
			for _, nn := range vv {
				name, ok := nn.(string)
				if !ok {
					return nil, fmt.Errorf("not a string")
				}
				n, err := findNetwork(networks, name)
				if err != nil {
					return nil, err
				}
				picked = append(picked, n)
			}
			ret[k] = picked
		case KindCluster:
			name, ok := v.(string)
			if !ok {
				// TODO(gio): validate that value has cluster schema
				ret[k] = v
			} else {
				c, err := findCluster(clusters, name)
				if err != nil {
					return nil, err
				}
				if c == nil {
					delete(ret, k)
				} else {
					ret[k] = c
				}
			}
		case KindAuth:
			r, err := deriveValues(root, v, AuthSchema, networks, clusters, vpnKeyGen)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		case KindSSHKey:
			r, err := deriveValues(root, v, SSHKeySchema, networks, clusters, vpnKeyGen)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		case KindStruct:
			r, err := deriveValues(root, v, def, networks, clusters, vpnKeyGen)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		default:
			return nil, fmt.Errorf("Should not reach!")
		}
	}
	return ret, nil
}

func derivedToConfig(derived map[string]any, schema Schema) (map[string]any, error) {
	ret := make(map[string]any)
	for _, f := range schema.Fields() {
		k := f.Name
		def := f.Schema
		v, ok := derived[k]
		// TODO(gio): if missing use default value
		if !ok {
			if def.Kind() == KindCluster {
				ret[k] = "default"
			}
			continue
		}
		switch def.Kind() {
		case KindBoolean:
			ret[k] = v
		case KindString:
			ret[k] = v
		case KindInt:
			ret[k] = v
		case KindPort:
			ret[k] = v
		case KindVPNAuthKey:
			ret[k] = v
		case KindArrayString:
			a, ok := v.([]string)
			if !ok {
				return nil, fmt.Errorf("expected string array")
			}
			ret[k] = a
		case KindNetwork:
			vm, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected map")
			}
			name, ok := vm["name"]
			if !ok {
				return nil, fmt.Errorf("expected network name")
			}
			ret[k] = name
		case KindMultiNetwork:
			nl, ok := v.([]any)
			if !ok {
				return nil, fmt.Errorf("expected map")
			}
			names := []string{}
			for _, n := range nl {
				i, ok := n.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("expected map")
				}
				name, ok := i["name"]
				if !ok {
					return nil, fmt.Errorf("expected network name")
				}
				names = append(names, name.(string))
			}
			ret[k] = names
		case KindAuth:
			vm, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected map")
			}
			r, err := derivedToConfig(vm, AuthSchema)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		case KindSSHKey:
			vm, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected map")
			}
			r, err := derivedToConfig(vm, SSHKeySchema)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		case KindStruct:
			vm, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected map")
			}
			r, err := derivedToConfig(vm, def)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		case KindCluster:
			vm, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected map")
			}
			name, ok := vm["name"]
			if !ok {
				return nil, fmt.Errorf("expected cluster name")
			}
			ret[k] = name
		default:
			return nil, fmt.Errorf("Should not reach!")
		}
	}
	return ret, nil
}

func findNetwork(networks []Network, name string) (Network, error) {
	for _, n := range networks {
		if n.Name == name {
			return n, nil
		}
	}
	return Network{}, fmt.Errorf("Network not found: %s", name)
}

func findCluster(clusters []Cluster, name string) (*Cluster, error) {
	if name == defaultClusterName {
		return nil, nil
	}
	for _, c := range clusters {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("Cluster not found: %s", name)
}
