package installer

import (
	"fmt"
)

type Release struct {
	Namespace string `json:"namespace"`
	RepoAddr  string `json:"repoAddr"`
	AppDir    string `json:"appDir"`
}

type Derived struct {
	Release Release        `json:"release"`
	Global  Values         `json:"global"`
	Values  map[string]any `json:"input"` // TODO(gio): rename to input
}

type Network struct {
	Name              string `json:"name,omitempty"`
	IngressClass      string `json:"ingressClass,omitempty"`
	CertificateIssuer string `json:"certificateIssuer,omitempty"`
	Domain            string `json:"domain,omitempty"`
	AllocatePortAddr  string `json:"allocatePortAddr,omitempty"`
}

type AppConfig struct {
	Id      string         `json:"id"`
	AppId   string         `json:"appId"`
	Config  map[string]any `json:"config"`
	Derived Derived        `json:"derived"`
}

func (a AppConfig) Input(schema Schema) map[string]any {
	ret, err := derivedToConfig(a.Derived.Values, schema)
	if err != nil {
		panic(err) // TODO(gio): handle
	}
	return ret
}

func deriveValues(values any, schema Schema, networks []Network) (map[string]any, error) {
	ret := make(map[string]any)
	for k, def := range schema.Fields() {
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
			continue
		}
		switch def.Kind() {
		case KindBoolean:
			ret[k] = v
		case KindString:
			ret[k] = v
		case KindInt:
			ret[k] = v
		case KindNetwork:
			n, err := findNetwork(networks, v.(string)) // TODO(giolekva): validate
			if err != nil {
				return nil, err
			}
			ret[k] = n
		case KindAuth:
			r, err := deriveValues(v, AuthSchema, networks)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		case KindSSHKey:
			r, err := deriveValues(v, SSHKeySchema, networks)
			if err != nil {
				return nil, err
			}
			ret[k] = r
		case KindStruct:
			r, err := deriveValues(v, def, networks)
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
	for k, def := range schema.Fields() {
		v, ok := derived[k]
		// TODO(gio): if missing use default value
		if !ok {
			continue
		}
		switch def.Kind() {
		case KindBoolean:
			ret[k] = v
		case KindString:
			ret[k] = v
		case KindInt:
			ret[k] = v
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
