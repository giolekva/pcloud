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

type Schema interface {
	Kind() Kind
	Fields() map[string]Schema
}

var AuthSchema Schema = structSchema{
	fields: map[string]Schema{
		"enabled": basicSchema{KindBoolean},
		"groups":  basicSchema{KindString},
	},
}

var SSHKeySchema Schema = structSchema{
	fields: map[string]Schema{
		"public":  basicSchema{KindString},
		"private": basicSchema{KindString},
	},
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
	kind Kind
}

func (s basicSchema) Kind() Kind {
	return s.kind
}

func (s basicSchema) Fields() map[string]Schema {
	return nil
}

type structSchema struct {
	fields map[string]Schema
}

func (s structSchema) Kind() Kind {
	return KindStruct
}

func (s structSchema) Fields() map[string]Schema {
	return s.fields
}

func NewCueSchema(v cue.Value) (Schema, error) {
	switch v.IncompleteKind() {
	case cue.StringKind:
		return basicSchema{KindString}, nil
	case cue.BoolKind:
		return basicSchema{KindBoolean}, nil
	case cue.NumberKind:
		return basicSchema{KindNumber}, nil
	case cue.IntKind:
		return basicSchema{KindInt}, nil
	case cue.ListKind:
		return basicSchema{KindArrayString}, nil
	case cue.StructKind:
		if isNetwork(v) {
			return basicSchema{KindNetwork}, nil
		} else if isAuth(v) {
			return basicSchema{KindAuth}, nil
		} else if isSSHKey(v) {
			return basicSchema{KindSSHKey}, nil
		}
		s := structSchema{make(map[string]Schema)}
		f, err := v.Fields(cue.Schema())
		if err != nil {
			return nil, err
		}
		for f.Next() {
			scm, err := NewCueSchema(f.Value())
			if err != nil {
				return nil, err
			}
			s.fields[f.Selector().String()] = scm
		}
		return s, nil
	default:
		return nil, fmt.Errorf("SHOULD NOT REACH!")
	}
}
