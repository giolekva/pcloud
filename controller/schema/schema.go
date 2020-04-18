package schema

import (
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/parser"
)

type SchemaStore interface {
	Schema() *ast.SchemaDocument
	SetSchema(gqlSchema string) error
	AddSchema(gqlSchema string) error
}

type InMemorySchemaStore struct {
	gqlSchema string
	schema    *ast.SchemaDocument
}

func NewInMemorySchemaStore() SchemaStore {
	return &InMemorySchemaStore{gqlSchema: ""}
}

func (s *InMemorySchemaStore) Schema() *ast.SchemaDocument {
	return s.schema
}

func (s *InMemorySchemaStore) AddSchema(gqlSchema string) error {
	return s.SetSchema(s.gqlSchema + gqlSchema)
}

func (s *InMemorySchemaStore) SetSchema(gqlSchema string) error {
	schema, err := parser.ParseSchema(&ast.Source{Input: gqlSchema})
	if err != nil {
		return err
	}
	s.schema = schema
	return nil
}
