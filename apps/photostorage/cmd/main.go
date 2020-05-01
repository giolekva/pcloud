package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/giolekva/pcloud/apps/photostorage"
)

var port = flag.Int("port", 123, "Port to listen on.")
var apiAddr = flag.String("api_addr", "http://localhost/graphql", "PCloud GraphQL API server address.")

func main() {
	flag.Parse()
	http.Handle("/new_object", &photostorage.Handler{*apiAddr})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
