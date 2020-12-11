package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"text/template"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var port = flag.Int("port", 3000, "Port to listen on.")
var pcloudApiAddr = flag.String("pcloud_api_addr", "", "PCloud API Server address.")

func handle_gallery(w http.ResponseWriter, r *http.Request) {
	gallery_html, err := bazel.Runfile("apps/photos-ui/gallery.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	http.ServeFile(w, r, gallery_html)
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
	photo_html, err := bazel.Runfile("apps/photos-ui/photo.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	t, err := template.ParseFiles(photo_html)
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

func newGqlProxy(pcloudApiAddr string) *httputil.ReverseProxy {
	u, err := url.Parse(pcloudApiAddr)
	if err != nil {
		panic(err)
	}
	return httputil.NewSingleHostReverseProxy(u)
}

func main() {
	flag.Parse()
	static_dir, err := bazel.Runfile("apps/photos-ui/static")
	if err != nil {
		log.Fatal(err)
	}
	fs := http.FileServer(http.Dir(static_dir))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/graphql", newGqlProxy(*pcloudApiAddr))
	http.HandleFunc("/photo", handle_photo)
	http.HandleFunc("/", handle_gallery)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
