package schema

import (
	"github.com/vektah/gqlparser/ast"
)

type GraphQLClient interface {
	Schema() *ast.Schema
	SetSchema(schema string) error
	AddSchema(schema string) error
	RunQuery(query string) (string, error)
}
