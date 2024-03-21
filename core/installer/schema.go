package installer

import (
	"encoding/json"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

type Kind int

const (
	KindBoolean Kind = 0
	KindString       = 1
	KindStruct       = 2
	KindNetwork      = 3
	KindAuth         = 5
	KindNumber       = 4
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

const networkSchema = `
#Network: {
    name: string
	ingressClass: string
	certificateIssuer: string | *""
	domain: string
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
	case cue.StructKind:
		if isNetwork(v) {
			return basicSchema{KindNetwork}, nil
		} else if isAuth(v) {
			return basicSchema{KindAuth}, nil
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

func newSchema(schema map[string]any) (Schema, error) {
	switch schema["type"] {
	case "string":
		if r, ok := schema["role"]; ok && r == "network" {
			return basicSchema{KindNetwork}, nil
		} else {
			return basicSchema{KindString}, nil
		}
	case "object":
		s := structSchema{make(map[string]Schema)}
		props := schema["properties"].(map[string]any)
		for name, schema := range props {
			sm, _ := schema.(map[string]any)
			scm, err := newSchema(sm)
			if err != nil {
				return nil, err
			}
			s.fields[name] = scm
		}
		return s, nil
	default:
		return nil, fmt.Errorf("SHOULD NOT REACH!")
	}
}

func NewJSONSchema(schema string) (Schema, error) {
	ret := make(map[string]any)
	if err := json.NewDecoder(strings.NewReader(schema)).Decode(&ret); err != nil {
		return nil, err
	}
	return newSchema(ret)
}
