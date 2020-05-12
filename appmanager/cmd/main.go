package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/golang/glog"

	app "github.com/giolekva/pcloud/appmanager"
)

var port = flag.Int("port", 1234, "Port to listen on.")
var apiAddr = flag.String("api_addr", "", "PCloud API service address.")

var helmUploadPage = `
<html>
<head>
       <title>Upload Helm chart</title>
</head>
<body>
<form enctype="multipart/form-data" action="/" method="post">
    <input type="file" name="chartfile" />
    <input type="submit" value="upload" />
</form>
</body>
</html>
`

func helmHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		_, err := io.WriteString(w, helmUploadPage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if r.Method == "POST" {
		r.ParseMultipartForm(1000000)
		file, handler, err := r.FormFile("chartfile")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		p := "/tmp/" + handler.Filename
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		_, err = io.Copy(f, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = installHelmChart(p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Installed"))
	}
}

func installHelmChart(path string) error {
	h, err := app.HelmChartFromDir("/Users/lekva/dev/go/src/github.com/giolekva/pcloud/apps/rpuppy/chart")
	if err != nil {
		return err
	}
	err = app.InstallSchema(h.Schema, *apiAddr)
	if err != nil {
		return err
	}
	glog.Infof("Installed schema: %s", h.Schema)
	err = h.Install(
		"/usr/local/bin/helm",
		map[string]string{})
	return err
}

func main() {
	flag.Parse()
	http.HandleFunc("/", helmHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))

}
