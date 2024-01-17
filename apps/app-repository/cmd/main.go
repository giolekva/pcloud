package main

import (
	"flag"
	"log"
	"os"

	"github.com/giolekva/pcloud/apps/apprepo"
)

var port = flag.Int("port", 8080, "Port to listen on")
var appsDir = flag.String("apps-dir", "./apps", "Directory listing application archives")
var schemeWithHost = flag.String("scheme-with-host", "http://localhost:8080", "")

func main() {
	flag.Parse()
	l := apprepo.NewFSLoader(os.DirFS(*appsDir))
	apps, err := l.Load()
	if err != nil {
		log.Fatal(err)
	}
	s := apprepo.NewServer(*schemeWithHost, *port, apps)
	log.Fatal(s.Start())
}
