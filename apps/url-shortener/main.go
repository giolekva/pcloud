package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var dbPath = flag.String("db-path", "url-shortener.db", "Path to the SQLite file")

type NamedAddress struct {
	// Name of the address. Must be unique across the service.
	Name    string
	Address string
	OwnerId string
	Active  bool
}

type Store interface {
	// Creates new named address.
	Create(name, address, ownerId string) error
	// Activates given named address. Does nothing if named address is already active.
	Activate(name string) error
	// Deactivates given named address. Does nothing if named address is already inactive.
	Deactivate(name string) error
	// Transfers ownership of the given named address to new owner.
	ChangeOwner(name, ownerId string) error
	// Retreives all named addresses owned by given owner.
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
	return nil, nil
}

func newEntryHandler(w http.ResponseWriter, r *http.Request) {
	var namedAddress NamedAddress
	db, err := openDatabase()
	if err != nil {
		fmt.Println("Error opening database:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	if r.Method == "POST" {
		store := &SQLiteStore{db: db}
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

	fmt.Println("BODY: ", r.Body)
	// fmt.Println("REQUEST CHECK")
}

func main() {
	flag.Parse()

	// check if database file exists
	_, err := os.Stat(*dbPath)
	if os.IsNotExist(err) {
		// if not, create it and initialize the table
		db, err := openDatabase()
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
		// other errors handler to check file existance
		fmt.Println("Error checking database file:", err)
		return
	}

	fmt.Println("TEST")

	http.HandleFunc("/", newEntryHandler)
	// http.HandleFunc("/", activate)
	http.ListenAndServe(":8080", nil)
}
