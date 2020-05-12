package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/giolekva/pcloud/controller/schema"

	"github.com/golang/glog"
	"github.com/itaysk/regogo"
)

var kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file.")

var port = flag.Int("port", 123, "Port to listen on.")
var dgraphGqlAddress = flag.String("graphql_address", "", "GraphQL server address.")
var dgraphSchemaAddress = flag.String("dgraph_admin_address", "", "Dgraph server admin address.")

const imgJson = `{ objectPath: \"%s\"}`
const insertQuery = `mutation { add%s(input: [%s]) { %s { id } } }`
const getQuery = `{ "query": "{ get%s(id: \"%s\") { id objectPath } } " }`

type MinioWebhook struct {
	gql schema.GraphQLClient
}

type query struct {
	query     string
	operation string
	variables string
}

func extractQuery(r *http.Request) (*query, error) {
	if r.Method == "GET" {
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		q, ok := r.Form["query"]
		if !ok || len(q) != 1 {
			return nil, errors.New("Exactly one query must be provided")
		}
		return &query{query: q[0]}, nil
	} else {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		q, err := regogo.Get(string(body), "input.query")
		if err != nil {
			return nil, err
		}
		return &query{query: q.String()}, nil
	}
}

func (m *MinioWebhook) graphqlHandler(w http.ResponseWriter, r *http.Request) {
	glog.Infof("New GraphQL query received: %s", r.Method)
	q, err := extractQuery(r)
	if err != nil {
		glog.Error(err.Error())
		http.Error(w, "Could not extract query", http.StatusBadRequest)
	}
	resp, err := m.gql.RunQuery(q.query)
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, resp)
	w.Header().Set("Content-Type", "application/json")
}

func main() {
	flag.Parse()

	gqlClient, err := schema.NewDgraphClient(
		*dgraphGqlAddress, *dgraphSchemaAddress)
	if err != nil {
		panic(err)
	}
	err = gqlClient.SetSchema(`
enum EventState {
  NEW
  PROCESSING
  DONE
}

type Foo { bar: Int }`)
	if err != nil {
		panic(err)
	}
	err = gqlClient.AddSchema(`
	type Image {
	     id: ID!
	     objectPath: String! @search(by: [exact])
	}

	type ImageSegment {
	     id: ID!
	     upperLeftX: Float!
	     upperLeftY: Float!
	     lowerRightX: Float!
	     lowerRightY: Float!
	     sourceImage: Image! @hasInverse(field: segments)
	}

	extend type Image {
	     segments: [ImageSegment] @hasInverse(field: sourceImage)
	}`)
	if err != nil {
		panic(err)
	}
	mw := MinioWebhook{gqlClient}
	http.HandleFunc("/graphql", mw.graphqlHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
