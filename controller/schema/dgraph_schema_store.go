package schema

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/itaysk/regogo"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/formatter"
	//"github.com/bradleyjkemp/memviz"
)

const jsonContentType = "application/json"
const textContentType = "text/plain"

// TODO(giolekva): escape
const getSchemaQuery = `{ getGQLSchema() { schema generatedSchema } }`
const setSchemaQuery = `mutation { updateGQLSchema(input: { set: { schema: "%s" } }) { gqlSchema { schema generatedSchema } } }`
const runQuery = `{ "query": "%s" }`
const eventTmpl = `
type %sEvent {
  id: ID!
  state: EventState! @search(by: [exact])
  node: %s! @hasInverse(field: events)
}

extend type %s {
  events: [%sEvent] @hasInverse(field: node)
}`

type DgraphClient struct {
	gqlAddress      string
	schemaAddress   string
	userSchema      string
	generatedSchema string
	schema          *ast.Schema
}

func NewDgraphClient(gqlAddress, schemaAddress string) (GraphQLClient, error) {
	ret := &DgraphClient{
		gqlAddress:      gqlAddress,
		schemaAddress:   schemaAddress,
		userSchema:      "",
		generatedSchema: ""}
	if err := ret.fetchSchema(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *DgraphClient) Schema() *ast.Schema {
	return s.schema
}

func (s *DgraphClient) AddSchema(gqlSchema string) error {
	extendedSchema, err := s.extendSchema(gqlSchema)
	if err != nil {
		return err
	} else {
		return s.SetSchema(extendedSchema)
	}
}

func (s *DgraphClient) SetSchema(gqlSchema string) error {
	glog.Info("Setting GraphQL schema")
	glog.Info(gqlSchema)
	resp, err := s.runQuery(
		fmt.Sprintf(setSchemaQuery, gqlSchema),
		s.schemaAddress)
	if err != nil {
		return err
	}
	data, err := regogo.Get(resp, "input.updateGQLSchema.gqlSchema")
	if err != nil {
		return err
	}
	return s.updateSchema(data.JSON())
}

func (s *DgraphClient) fetchSchema() error {
	glog.Infof("Getting GraphQL schema")
	resp, err := s.runQuery(getSchemaQuery, s.schemaAddress)
	if err != nil {
		return err
	}
	data, err := regogo.Get(resp, "input.getGQLSchema")
	if err != nil {
		return err
	}
	return s.updateSchema(data.JSON())
}

func (s *DgraphClient) updateSchema(resp string) error {
	userSchema, err := regogo.Get(resp, "input.schema")
	if err != nil {
		return err
	}
	generatedSchema, err := regogo.Get(resp, "input.generatedSchema")
	if err != nil {
		return err
	}
	schema, gqlErr := gqlparser.LoadSchema(&ast.Source{Input: generatedSchema.String()})
	if gqlErr != nil {
		return gqlErr
	}
	s.userSchema = userSchema.String()
	s.generatedSchema = generatedSchema.String()
	s.schema = schema
	return nil
}

func (s *DgraphClient) RunQuery(query string) (string, error) {
	q, gqlErr := gqlparser.LoadQuery(s.Schema(), query)
	if gqlErr != nil {
		return "", errors.New(gqlErr.Error())
	}
	rewritten := rewriteQuery(q, s.Schema())
	var b strings.Builder
	// TODO(giolekva): gqlparser should be reporting error back
	formatter.NewFormatter(&b).FormatQueryDocument(rewritten)
	query = b.String()
	return s.runQuery(query, s.gqlAddress)
}

func (s *DgraphClient) runQuery(query string, onAddr string) (string, error) {
	glog.Infof("Running GraphQL query: %s", query)
	queryJson := fmt.Sprintf(runQuery, fixWhitespaces(escapeQuery(query)))
	resp, err := http.Post(
		onAddr,
		jsonContentType,
		bytes.NewReader([]byte(queryJson)))
	if err != nil {
		return "", err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respStr := string(respBody)
	glog.Infof("Result: %s", string(respStr))
	// errStr, err := regogo.Get(respStr, "input.errors")
	// if err == nil {
	// 	return "", errors.New(errStr.JSON())
	// }
	data, err := regogo.Get(respStr, "input.data")
	if err != nil {
		return "", err
	}
	return data.JSON(), nil
}

func (s *DgraphClient) extendSchema(schema string) (string, error) {
	try := s.generatedSchema + "   " + schema
	parsed, gqlErr := gqlparser.LoadSchema(&ast.Source{Input: try})
	if gqlErr != nil {
		return "", errors.New(gqlErr.Error())
	}
	var extended strings.Builder
	_, err := io.WriteString(&extended, s.userSchema)
	if err != nil {
		return "", err
	}
	_, err = io.WriteString(&extended, schema)
	if err != nil {
		return "", err
	}
	for _, t := range parsed.Types {
		if shouldIgnoreDefinition(t) {
			continue
		}
		_, err := fmt.Fprintf(&extended, eventTmpl, t.Name, t.Name, t.Name, t.Name)
		if err != nil {
			return "", err
		}
	}
	return extended.String(), nil
}

func findDefinitionWithName(name string, s *ast.Schema) *ast.Definition {
	for n, d := range s.Types {
		if n == name {
			return d
		}
	}
	panic(fmt.Sprintf("Expected event input definiton for  %s", name))
}

func findEventnputDefinitionFor(d *ast.Definition, s *ast.Schema) *ast.Definition {
	if !strings.HasSuffix(d.Name, "Input") {
		panic(fmt.Sprintf("Expected input definiton, got %s", d.Name))
	}
	eventInput := fmt.Sprintf("%sEventInput", strings.TrimSuffix(d.Name, "Input"))
	return findDefinitionWithName(eventInput, s)
}

func newEventStateValue(s *ast.Schema) *ast.ChildValue {
	return &ast.ChildValue{
		Name:     "state",
		Position: nil,
		Value: &ast.Value{
			Raw:                "NEW",
			Children:           ast.ChildValueList{},
			Kind:               ast.EnumValue,
			Position:           nil,
			Definition:         findDefinitionWithName("EventState", s),
			VariableDefinition: nil,
			ExpectedType:       nil,
		},
	}
}

func newEventListValue(d *ast.Definition, s *ast.Schema) *ast.ChildValue {
	return &ast.ChildValue{
		Name:     "events",
		Position: nil,
		Value: &ast.Value{
			Raw:                "",
			Children:           ast.ChildValueList{newEventValue(d, s)},
			Kind:               ast.ListValue,
			Position:           nil,
			Definition:         findEventnputDefinitionFor(d, s),
			VariableDefinition: nil,
			ExpectedType:       nil,
		},
	}
}

func newEventValue(d *ast.Definition, s *ast.Schema) *ast.ChildValue {
	return &ast.ChildValue{
		Name:     "events",
		Position: nil,
		Value: &ast.Value{
			Raw:                "",
			Children:           ast.ChildValueList{newEventStateValue(s)},
			Kind:               ast.ObjectValue,
			Position:           nil,
			Definition:         findEventnputDefinitionFor(d, s),
			VariableDefinition: nil,
			ExpectedType:       nil,
		},
	}
}

func rewriteValue(v *ast.Value, s *ast.Schema) {
	if v == nil {
		panic("Received nil value")
	}
	switch v.Kind {
	case ast.Variable:
	case ast.IntValue:
	case ast.FloatValue:
	case ast.StringValue:
	case ast.BlockValue:
	case ast.BooleanValue:
	case ast.NullValue:
	case ast.EnumValue:
	case ast.ListValue:
		for _, c := range v.Children {
			rewriteValue(c.Value, s)
		}
	case ast.ObjectValue:
		for _, c := range v.Children {
			rewriteValue(c.Value, s)
		}
		if v.Definition.Kind == ast.InputObject &&
			!strings.HasSuffix(v.Definition.Name, "Event") {
			v.Children = append(v.Children, newEventListValue(v.Definition, s))
		}
	}
}

func rewriteQuery(q *ast.QueryDocument, s *ast.Schema) *ast.QueryDocument {
	for _, op := range q.Operations {
		if op.Operation != ast.Mutation {
			continue
		}
		for _, sel := range op.SelectionSet {
			field, ok := sel.(*ast.Field)
			if !ok {
				panic(sel)
			}
			for _, arg := range field.Arguments {
				rewriteValue(arg.Value, s)
			}
		}
	}
	return q

}

// TODO(giolekva): will be safer to use directive instead
func shouldIgnoreDefinition(d *ast.Definition) bool {
	return d.Kind != ast.Object ||
		d.Name == "Query" ||
		d.Name == "Mutation" ||
		strings.HasPrefix(d.Name, "__") ||
		strings.HasSuffix(d.Name, "Payload") ||
		strings.HasSuffix(d.Name, "Event")
}

func fixWhitespaces(schema string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(schema, "\n", " "), "\t", " ")
}

func escapeQuery(query string) string {
	return strings.ReplaceAll(query, "\"", "\\\"")
}
