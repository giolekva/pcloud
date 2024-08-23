package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Server struct {
	s          *http.Server
	m          *http.ServeMux
	store      RecordStore
	zone       string
	ds         string
	nameserver []string
}

func NewServer(port int, zone string, ds string, store RecordStore, nameserver []string) *Server {
	m := http.NewServeMux()
	s := &Server{
		s: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: m,
		},
		m:          m,
		store:      store,
		zone:       zone,
		ds:         ds,
		nameserver: nameserver,
	}
	m.HandleFunc("/records-to-publish", s.recordsToPublish)
	m.HandleFunc("/create-txt-record", s.createTxtRecord)
	m.HandleFunc("/create-a-record", s.createARecord)
	m.HandleFunc("/delete-a-record", s.deleteARecord)
	m.HandleFunc("/delete-txt-record", s.deleteTxtRecord)
	return s
}

func (s *Server) Start() error {
	return s.s.ListenAndServe()
}

type record struct {
	Domain string `json:"domain,omitempty"`
	Entry  string `json:"entry,omitempty"`
	Text   string `json:"text,omitempty"`
}

func (s *Server) recordsToPublish(w http.ResponseWriter, r *http.Request) {
	subdomain := strings.Split(s.zone, ".")[0]
	if _, err := fmt.Fprintln(w, s.ds); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i, ip := range s.nameserver {
		if _, err := fmt.Fprintf(w, "ns%d.%s. 10800 IN A %s\n", i+1, s.zone, ip); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := fmt.Fprintf(w, "%s 10800 IN NS ns%d.%s.\n", subdomain, i+1, s.zone); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) createTxtRecord(w http.ResponseWriter, r *http.Request) {
	var req record
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.Add(req.Entry, req.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) createARecord(w http.ResponseWriter, r *http.Request) {
	var req record
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.AddARecord(req.Entry, req.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) deleteTxtRecord(w http.ResponseWriter, r *http.Request) {
	var req record
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.Delete(req.Entry, req.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) deleteARecord(w http.ResponseWriter, r *http.Request) {
	var req record
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteARecord(req.Entry, req.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
