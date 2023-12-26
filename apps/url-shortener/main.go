package main

import (
	"database/sql"
	"embed"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"github.com/mattn/go-sqlite3"
)

var dbPath = flag.String("db-path", "url-shortener.db", "Path to the SQLite file")

//go:embed index.html
var indexHTML embed.FS

type NamedAddress struct {
	Name    string
	Address string
	OwnerId string
	Active  bool
}

type Store interface {
	Create(addr NamedAddress) error
	Get(name string) (NamedAddress, error)
	Activate(name string) error
	Deactivate(name string) error
	ChangeOwner(name, ownerId string) error
	List(ownerId string) ([]NamedAddress, error)
}

type SQLiteStore struct {
	db *sql.DB
}

func openDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS named_addresses (
            name TEXT PRIMARY KEY,
            address TEXT,
            ownerId TEXT,
            active BOOLEAN
        )
    `)
	if err != nil {
		return nil, err
	}

	return db, nil
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
	if !strings.HasPrefix(addr.Address, "http://") && !strings.HasPrefix(addr.Address, "https://") {
		return errors.New("Address must start with http:// or https://")
	}
	_, err := s.db.Exec(`
		INSERT INTO named_addresses (name, address, ownerId, active)
		VALUES (?, ?, ?, ?)
	`, addr.Name, addr.Address, addr.OwnerId, addr.Active)
	return err
}

func (s *SQLiteStore) Get(name string) (NamedAddress, error) {
	row := s.db.QueryRow("SELECT name, address, ownerID, active FROM named_addresses WHERE name = ?", name)
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

func (s *SQLiteStore) Activate(name string) error {
	return nil
}

func (s *SQLiteStore) Deactivate(name string) error {
	return nil
}

func (s *SQLiteStore) ChangeOwner(name, ownerId string) error {
	return nil
}

func (s *SQLiteStore) List(ownerId string) ([]NamedAddress, error) {
	rows, err := s.db.Query("SELECT name, address, ownerId, active FROM named_addresses WHERE ownerId = ?", ownerId)
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

type Server struct {
	store Store
}

func (s *Server) Start() {
	http.HandleFunc("/", s.handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	// Check if request is POST
	if r.Method == http.MethodPost {
		customName := r.PostFormValue("custom")
		address := r.PostFormValue("address")
		for {
			cn := customName
			if cn == "" {
				cn = generateRandomURL()
			}
			// check if custom exists
			namedAddress := NamedAddress{
				Name:    cn,
				Address: address,
				OwnerId: "tabo",
				Active:  true,
			}
			if err := s.store.Create(namedAddress); err == nil {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			} else if !errors.Is(err, sqlite3.ErrConstraintUnique) {
				http.Error(w, "try again later", http.StatusInternalServerError)
				return
			} else if customName != "" {
				// error
			}
		}
	}

	// Get Name from request path for redirection
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name != "" {
		namedAddress, err := s.store.Get(name)
		if err != nil {
			// TODO
			return
		}
		fmt.Println("redirection URL: ", namedAddress.Address)
		// Redirect to the address
		http.Redirect(w, r, namedAddress.Address, http.StatusSeeOther)
		return
	}

	// Retrieve named addresses for the owner
	namedAddresses, err := s.store.List("tabo")
	if err != nil {
		fmt.Println("Error retrieving named addresses:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Combine data for rendering
	pageVariables := PageVariables{
		NamedAddresses: namedAddresses,
	}

	// Read the embedded HTML content
	indexHtmlContent, err := indexHTML.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Parse the HTML content
	tmpl, err := template.New("index").Parse(string(indexHtmlContent))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Render the HTML table
	renderHTML(w, r, tmpl, pageVariables)
}

func main() {
	flag.Parse()
	db, err := openDatabase()
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}
	// createTable(db)
	s := Server{&SQLiteStore{db}}
	s.Start()
}
