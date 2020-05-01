package photostorage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"github.com/itaysk/regogo"
)

var jsonContentType = "application/json"

var addImgTmpl = `mutation {
  addImage(input: [{ objectPath: %s }]) {
    image {
      id
    }
  }
}`

type Query struct {
	Query string
}

func EventToQuery(event string) (Query, error) {
	key, err := regogo.Get(event, "input.Key")
	if err != nil {
		return Query{}, err
	}
	keyStr := key.String()
	if keyStr == "" {
		return Query{}, errors.New("Key not found")
	}
	objectPath, err := json.Marshal(key.String())
	if err != nil {
		return Query{}, err
	}
	return Query{fmt.Sprintf(addImgTmpl, objectPath)}, nil
}

type Handler struct {
	ApiAddr string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Error(err)
		http.Error(w, "Could not read HTTP request body", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		// Just a health check from Minio
		return
	}
	bodyStr := string(body)
	glog.Infof("Received event from Minio: %s", bodyStr)
	query, err := EventToQuery(bodyStr)
	if err != nil {
		glog.Error(err)
		http.Error(w, "INTERNAL", http.StatusBadRequest)
		return
	}
	glog.Info(query)
	queryJson, err := json.Marshal(query)
	if err != nil {
		panic(err)
	}
	resp, err := http.Post(h.ApiAddr, jsonContentType, bytes.NewReader(queryJson))
	if err != nil {
		glog.Error(err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusOK {
		glog.Error(resp.StatusCode)
		http.Error(w, "Query failed", resp.StatusCode)
		return
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	glog.Info(string(respBody))
}
