package main

import (
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	_ "github.com/glebarez/go-sqlite"
)

var port = flag.Int("port", 8080, "Port to listen on")
var dbPath = flag.String("db-path", "entries.db", "Path to the sqlite file")

//go:embed index.html
var indexHtml []byte

//go:embed ok.html
var okHtml []byte

type entry struct {
	Email            string
	InstallationType string
	NumMembers       int
	Apps             []string
	PayPerMonth      float64
	PrepayFullYear   bool
	Thoughts         string
}

func getFormValues(f url.Values, name string) ([]string, error) {
	return f[name], nil
}

func getFormValue(f url.Values, name string) (string, error) {
	if ret, ok := f[name]; ok {
		switch len(ret) {
		case 0:
			return "", fmt.Errorf("%s is required", name)
		case 1:
			return ret[0], nil
		default:
			return "", fmt.Errorf("%s too many values", name)
		}
	}
	return "", fmt.Errorf("%s is required", name)
}

func getFormValueInt(f url.Values, name string) (int, error) {
	v, err := getFormValue(f, name)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(v)
}

func getFormValueBool(f url.Values, name string) (bool, error) {
	v, err := getFormValue(f, name)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

func getFormValueFloat64(f url.Values, name string) (float64, error) {
	v, err := getFormValue(f, name)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(v, 64)
}

func NewEntry(f url.Values) (*entry, error) {
	var e entry
	var err error = nil
	e.Email, err = getFormValue(f, "email")
	if err != nil {
		return nil, err
	}
	e.InstallationType, err = getFormValue(f, "installation_type")
	if err != nil {
		return nil, err
	}
	e.NumMembers, err = getFormValueInt(f, "num_members")
	if err != nil {
		return nil, err
	}
	e.Apps, err = getFormValues(f, "apps")
	if err != nil {
		return nil, err
	}
	e.PayPerMonth, err = getFormValueFloat64(f, "pay_per_month")
	if err != nil {
		return nil, err
	}
	e.PrepayFullYear, err = getFormValueBool(f, "pay_full_year")
	if err != nil {
		return nil, err
	}
	e.Thoughts, err = getFormValue(f, "thoughts")
	if err != nil {
		return nil, err
	}
	return &e, nil
}

type AddToWaitlistFn func(e *entry) error

type Server struct {
	port          int
	indexHtml     []byte
	okHtml        []byte
	addToWaitlist func(e *entry) error
}

func NewServer(port int, indexHtml, okHtml []byte, addToWaitlist AddToWaitlistFn) *Server {
	return &Server{
		port,
		indexHtml,
		okHtml,
		addToWaitlist,
	}
}

func (s *Server) Start() {
	http.HandleFunc("/waitlist", s.waitlist)
	http.HandleFunc("/", s.index)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write(s.indexHtml); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) waitlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	e, err := NewEntry(r.PostForm)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %s", err), http.StatusBadRequest)
		return
	}
	if err := s.addToWaitlist(e); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(s.okHtml))
}

func NewAddToWaitlist(db *sql.DB) AddToWaitlistFn {
	return func(e *entry) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		stm, err := tx.Prepare(`
INSERT INTO waitlist (
  email,
  installation_type,
  num_members,
  apps,
  pay_per_month,
  prepay_full_year,
  thoughts
) VALUES (
  ?, ?, ?, ?, ?, ?, ?
);`)
		if err != nil {
			return err
		}
		defer stm.Close()
		if _, err := stm.Exec(e.Email, e.InstallationType, e.NumMembers, strings.Join(e.Apps, ","), e.PayPerMonth, e.PrepayFullYear, e.Thoughts); err != nil {
			return err
		}
		return tx.Commit()
	}
}

func main() {
	flag.Parse()
	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	s := NewServer(*port, indexHtml, okHtml, NewAddToWaitlist(db))
	s.Start()
}
