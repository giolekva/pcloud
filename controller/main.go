package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/giolekva/pcloud/controller/schema"

	"github.com/golang/glog"
	"github.com/itaysk/regogo"
)

var port = flag.Int("port", 123, "Port to listen on.")
var graphqlAddress = flag.String("graphql_address", "", "GraphQL server address.")
var dgraphAdminAddress = flag.String("dgraph_admin_address", "", "Dgraph server admin address.")

const imgJson = `{ objectPath: \"%s\"}`
const insertQuery = `{ "query": "mutation { add%s(input: [%s]) { %s { id } } }" }`
const getQuery = `{ "query": "{ get%s(id: \"%s\") { id objectPath } } " }`

type GraphQLClient struct {
	serverAddress string
}

func (g *GraphQLClient) Insert(typ string, obj string) (string, error) {
	req := []byte(fmt.Sprintf(insertQuery, typ, obj, strings.ToLower(typ)))
	glog.Info("Insering new item, mutation query:")
	glog.Info(string(req))
	resp, err := http.Post(g.serverAddress, "application/json", bytes.NewReader(req))
	glog.Infof("Response status: %d", resp.StatusCode)
	if err != nil {
		return "", err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	glog.Infof("Response: %s", string(respBody))
	return string(respBody), nil
}

func (g *GraphQLClient) Get(typ string, id string) (string, error) {
	req := []byte(fmt.Sprintf(insertQuery, typ, id, strings.ToLower(typ)))
	glog.Info("Getting node with query:")
	glog.Info(string(req))
	resp, err := http.Post(g.serverAddress, "application/json", bytes.NewReader(req))
	glog.Infof("Response status: %s", resp.StatusCode)
	if err != nil {
		return "", err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	glog.Info(string(respBody))
	return string(respBody), nil
}

type MinioWebhook struct {
	gqlClient         *GraphQLClient
	dgraphAdminClient schema.SchemaStore
}

func (m *MinioWebhook) handler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if len(body) == 0 {
		return
	}
	glog.Infof("Received event from Minio: %s", string(body))
	if err != nil {
		http.Error(w, "Could not read HTTP request body", http.StatusInternalServerError)
		return
	}
	key, err := regogo.Get(string(body), "input.Key")
	if err != nil {
		http.Error(w, "Could not find object key", http.StatusBadRequest)
		return
	}
	resp, err := m.gqlClient.Insert("Image", fmt.Sprintf(imgJson, key.String()))
	if err != nil {
		http.Error(w, "Can not add given objects", http.StatusInternalServerError)
		return
	}
	id, err := regogo.Get(resp, "input.data.addImage.image[0]id")
	if err != nil {
		http.Error(w, "Could not extract node id", http.StatusInternalServerError)
		return
	}
	resp, err = m.gqlClient.Get("Image", id.String())
	if err != nil {
		http.Error(w, "Could not fetch node", http.StatusInternalServerError)
		return
	}
}

func main() {
	flag.Parse()
	dgraphAdminClient, err := schema.NewDgraphSchemaStore(*dgraphAdminAddress)
	if err != nil {
		panic(err)
	}
	err = dgraphAdminClient.SetSchema(`
	type Image {
	     id: ID!
	     objectPath: String! @search(by: [exact])
	     segments: [ImageSegment] @hasInverse(field: sourceImage)
	}

	type ImageSegment {
	     id: ID!
	     upperLeftX: Int!
	     upperLeftY: Int!
	     lowerRightX: Int!
	     lowerRightY: Int!
	     sourceImage: Image!
	     objectPath: String
	}`)
	if err != nil {
		panic(err)
	}
	mw := MinioWebhook{
		&GraphQLClient{*graphqlAddress},
		nil}
	http.HandleFunc("/minio_webhook", mw.handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
