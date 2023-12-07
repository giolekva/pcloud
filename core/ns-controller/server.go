package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/giolekva/pcloud/core/ns-controller/controllers"
)

type Server struct {
	s     *http.Server
	m     *http.ServeMux
	store controllers.ZoneStoreFactory
}

func NewServer(port int, store controllers.ZoneStoreFactory) *Server {
	m := http.NewServeMux()
	s := &Server{
		s: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: m,
		},
		m:     m,
		store: store,
	}
	m.HandleFunc("/create-txt-record", s.createTxtRecord)
	m.HandleFunc("/delete-txt-record", s.deleteTxtRecord)
	m.HandleFunc("/admin/purge", s.purge)
	return s
}

func (s *Server) Start() error {
	return s.s.ListenAndServe()
}

type createTextRecordReq struct {
	Domain string `json:"domain,omitempty"`
	Entry  string `json:"entry,omitempty"`
	Text   string `json:"text,omitempty"`
}

func (s *Server) purge(w http.ResponseWriter, r *http.Request) {
	s.store.Purge()
}

func (s *Server) createTxtRecord(w http.ResponseWriter, r *http.Request) {
	var req createTextRecordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	zone, err := s.store.Get(req.Domain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := zone.AddTextRecord(req.Entry, req.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.store.Debug()
}

func (s *Server) deleteTxtRecord(w http.ResponseWriter, r *http.Request) {
	var req createTextRecordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	zone, err := s.store.Get(req.Domain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := zone.DeleteTextRecord(req.Entry, req.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.store.Debug()
}
