package installer

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

type Kind int

const (
	KindBoolean      Kind = 0
	KindInt               = 7
	KindString            = 1
	KindStruct            = 2
	KindNetwork           = 3
	KindMultiNetwork      = 10
	KindAuth              = 5
	KindSSHKey            = 6
	KindNumber            = 4
	KindArrayString       = 8
	KindPort              = 9
	KindVPNAuthKey        = 11
)

type Field struct {
	Name   string
	Schema Schema
}

type Schema interface {
	Name() string
	Kind() Kind
	Fields() []Field
	Advanced() bool
	Meta() map[string]string
}

var AuthSchema Schema = structSchema{
	name: "Auth",
	fields: []Field{
		Field{"enabled", basicSchema{"Enabled", KindBoolean, false, nil}},
		Field{"groups", basicSchema{"Groups", KindString, false, nil}},
	},
	advanced: false,
}

var SSHKeySchema Schema = structSchema{
	name: "SSH Key",
	fields: []Field{
		Field{"public", basicSchema{"Public Key", KindString, false, nil}},
		Field{"private", basicSchema{"Private Key", KindString, false, nil}},
	},
	advanced: true,
}

const networkSchema = `
#Network: {
    name: string
	ingressClass: string
	certificateIssuer: string | *""
	domain: string
	allocatePortAddr: string
	reservePortAddr: string
	deallocatePortAddr: string
}

value: { %s }
`

func isNetwork(v cue.Value) bool {
	if v.Value().Kind() != cue.StructKind {
		return false
	}
	s := fmt.Sprintf(networkSchema, fmt.Sprintf("%#v", v))
	c := cuecontext.New()
	u := c.CompileString(s)
	network := u.LookupPath(cue.ParsePath("#Network"))
	vv := u.LookupPath(cue.ParsePath("value"))
	if err := network.Subsume(vv); err == nil {
		return true
	}
	return false
}

const multiNetworkSchema = `
#Network: {
    name: string
	ingressClass: string
	certificateIssuer: string | *""
	domain: string
	allocatePortAddr: string
	reservePortAddr: string
	deallocatePortAddr: string
}

#Networks: [...#Network]

value: %s
`

func isMultiNetwork(v cue.Value) bool {
	if v.Value().IncompleteKind() != cue.ListKind {
		return false
	}
	s := fmt.Sprintf(multiNetworkSchema, fmt.Sprintf("%#v", v))
	c := cuecontext.New()
	u := c.CompileString(s)
	networks := u.LookupPath(cue.ParsePath("#Networks"))
	vv := u.LookupPath(cue.ParsePath("value"))
	if err := networks.Subsume(vv); err == nil {
		return true
	}
	return false
}

const authSchema = `
#Auth: {
    enabled: bool | false
    groups: string | *""
}

value: { %s }
`

func isAuth(v cue.Value) bool {
	if v.Value().Kind() != cue.StructKind {
		return false
	}
	s := fmt.Sprintf(authSchema, fmt.Sprintf("%#v", v))
	c := cuecontext.New()
	u := c.CompileString(s)
	auth := u.LookupPath(cue.ParsePath("#Auth"))
	vv := u.LookupPath(cue.ParsePath("value"))
	if err := auth.Subsume(vv); err == nil {
		return true
	}
	return false
}

const sshKeySchema = `
#SSHKey: {
    public: string
    private: string
}

value: { %s }
`

func isSSHKey(v cue.Value) bool {
	if v.Value().Kind() != cue.StructKind {
		return false
	}
	s := fmt.Sprintf(sshKeySchema, fmt.Sprintf("%#v", v))
	c := cuecontext.New()
	u := c.CompileString(s)
	sshKey := u.LookupPath(cue.ParsePath("#SSHKey"))
	vv := u.LookupPath(cue.ParsePath("value"))
	if err := sshKey.Subsume(vv); err == nil {
		return true
	}
	return false
}

type basicSchema struct {
	name     string
	kind     Kind
	advanced bool
	meta     map[string]string
}

func (s basicSchema) Name() string {
	return s.name
}

func (s basicSchema) Kind() Kind {
	return s.kind
}

func (s basicSchema) Fields() []Field {
	return nil
}

func (s basicSchema) Advanced() bool {
	return s.advanced
}

func (s basicSchema) Meta() map[string]string {
	return s.meta
}

type structSchema struct {
	name     string
	fields   []Field
	advanced bool
}

func (s structSchema) Name() string {
	return s.name
}

func (s structSchema) Kind() Kind {
	return KindStruct
}

func (s structSchema) Fields() []Field {
	return s.fields
}

func (s structSchema) Advanced() bool {
	return s.advanced
}

func (s structSchema) Meta() map[string]string {
	return map[string]string{}
}

func NewCueSchema(name string, v cue.Value) (Schema, error) {
	nameAttr := v.Attribute("name")
	if nameAttr.Err() == nil {
		name = nameAttr.Contents()
	}
	role := ""
	roleAttr := v.Attribute("role")
	if roleAttr.Err() == nil {
		role = strings.ToLower(roleAttr.Contents())
	}
	switch v.IncompleteKind() {
	case cue.StringKind:
		if role == "vpnauthkey" {
			meta := map[string]string{}
			usernameFieldAttr := v.Attribute("usernameField")
			if usernameFieldAttr.Err() == nil {
				meta["usernameField"] = strings.ToLower(usernameFieldAttr.Contents())
			}
			usernameAttr := v.Attribute("username")
			if usernameAttr.Err() == nil {
				meta["username"] = strings.ToLower(usernameAttr.Contents())
			}
			if len(meta) != 1 {
				return nil, fmt.Errorf("invalid vpn auth key field meta: %+v", meta)
			}
			enabledFieldAttr := v.Attribute("enabledField")
			if enabledFieldAttr.Err() == nil {
				meta["enabledField"] = strings.ToLower(enabledFieldAttr.Contents())
			}
			return basicSchema{name, KindVPNAuthKey, true, meta}, nil
		} else {
			return basicSchema{name, KindString, false, nil}, nil
		}
	case cue.BoolKind:
		return basicSchema{name, KindBoolean, false, nil}, nil
	case cue.NumberKind:
		return basicSchema{name, KindNumber, false, nil}, nil
	case cue.IntKind:
		if role == "port" {
			return basicSchema{name, KindPort, true, nil}, nil
		} else {
			return basicSchema{name, KindInt, false, nil}, nil
		}
	case cue.ListKind:
		if isMultiNetwork(v) {
			return basicSchema{name, KindMultiNetwork, false, nil}, nil
		}
		return basicSchema{name, KindArrayString, false, nil}, nil
	case cue.StructKind:
		if isNetwork(v) {
			return basicSchema{name, KindNetwork, false, nil}, nil
		} else if isAuth(v) {
			return basicSchema{name, KindAuth, false, nil}, nil
		} else if isSSHKey(v) {
			return basicSchema{name, KindSSHKey, true, nil}, nil
		}
		s := structSchema{name, make([]Field, 0), false}
		f, err := v.Fields(cue.Schema())
		if err != nil {
			return nil, err
		}
		for f.Next() {
			scm, err := NewCueSchema(f.Selector().String(), f.Value())
			if err != nil {
				return nil, err
			}
			s.fields = append(s.fields, Field{f.Selector().String(), scm})
		}
		return s, nil
	default:
		return nil, fmt.Errorf("SHOULD NOT REACH!")
	}
}
