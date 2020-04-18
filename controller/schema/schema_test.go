package schema

import (
	"fmt"
	"log"
	"testing"
)

func TestInMemorySimple(t *testing.T) {
	s := NewInMemorySchemaStore()
	err := s.AddSchema(`
type M {
  X: Int
}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, def := range s.Schema().Definitions {
		fmt.Printf("%s - %s\n", def.Name, def.Kind)
	}
}

func TestInMemory(t *testing.T) {
	s := NewInMemorySchemaStore()
	err := s.AddSchema(`
type Image {
  id: ID!
  objectPath: String!
}

type ImageSegment {
  id: ID! @search
  upperLeftX: Float!
  upperLeftY: Float!
  lowerRightX: Float!
  lowerRightY: Float!
  sourceImage: Image!
}

extend type Image {
  segments: [ImageSegment]
}
`)
	if err != nil {
		t.Fatal(err)
	}
	for _, def := range s.Schema().Definitions {
		fmt.Printf("%s - %s\n", def.Name, def.Kind)
	}
}

func TestDgraph(t *testing.T) {
	s, err := NewDgraphSchemaStore("http://localhost:8080/admin")
	if err != nil {
		t.Fatal(err)
	}
	if s.Schema() != nil {
		for _, def := range s.Schema().Definitions {
			fmt.Printf("%s - %s\n", def.Name, def.Kind)
		}
	}
	err = s.AddSchema("type N { Y: ID! Z: Float }")
	if err != nil {
		t.Fatal(err)
	}
	if s.Schema() != nil {
		for _, def := range s.Schema().Definitions {
			fmt.Printf("%s - %s\n", def.Name, def.Kind)
		}
	}
	log.Print("123123")
	err = s.AddSchema("type M { X: Int }")
	if err != nil {
		t.Fatal(err)
	}
	if s.Schema() != nil {
		for _, def := range s.Schema().Definitions {
			fmt.Printf("%s - %s\n", def.Name, def.Kind)
		}
	}
}
