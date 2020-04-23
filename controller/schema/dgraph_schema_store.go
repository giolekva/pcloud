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

const getSchemaQuery = `{ "query": "{ getGQLSchema() { generatedSchema } }" }`
const runQuery = `{ "query": "%s" }`

type DgraphClient struct {
	gqlAdddress   string
	schemaAddress string
	gqlSchema     string
	schema        *ast.Schema
}

func NewDgraphClient(gqlAddress, schemaAddress string) (GraphQLClient, error) {
	ret := &DgraphClient{
		gqlAdddress:   gqlAddress,
		schemaAddress: schemaAddress,
		gqlSchema:     ""}
	if err := ret.fetchSchema(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *DgraphClient) Schema() *ast.Schema {
	return s.schema
}

func (s *DgraphClient) AddSchema(gqlSchema string) error {
	return s.SetSchema(s.gqlSchema + gqlSchema)
}

func (s *DgraphClient) SetSchema(gqlSchema string) error {
	glog.Info("Setting GraphQL schema:")
	glog.Info(gqlSchema)
	resp, err := http.Post(
		s.schemaAddress+"/schema",
		textContentType,
		bytes.NewReader([]byte(gqlSchema)))
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

func (s *DgraphClient) fetchSchema() error {
	glog.Infof("Getting GraphQL schema with query: %s", getSchemaQuery)
	resp, err := http.Post(
		s.schemaAddress,
		jsonContentType,
		bytes.NewReader([]byte(getSchemaQuery)))
	if err != nil {
		return err
	}
	glog.Infof("Response status code: %d", resp.StatusCode)
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	glog.Infof("Result: %s", string(respBody))
	gqlSchema, err := regogo.Get(
		string(respBody),
		"input.data.getGQLSchema.generatedSchema")
	if err != nil {
		return err
	}
	schema, gqlErr := gqlparser.LoadSchema(&ast.Source{Input: gqlSchema.String()})
	if gqlErr != nil {
		return gqlErr
	}
	s.schema = schema
	return nil
}

func (s *DgraphClient) RunQuery(query string) (string, error) {
	_, gqlErr := gqlparser.LoadQuery(s.Schema(), query)
	if gqlErr != nil {
		return "", errors.New(gqlErr.Error())
	}
	glog.Infof("Running GraphQL query: %s", query)
	queryJson := fmt.Sprintf(runQuery, sanitizeQuery(query))
	resp, err := http.Post(
		s.gqlAdddress,
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
