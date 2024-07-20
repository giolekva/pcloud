package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var port = flag.Int("port", 8080, "Port to listen on")

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from Dodo App!")
}

func main() {
	flag.Parse()
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
