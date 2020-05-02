package importer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

type OkHandler struct {
}

func (h *OkHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}

type ErrorHandler struct {
}

func (h *ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "", http.StatusBadRequest)
}

func TestValidEvent(t *testing.T) {
	q, err := EventToQuery(`{"Key": "foo/bar"}`)
	if err != nil {
		t.Fatal(err)
	}
	expected := `mutation {
  addImage(input: [{ objectPath: "foo/bar" }]) {
    image {
      id
    }
  }
}`
	if q.Query != expected {
		t.Fatal(q.Query)
	}
}

func TestValidEventEscaping(t *testing.T) {
	q, err := EventToQuery(`{"Key": "foo\"bar"}`)
	if err != nil {
		t.Fatal(err)
	}
	expected := `mutation {
  addImage(input: [{ objectPath: "foo\"bar" }]) {
    image {
      id
    }
  }
}`
	if q.Query != expected {
		t.Fatal(q.Query)
	}
}

func TestNoKey(t *testing.T) {
	_, err := EventToQuery(`{"foo": "bar"}`)
	if err == nil {
		t.Fatal("Got key")
	}
}

func TestInvalidKey(t *testing.T) {
	_, err := EventToQuery(`{"foo": 123}`)
	if err == nil {
		t.Fatal("Got key")
	}
}

func TestInvalidKeyComplex(t *testing.T) {
	_, err := EventToQuery(`{"foo": {"bar": 5}}`)
	if err == nil {
		t.Fatal("Got key")
	}
}

func TestHandlerOk(t *testing.T) {
	mockApi := httptest.NewServer(&OkHandler{})
	r, err := http.NewRequest("GET", "/foo", strings.NewReader(`{"Key": "foo/bar"}`))
	checkErr(err, t)
	rec := httptest.NewRecorder()
	(&Handler{mockApi.URL}).ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
}

func TestHandlerInvalidEvent(t *testing.T) {
	mockApi := httptest.NewServer(&OkHandler{})
	r, err := http.NewRequest("GET", "/foo", strings.NewReader(`{"Key": 123}`))
	checkErr(err, t)
	rec := httptest.NewRecorder()
	(&Handler{mockApi.URL}).ServeHTTP(rec, r)
	if rec.Code != http.StatusBadRequest {
		t.Fatal(rec.Code)
	}
}

func TestHandlerError(t *testing.T) {
	mockApi := httptest.NewServer(&ErrorHandler{})
	r, err := http.NewRequest("GET", "/foo", strings.NewReader(`{"Key": "foo/bar"}`))
	checkErr(err, t)
	rec := httptest.NewRecorder()
	(&Handler{mockApi.URL}).ServeHTTP(rec, r)
	if rec.Code == http.StatusOK {
		t.Fatal(rec.Code)
	}
}
