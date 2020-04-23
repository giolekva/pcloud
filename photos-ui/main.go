package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"text/template"
)

var port = flag.Int("port", 3000, "Port to listen on.")
var pcloudApiServer = flag.String("pcloud_api_server", "", "PCloud API Server address.")

func handle_gallery(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./gallery.html")
}

func handle_photo(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Could not read query", http.StatusInternalServerError)
		return
	}
	id, ok := r.Form["id"]
	if !ok {
		http.Error(w, "Photo id must be provided", http.StatusBadRequest)
		return
	}
	t, err := template.ParseFiles("photo.html")
	if err != nil {
		log.Print(err)
		http.Error(w, "Could not process page", http.StatusInternalServerError)
		return
	}
	err = t.Execute(w, struct{ Id string }{id[0]})
	if err != nil {
		log.Print(err)
		http.Error(w, "Could not process page", http.StatusInternalServerError)
		return
	}
}

func newGqlProxy(pcloudApiServer string) *httputil.ReverseProxy {
	u, err := url.Parse(pcloudApiServer)
	if err != nil {
		panic(err)
	}
	return httputil.NewSingleHostReverseProxy(u)
}

func main() {
	flag.Parse()
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/graphql", newGqlProxy(*pcloudApiServer))
	http.HandleFunc("/photo", handle_photo)
	http.HandleFunc("/", handle_gallery)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
