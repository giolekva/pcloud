package installer

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

type Kind int

const (
	KindBoolean     Kind = 0
	KindInt              = 7
	KindString           = 1
	KindStruct           = 2
	KindNetwork          = 3
	KindAuth             = 5
	KindSSHKey           = 6
	KindNumber           = 4
	KindArrayString      = 8
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
}

var AuthSchema Schema = structSchema{
	name: "Auth",
	fields: []Field{
		Field{"enabled", basicSchema{"Enabled", KindBoolean, false}},
		Field{"groups", basicSchema{"Groups", KindString, false}},
	},
	advanced: false,
}

var SSHKeySchema Schema = structSchema{
	name: "SSH Key",
	fields: []Field{
		Field{"public", basicSchema{"Public Key", KindString, false}},
		Field{"private", basicSchema{"Private Key", KindString, false}},
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

func NewCueSchema(name string, v cue.Value) (Schema, error) {
	nameAttr := v.Attribute("name")
	if nameAttr.Err() == nil {
		name = nameAttr.Contents()
	}
	switch v.IncompleteKind() {
	case cue.StringKind:
		return basicSchema{name, KindString, false}, nil
	case cue.BoolKind:
		return basicSchema{name, KindBoolean, false}, nil
	case cue.NumberKind:
		return basicSchema{name, KindNumber, false}, nil
	case cue.IntKind:
		return basicSchema{name, KindInt, false}, nil
	case cue.ListKind:
		return basicSchema{name, KindArrayString, false}, nil
	case cue.StructKind:
		if isNetwork(v) {
			return basicSchema{name, KindNetwork, false}, nil
		} else if isAuth(v) {
			return basicSchema{name, KindAuth, false}, nil
		} else if isSSHKey(v) {
			return basicSchema{name, KindSSHKey, true}, nil
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
