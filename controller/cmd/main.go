package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/golang/glog"
	"github.com/itaysk/regogo"
)

var port = flag.Int("port", 123, "Port to listen on.")
var apiAddr = flag.String("api_addr", "", "PCloud GraphQL API server address.")

var jsonContentType = "application/json"

var addImgTmpl = `
mutation {
  addImage(input: [%s]) {
    image {
      id
    }
  }
}`

type image struct {
	ObjectPath string
}

type query struct {
	Query string
}

func minioHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Error(err)
		http.Error(w, "Could not read HTTP request body", http.StatusInternalServerError)
		return
	}
	if len(body) == 0 {
		// Just a health check from Minio
		return
	}
	bodyStr := string(body)
	glog.Infof("Received event from Minio: %s", bodyStr)
	key, err := regogo.Get(bodyStr, "input.Key")
	if err != nil {
		glog.Error(err)
		http.Error(w, "Could not find object key", http.StatusBadRequest)
		return
	}
	img := image{key.String()}
	imgJson, err := json.Marshal(img)
	if err != nil {
		panic(err)
	}
	q := query{fmt.Sprintf(addImgTmpl, imgJson)}
	glog.Info(q)
	queryJson, err := json.Marshal(q)
	if err != nil {
		panic(err)
	}
	resp, err := http.Post(
		*apiAddr,
		jsonContentType,
		bytes.NewReader(queryJson))
	if err != nil {
		glog.Error(err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	glog.Info(string(respBody))
}

func main() {
	flag.Parse()

	http.HandleFunc("/minio_webhook", minioHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
