package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/itaysk/regogo"
)

var port = flag.Int("port", 3000, "Port to listen on")

func handle(w http.ResponseWriter, r *http.Request) {
	log.Printf("---------- %s", r.RemoteAddr)
	resp, err := http.Get("https://dog.ceo/api/breeds/image/random")
	if err != nil {
		log.Print(err)
		return
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return
	}
	imgPath, err := regogo.Get(string(respBody), "input.message")
	if err != nil {
		log.Print(err)
		return
	}
	w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
    <head>
        <title>Photos</title>
    </head>
    <script src="static/photos.js"></script>
    <body>
      <img src="%s"></img>
    </body>
</html>`, imgPath.String())))
}

func main() {
	flag.Parse()
	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
