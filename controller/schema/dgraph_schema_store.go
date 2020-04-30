package schema

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/itaysk/regogo"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

const jsonContentType = "application/json"
const textContentType = "text/plain"

// TODO(giolekva): escape
const getSchemaQuery = `{ getGQLSchema() { schema generatedSchema } }`
const setSchemaQuery = `mutation { updateGQLSchema(input: { set: { schema: "%s" } }) { gqlSchema { schema generatedSchema } } }`
const runQuery = `{ "query": "%s" }`

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
	return s.SetSchema(s.userSchema + gqlSchema)
}

func (s *DgraphClient) SetSchema(gqlSchema string) error {
	glog.Info("Setting GraphQL schema")
	glog.Info(gqlSchema)
	resp, err := s.runQuery(
		fmt.Sprintf(setSchemaQuery, sanitizeSchema(gqlSchema)),
		s.schemaAddress)
	if err != nil {
		return err
	}
	return s.updateSchema(resp)
}

func (s *DgraphClient) fetchSchema() error {
	glog.Infof("Getting GraphQL schema")
	resp, err := s.runQuery(getSchemaQuery, s.schemaAddress)
	if err != nil {
		return err
	}
	return s.updateSchema(resp)
}

func (s *DgraphClient) updateSchema(resp string) error {
	userSchema, err := regogo.Get(resp, "input.getGQLSchema.schema")
	if err != nil {
		return err
	}
	generatedSchema, err := regogo.Get(resp, "input.getGQLSchema.generatedSchema")
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
	_, gqlErr := gqlparser.LoadQuery(s.Schema(), query)
	if gqlErr != nil {
		return "", errors.New(gqlErr.Error())
	}
	return s.runQuery(query, s.gqlAddress)
}

func (s *DgraphClient) runQuery(query string, onAddr string) (string, error) {
	glog.Infof("Running GraphQL query: %s", query)
	queryJson := fmt.Sprintf(runQuery, sanitizeQuery(query))
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

func sanitizeSchema(schema string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(schema, "\n", " "), "\t", " ")
}

func sanitizeQuery(query string) string {
	return strings.ReplaceAll(query, "\"", "\\\"")
}
