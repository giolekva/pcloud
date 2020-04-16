package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var port = flag.Int("port", 123, "Port to listen on.")
var graphql_address = flag.String("graphql_address", "", "GraphQL server address.")

func minio_webhook_handler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if len(body) == 0 {
		return
	}
	log.Print(string(body))
	if err != nil {
		log.Print("-----")
		log.Print(err)
		http.Error(w, "Could not read HTTP request body", http.StatusInternalServerError)
		return
	}
	event := make(map[string]interface{})
	err = json.Unmarshal(body, &event)
	if err != nil {
		log.Print("++++++")
		log.Print(err)
		http.Error(w, "Could not parse Event JSON object", http.StatusBadRequest)
		return
	}
	buf := []byte("{ \"query\": \"mutation { addImage(input: [{ objectPath: \\\"" + event["Key"].(string) + "\\\"}]) { image { id } }} \" }")
	log.Print(string(buf))
	resp, err := http.Post(*graphql_address, "application/json", bytes.NewReader(buf))
	if err != nil {
		log.Print("#######")
		log.Print(err)
		http.Error(w, "Could not post to GraphQL", http.StatusInternalServerError)
		return
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print("@@@@@@")
		log.Print(err)
		http.Error(w, "Could not parse GraphQL response", http.StatusInternalServerError)
		return
	}
	log.Print(string(body))
}

func main() {
	flag.Parse()
	http.HandleFunc("/minio_webhook", minio_webhook_handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
