package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type APIServer struct {
	r          *mux.Router
	serv       *http.Server
	kratosAddr string
}

type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewAPIServer(port int, kratosAddr string) *APIServer {
	r := mux.NewRouter()
	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	return &APIServer{r, serv, kratosAddr}
}

func (s *APIServer) Start() error {
	s.r.Path("/identities").Methods(http.MethodPost).HandlerFunc(s.identityCreate)
	return s.serv.ListenAndServe()
}

const identityCreateTmpl = `
{
  "credentials": {
    "password": {
      "config": {
        "password": "%s"
      }
    }
  },
  "schema_id": "user",
  "state": "active",
  "traits": {
    "username": "%s"
  }
}
`

type identityCreateReq struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func extractErrorMessage(errResp ErrorResponse) string {
	switch errResp.Error.Status {
	case "Conflict":
		return "Username is not available."
	case "Bad Request":
		return "Username is less than 3 characters."
	default:
		return "Unexpected error."
	}
}

func (s *APIServer) identityCreate(w http.ResponseWriter, r *http.Request) {
	var req identityCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "request can not be parsed", http.StatusBadRequest)
		return
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, identityCreateTmpl, req.Password, req.Username)
	resp, err := http.Post(s.identitiesEndpoint(), "application/json", &buf)
	if err != nil {
		http.Error(w, "failed", http.StatusInternalServerError)
		return
	}
	fmt.Println("Status Code:", resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		var e ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			fmt.Printf("%+v\n", e)
			http.Error(w, "failed to decode", http.StatusInternalServerError)
			return
		}
		errorMessage := extractErrorMessage(e)
		http.Error(w, errorMessage, http.StatusInternalServerError)
		return
	}
}

func (s *APIServer) identitiesEndpoint() string {
	return fmt.Sprintf("%s/admin/identities", s.kratosAddr)
}
