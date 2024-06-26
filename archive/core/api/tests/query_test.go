package tests

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"

	"github.com/bradleyjkemp/memviz"
)

var gqlSchema = `#######################\n# Input Schema\n#######################\n\ntype Image {\n\tid: ID!\n\tobjectPath: String! @search(by: [exact])\n\tsegments(filter: ImageSegmentFilter, order: ImageSegmentOrder, first: Int, offset: Int): [ImageSegment] @hasInverse(field: sourceImage)\n}\n\ntype ImageSegment {\n\tid: ID!\n\tupperLeftX: Float!\n\tupperLeftY: Float!\n\tlowerRightX: Float!\n\tlowerRightY: Float!\n\tsourceImage(filter: ImageFilter): Image! @hasInverse(field: segments)\n\tobjectPath: String\n}\n\n#######################\n# Extended Definitions\n#######################\n\nscalar DateTime\n\nenum DgraphIndex {\n\tint\n\tfloat\n\tbool\n\thash\n\texact\n\tterm\n\tfulltext\n\ttrigram\n\tregexp\n\tyear\n\tmonth\n\tday\n\thour\n}\n\ndirective @hasInverse(field: String!) on FIELD_DEFINITION\ndirective @search(by: [DgraphIndex!]) on FIELD_DEFINITION\ndirective @dgraph(type: String, pred: String) on OBJECT | INTERFACE | FIELD_DEFINITION\ndirective @id on FIELD_DEFINITION\ndirective @secret(field: String!, pred: String) on OBJECT | INTERFACE\n\ninput IntFilter {\n\teq: Int\n\tle: Int\n\tlt: Int\n\tge: Int\n\tgt: Int\n}\n\ninput FloatFilter {\n\teq: Float\n\tle: Float\n\tlt: Float\n\tge: Float\n\tgt: Float\n}\n\ninput DateTimeFilter {\n\teq: DateTime\n\tle: DateTime\n\tlt: DateTime\n\tge: DateTime\n\tgt: DateTime\n}\n\ninput StringTermFilter {\n\tallofterms: String\n\tanyofterms: String\n}\n\ninput StringRegExpFilter {\n\tregexp: String\n}\n\ninput StringFullTextFilter {\n\talloftext: String\n\tanyoftext: String\n}\n\ninput StringExactFilter {\n\teq: String\n\tle: String\n\tlt: String\n\tge: String\n\tgt: String\n}\n\ninput StringHashFilter {\n\teq: String\n}\n\n#######################\n# Generated Types\n#######################\n\ntype AddImagePayload {\n\timage(filter: ImageFilter, order: ImageOrder, first: Int, offset: Int): [Image]\n\tnumUids: Int\n}\n\ntype AddImageSegmentPayload {\n\timagesegment(filter: ImageSegmentFilter, order: ImageSegmentOrder, first: Int, offset: Int): [ImageSegment]\n\tnumUids: Int\n}\n\ntype DeleteImagePayload {\n\tmsg: String\n\tnumUids: Int\n}\n\ntype DeleteImageSegmentPayload {\n\tmsg: String\n\tnumUids: Int\n}\n\ntype UpdateImagePayload {\n\timage(filter: ImageFilter, order: ImageOrder, first: Int, offset: Int): [Image]\n\tnumUids: Int\n}\n\ntype UpdateImageSegmentPayload {\n\timagesegment(filter: ImageSegmentFilter, order: ImageSegmentOrder, first: Int, offset: Int): [ImageSegment]\n\tnumUids: Int\n}\n\n#######################\n# Generated Enums\n#######################\n\nenum ImageOrderable {\n\tobjectPath\n}\n\nenum ImageSegmentOrderable {\n\tupperLeftX\n\tupperLeftY\n\tlowerRightX\n\tlowerRightY\n\tobjectPath\n}\n\n#######################\n# Generated Inputs\n#######################\n\ninput AddImageInput {\n\tobjectPath: String!\n\tsegments: [ImageSegmentRef]\n}\n\ninput AddImageSegmentInput {\n\tupperLeftX: Float!\n\tupperLeftY: Float!\n\tlowerRightX: Float!\n\tlowerRightY: Float!\n\tsourceImage: ImageRef!\n\tobjectPath: String\n}\n\ninput ImageFilter {\n\tid: [ID!]\n\tobjectPath: StringExactFilter\n\tand: ImageFilter\n\tor: ImageFilter\n\tnot: ImageFilter\n}\n\ninput ImageOrder {\n\tasc: ImageOrderable\n\tdesc: ImageOrderable\n\tthen: ImageOrder\n}\n\ninput ImagePatch {\n\tobjectPath: String\n\tsegments: [ImageSegmentRef]\n}\n\ninput ImageRef {\n\tid: ID\n\tobjectPath: String\n\tsegments: [ImageSegmentRef]\n}\n\ninput ImageSegmentFilter {\n\tid: [ID!]\n\tnot: ImageSegmentFilter\n}\n\ninput ImageSegmentOrder {\n\tasc: ImageSegmentOrderable\n\tdesc: ImageSegmentOrderable\n\tthen: ImageSegmentOrder\n}\n\ninput ImageSegmentPatch {\n\tupperLeftX: Float\n\tupperLeftY: Float\n\tlowerRightX: Float\n\tlowerRightY: Float\n\tsourceImage: ImageRef\n\tobjectPath: String\n}\n\ninput ImageSegmentRef {\n\tid: ID\n\tupperLeftX: Float\n\tupperLeftY: Float\n\tlowerRightX: Float\n\tlowerRightY: Float\n\tsourceImage: ImageRef\n\tobjectPath: String\n}\n\ninput UpdateImageInput {\n\tfilter: ImageFilter!\n\tset: ImagePatch\n\tremove: ImagePatch\n}\n\ninput UpdateImageSegmentInput {\n\tfilter: ImageSegmentFilter!\n\tset: ImageSegmentPatch\n\tremove: ImageSegmentPatch\n}\n\n#######################\n# Generated Query\n#######################\n\ntype Query {\n\tgetImage(id: ID!): Image\n\tqueryImage(filter: ImageFilter, order: ImageOrder, first: Int, offset: Int): [Image]\n\tgetImageSegment(id: ID!): ImageSegment\n\tqueryImageSegment(filter: ImageSegmentFilter, order: ImageSegmentOrder, first: Int, offset: Int): [ImageSegment]\n}\n\n#######################\n# Generated Mutations\n#######################\n\ntype Mutation {\n\taddImage(input: [AddImageInput!]!): AddImagePayload\n\tupdateImage(input: UpdateImageInput!): UpdateImagePayload\n\tdeleteImage(filter: ImageFilter!): DeleteImagePayload\n\taddImageSegment(input: [AddImageSegmentInput!]!): AddImageSegmentPayload\n\tupdateImageSegment(input: UpdateImageSegmentInput!): UpdateImageSegmentPayload\n\tdeleteImageSegment(filter: ImageSegmentFilter!): DeleteImageSegmentPayload\n}\n`

var simpleSchema = `
type ABC {
  id: ID!
  x: Int
}

input ABCInput {
  x: Int
}

extend type ABC {
  event: ID
}
`

var enumSchema = `
enum EventState {
  NEW
  PROCESSING
  DONE
}

input Event {
  id: ID
  state: EventState!
}

type E {
  foo: String
}

type Mutation {
  addEvent(input: Event!): E
}
`

// TODO(giolekva): will be safer to use directive instead
func shouldIgnoreDefinition(d *ast.Definition) bool {
	return d.Kind != ast.Object ||
		d.Name == "Query" ||
		d.Name == "Mutation" ||
		strings.HasPrefix(d.Name, "__") ||
		strings.HasSuffix(d.Name, "Payload")
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func TestParseSchema(t *testing.T) {
	schema := getSchema(gqlSchema)
	var extended strings.Builder
	_, err := io.WriteString(&extended, gqlSchema)
	checkErr(err)
	for _, n := range schema.Types {
		if shouldIgnoreDefinition(n) {
			continue
		}
		_, err := fmt.Fprintf(&extended, "\nextend type %s {\n event: ID\n}", n.Name)
		checkErr(err)
	}
	fmt.Print(&extended)
	f, _ := os.Create("schema.dot")
	memviz.Map(f, schema)
	f.Close()
}

func TestParseQuery(t *testing.T) {
	schema := getSchema(enumSchema)
	query, err := gqlparser.LoadQuery(schema, `mutation {
addEvent(input: {
  state: NEW
}) {
  foo
}
}`)
	if err != nil {
		panic(err)
	}
	f, _ := os.Create("query.dot")
	memviz.Map(f, query)
	f.Close()
}

func getSchema(schema string) *ast.Schema {
	s, err := gqlparser.LoadSchema(&ast.Source{
		Input: strings.ReplaceAll(strings.ReplaceAll(schema, "\\n", "\n"), "\\t", "\t")})
	if err != nil {
		panic(err)
	}
	return s
}
