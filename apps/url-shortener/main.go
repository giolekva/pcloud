package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var dbPath = flag.String("db-path", "url-shortener.db", "Path to the SQLite file")
var db *sql.DB

//go:embed index.html
var content embed.FS

type NamedAddress struct {
	Name    string
	Address string
	OwnerId string
	Active  bool
}

type Store interface {
	Create(name, address, ownerId string) error
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
	return db, nil
}

func createTable(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS named_addresses (
            Name TEXT PRIMARY KEY,
            Address TEXT,
            OwnerId TEXT,
            Active BOOLEAN DEFAULT true
        )
    `)
	return err
}

func generateRandomURL(store Store) (string, error) {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var urlShort string

	for {
		// Generate a random URL
		for i := 0; i < 6; i++ {
			urlShort += string(charset[rand.Intn(len(charset))])
		}

		// Check if the generated URL exists in the DB
		var exists bool
		err := store.(*SQLiteStore).db.QueryRow("SELECT EXISTS (SELECT 1 FROM named_addresses WHERE Name = ?)", urlShort).Scan(&exists)
		if err != nil {
			return "", err
		}

		// If not, break the loop
		if !exists {
			break
		}

		// If it exists, reset the URL and generate a new one
		urlShort = ""
	}

	return urlShort, nil
}

func (s *SQLiteStore) Create(name, address, ownerId string) error {
	_, err := s.db.Exec(`
		INSERT INTO named_addresses (Name, Address, OwnerId, Active)
		VALUES (?, ?, ?, ?)
	`, name, address, ownerId, true)
	return err
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
	rows, err := s.db.Query("SELECT Name, Address FROM named_addresses WHERE OwnerId = ?", ownerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var namedAddresses []NamedAddress
	for rows.Next() {
		var namedAddress NamedAddress
		if err := rows.Scan(&namedAddress.Name, &namedAddress.Address); err != nil {
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

func newEntryHandler(w http.ResponseWriter, r *http.Request) {
	db, err := openDatabase()
	if err != nil {
		fmt.Println("Error opening database:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	store := &SQLiteStore{db: db}

	// Check if request is POST
	if r.Method == http.MethodPost {
		var namedAddress NamedAddress
		namedAddress.Active = true

		// Check if a custom name is provided in POST request
		customName := r.PostFormValue("custom")
		if customName != "" {
			// Check in DB if custom Name exists
			var exists bool
			err := store.db.QueryRow("SELECT EXISTS (SELECT 1 FROM named_addresses WHERE Name = ?)", customName).Scan(&exists)
			if err != nil {
				fmt.Println("Error custom Name:", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if exists {
				// Custom Name already exists
				http.Error(w, "Name already exists", http.StatusBadRequest)
				return
			}

			namedAddress.Name = customName
		} else {
			// Generate a random URL if no custom name
			shortURL, err := generateRandomURL(store)
			if err != nil {
				fmt.Println("Error generating random URL:", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			namedAddress.Name = shortURL
		}

		namedAddress.OwnerId = "tabo"
		namedAddress.Address = r.PostFormValue("address")

		// Create named address in the DB
		err := store.Create(namedAddress.Name, namedAddress.Address, namedAddress.OwnerId)
		if err != nil {
			fmt.Println("Error creating named address:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		fmt.Println("Named address created:", namedAddress)
	}

	// Retrieve named addresses for the owner
	namedAddresses, err := store.List("tabo")
	if err != nil {
		fmt.Println("Error retrieving named addresses:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Combine data for rendering
	pageVariables := PageVariables{
		NamedAddresses: namedAddresses,
	}

	// Get Name from request path for redirection
	name := strings.TrimPrefix(r.URL.Path, "/")
	fmt.Println("redirection URL: ", r.URL.Path)
	if name != "" {
		// get coresponding Address for Name
		var address string
		err := store.db.QueryRow("SELECT Address FROM named_addresses WHERE Name = ?", name).Scan(&address)
		if err != nil {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}
		// Check if Address has https at the begining
		if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
			address = "http://" + address
		}
		// Redirect to the address
		http.Redirect(w, r, address, http.StatusFound)
		return
	}

	// Read the embedded HTML content
	indexHtmlContent, err := content.ReadFile("index.html")
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

	// check if database file exists
	_, err := os.Stat(*dbPath)
	if os.IsNotExist(err) {
		// if not, create it and initialize the table
		db, err = openDatabase()
		if err != nil {
			fmt.Println("Error opening database:", err)
			return
		}
		defer db.Close()

		err = createTable(db)
		if err != nil {
			fmt.Println("Error creating table:", err)
			return
		}

		fmt.Println("SQLite database and table created successfully!")
	} else if err != nil {
		fmt.Println("Error checking database file:", err)
		return
	}

	fmt.Println("Server listening on :8080")
	http.HandleFunc("/", newEntryHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
