package schema

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/itaysk/regogo"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/parser"
)

const jsonContentType = "application/json"

const getSchemaQuery = `{ "query": "{ getGQLSchema() { schema } }" }`

const addSchemaQuery = `{
  "query": "mutation { updateGQLSchema(input: {set: {schema: \"%s\"}}) { gqlSchema { id schema } } }" }`

type DgraphSchemaStore struct {
	dgraphAddress string
	gqlSchema     string
	schema        *ast.SchemaDocument
}

func NewDgraphSchemaStore(dgraphAddress string) (SchemaStore, error) {
	ret := &DgraphSchemaStore{dgraphAddress: dgraphAddress, gqlSchema: ""}
	if err := ret.fetchSchema(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *DgraphSchemaStore) Schema() *ast.SchemaDocument {
	return s.schema
}

func (s *DgraphSchemaStore) AddSchema(gqlSchema string) error {
	return s.SetSchema(s.gqlSchema + gqlSchema)
}

func (s *DgraphSchemaStore) SetSchema(gqlSchema string) error {
	glog.Info("Setting GraphQL schema:")
	glog.Info(gqlSchema)
	req := fmt.Sprintf(addSchemaQuery, strings.ReplaceAll(strings.ReplaceAll(gqlSchema, "\n", " "), "\t", " "))
	resp, err := http.Post(s.dgraphAddress, jsonContentType, bytes.NewReader([]byte(req)))
	if err != nil {
		return err
	}
	glog.Infof("Response status code: %d", resp.StatusCode)
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	glog.Infof("Result: %s", string(respBody))
	s.gqlSchema = gqlSchema
	return s.fetchSchema()
}

func (s *DgraphSchemaStore) fetchSchema() error {
	glog.Infof("Getting GraphQL schema with query: %s", getSchemaQuery)
	resp, err := http.Post(s.dgraphAddress, jsonContentType, bytes.NewReader([]byte(getSchemaQuery)))
	if err != nil {
		return err
	}
	glog.Infof("Response status code: %d", resp.StatusCode)
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	glog.Infof("Result: %s", string(respBody))
	gqlSchema, err := regogo.Get(string(respBody), "input.data.getGQLSchema.schema")
	if err != nil {
		return err
	}
	schema, gqlErr := parser.ParseSchema(&ast.Source{Input: gqlSchema.String()})
	if gqlErr != nil {
		return gqlErr
	}
	s.schema = schema
	return nil
}
