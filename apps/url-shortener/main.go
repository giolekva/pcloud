package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var port = flag.Int("port", 8080, "Port to listen on")
var dbPath = flag.String("db-path", "url-shortener.db", "Path to the SQLite file")

//go:embed index.html
var indexHTML embed.FS

//go:embed static/*
var f embed.FS

type NamedAddress struct {
	Name    string
	Address string
	OwnerId string
	Active  bool
}

type Store interface {
	Create(addr NamedAddress) error
	Get(name string) (NamedAddress, error)
	UpdateStatus(name string, active bool) error
	ChangeOwner(name, ownerId string) error
	List(ownerId string) ([]NamedAddress, error)
}

type NameAlreadyTaken struct {
	Name string
}

func (er NameAlreadyTaken) Error() string {
	return fmt.Sprintf("Name '%s' is already taken", er.Name)
}

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS named_addresses (
            name TEXT PRIMARY KEY,
            address TEXT,
            owner_id TEXT,
            active BOOLEAN
        )
    `)
	if err != nil {
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func generateRandomURL() string {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var urlShort string
	for i := 0; i < 6; i++ {
		urlShort += string(charset[rand.Intn(len(charset))])
	}
	return urlShort
}

func (s *SQLiteStore) Create(addr NamedAddress) error {
	_, err := s.db.Exec(`
		INSERT INTO named_addresses (name, address, owner_id, active)
		VALUES (?, ?, ?, ?)
	`, addr.Name, addr.Address, addr.OwnerId, addr.Active)
	if err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		// sqliteErr.ExtendedCode and sqlite3.ErrConstraintUnique are not the same. probably some lib error.
		// had to use actual code of unique const error
		if ok && sqliteErr.ExtendedCode() == 1555 {
			return NameAlreadyTaken{Name: addr.Name}
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) Get(name string) (NamedAddress, error) {
	row := s.db.QueryRow("SELECT name, address, owner_id, active FROM named_addresses WHERE name = ?", name)
	namedAddress := NamedAddress{}
	err := row.Scan(&namedAddress.Name, &namedAddress.Address, &namedAddress.OwnerId, &namedAddress.Active)
	if err != nil {
		if err == sql.ErrNoRows {
			return NamedAddress{}, fmt.Errorf("No record found for name %s", name)
		}
		return NamedAddress{}, err
	}
	return namedAddress, nil
}

func (s *SQLiteStore) UpdateStatus(name string, active bool) error {
	//TODO
	_, err := s.db.Exec("UPDATE named_addresses SET active = ? WHERE name = ?", active, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) ChangeOwner(name, ownerId string) error {
	//TODO
	return nil
}

func (s *SQLiteStore) List(ownerId string) ([]NamedAddress, error) {
	rows, err := s.db.Query("SELECT name, address, owner_id, active FROM named_addresses WHERE owner_id = ?", ownerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var namedAddresses []NamedAddress
	for rows.Next() {
		var namedAddress NamedAddress
		if err := rows.Scan(&namedAddress.Name, &namedAddress.Address, &namedAddress.OwnerId, &namedAddress.Active); err != nil {
			return nil, err
		}
		namedAddresses = append(namedAddresses, namedAddress)
	}
	return namedAddresses, nil
}

type PageVariables struct {
	NamedAddresses []NamedAddress
}

func renderHTML(w http.ResponseWriter, r *http.Request, tpl *template.Template, data interface{}) {
	w.Header().Set("Content-Type", "text/html")
	err := tpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getLoggedInUser(r *http.Request) (string, error) {
	if user := r.Header.Get("X-User"); user != "" {
		return user, nil
	} else {
		return "", fmt.Errorf("unauthenticated")
	}
}

type Server struct {
	store Store
}

func (s *Server) Start() {
	http.Handle("/static/", http.FileServer(http.FS(f)))
	http.HandleFunc("/", s.handler)
	http.HandleFunc("/api/update/", s.toggleHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	if r.Method == http.MethodPost {
		customName := r.PostFormValue("custom")
		address := r.PostFormValue("address")
		if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
			http.Error(w, "Address must start with http:// or https://", http.StatusBadRequest)
			return
		}
		for {
			cn := customName
			if cn == "" {
				cn = generateRandomURL()
			}
			namedAddress := NamedAddress{
				Name:    cn,
				Address: address,
				OwnerId: loggedInUser,
				Active:  true,
			}
			if err := s.store.Create(namedAddress); err == nil {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			} else if _, ok := err.(NameAlreadyTaken); ok && customName == "" {
				continue
			} else if _, ok := err.(NameAlreadyTaken); ok && customName != "" {
				http.Error(w, "Name is already taken", http.StatusBadRequest)
				return
			} else {
				http.Error(w, "Try again later", http.StatusInternalServerError)
				return
			}
		}
	}
	// Get Name from request path for redirection
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name != "" {
		namedAddress, err := s.store.Get(name)
		if err != nil {
			return
		}
		if !namedAddress.Active {
			http.Error(w, "address not found", http.StatusNotFound)
			return
		}
		http.Redirect(w, r, namedAddress.Address, http.StatusSeeOther)
		return
	}
	namedAddresses, err := s.store.List(loggedInUser)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	pageVariables := PageVariables{
		NamedAddresses: namedAddresses,
	}
	indexHtmlContent, err := indexHTML.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl, err := template.New("index").Parse(string(indexHtmlContent))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	renderHTML(w, r, tmpl, pageVariables)
}

type UpdateRequest struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

func (s *Server) toggleHandler(w http.ResponseWriter, r *http.Request) {
	var data UpdateRequest
	if r.Method == http.MethodPost {
		loggedInUser, err := getLoggedInUser(r)
		if err != nil {
			http.Error(w, "User Not Logged In", http.StatusUnauthorized)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Failed to decode JSON data", http.StatusBadRequest)
			return
		}
		namedAddress, err := s.store.Get(data.Name)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get named_address for name %s", data.Name), http.StatusInternalServerError)
			return
		}
		if namedAddress.OwnerId != loggedInUser {
			http.Error(w, "Invalid owner ID", http.StatusUnauthorized)
			return
		}
		if err := s.store.UpdateStatus(data.Name, data.Active); err != nil {
			http.Error(w, fmt.Sprintf("Failed to update status for name %s", data.Name), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func main() {
	flag.Parse()
	db, err := NewSQLiteStore(*dbPath)
	if err != nil {
		panic(err)
	}
	s := Server{store: db}
	s.Start()
}
