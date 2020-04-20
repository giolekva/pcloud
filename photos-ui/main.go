package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var port = flag.Int("port", 3000, "Port to listen on")

func handle_gallery(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./gallery.html")
}

func handle_photo(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./photo.html")
}

func handle_graphql(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://localhost:8080/graphql?query={queryImage(){id objectPath}}", http.StatusMovedPermanently)
}

func main() {
	flag.Parse()
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/graphql", handle_graphql)
	http.HandleFunc("/", handle_gallery)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
