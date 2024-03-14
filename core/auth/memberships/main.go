package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var port = flag.Int("port", 8080, "ort to listen on")
var dbPath = flag.String("db-path", "memberships.db", "Path to SQLite file")

//go:embed index.html
var indexHTML embed.FS

//go:embed static
var f embed.FS

type SQLiteStore struct {
	db *sql.DB
}

type Store interface {
	CreateGroup(loggedInUsername string, group Group) error
	GetOwnerGroups(username string) ([]Group, error)
	GetMembershipGroups(username string) ([]Group, error)
	AddUserIntoGroup(loggedInUsername, targetUsername, groupName string) error
}

type Server struct {
	store Store
}

type Group struct {
	Name        string
	Description string
}

type Member struct {
	Username string
	Groups   []Group
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS groups (
            name TEXT PRIMARY KEY,
            description TEXT
        );

        CREATE TABLE IF NOT EXISTS owners (
            username TEXT,
            group_name TEXT,
            FOREIGN KEY(group_name) REFERENCES groups(name)
        );

        CREATE TABLE IF NOT EXISTS group_to_group (
            parent_group TEXT,
            child_group TEXT,
            FOREIGN KEY(parent_group) REFERENCES groups(name),
            FOREIGN KEY(child_group) REFERENCES groups(name)
        );

        CREATE TABLE IF NOT EXISTS user_to_group (
            username TEXT,
            group_name TEXT,
            FOREIGN KEY(group_name) REFERENCES groups(name)
        );
    `)
	if err != nil {
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) queryGroups(query string, args ...interface{}) ([]Group, error) {
	groups := make([]Group, 0)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var group Group
		if err := rows.Scan(&group.Name, &group.Description); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *SQLiteStore) GetOwnerGroups(username string) ([]Group, error) {
	query := `
        SELECT groups.name, groups.description 
        FROM groups 
        JOIN owners ON groups.name = owners.group_name 
        WHERE owners.username = ?
    `
	return s.queryGroups(query, username)
}

func (s *SQLiteStore) GetMembershipGroups(username string) ([]Group, error) {
	query := `
        SELECT groups.name, groups.description 
        FROM groups 
        JOIN user_to_group ON groups.name = user_to_group.group_name 
        WHERE user_to_group.username = ?
    `
	return s.queryGroups(query, username)
}

func (s *SQLiteStore) CreateGroup(username string, group Group) error {
	query := `INSERT INTO groups (name, description) VALUES (?, ?)`
	_, err := s.db.Exec(query, group.Name, group.Description)
	if err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		if ok && sqliteErr.Code == sqlite3.ErrConstraintUnique {
			return fmt.Errorf("Group with the name %s already exists", group.Name)
		}
		return err
	}
	query = `INSERT INTO owners (username, group_name) VALUES (?, ?)`
	_, err = s.db.Exec(query, username, group.Name)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) AddUserIntoGroup(loggedInUsername, targetUsername, groupName string) error {
	// TODO(dtabidze): should check if username u adding exists(ask ORY)
	ownerGroups, err := s.GetOwnerGroups(loggedInUsername)
	if err != nil {
		return err
	}
	var isOwner bool
	for _, group := range ownerGroups {
		if group.Name == groupName {
			isOwner = true
			break
		}
	}
	if !isOwner {
		// or You don't have permission, You are not owner of this group.
		return fmt.Errorf("%s is not the owner of group %s", loggedInUsername, groupName)
	}
	membershipGroups, err := s.GetMembershipGroups(targetUsername)
	if err != nil {
		return err
	}

	for _, group := range membershipGroups {
		if group.Name == groupName {
			return fmt.Errorf("%s is already a member of group %s", targetUsername, groupName)
		}
	}
	query := `INSERT INTO user_to_group (username, group_name) VALUES (?, ?)`
	if _, err := s.db.Exec(query, targetUsername, groupName); err != nil {
		return err
	}
	return nil
}

func getLoggedInUser(r *http.Request) (string, error) {
	// TODO(dtabidze): should make a request to get loggedin user
	return "tabo", nil
}

func (s *Server) Start() {
	http.Handle("/static/", http.FileServer(http.FS(f)))
	http.HandleFunc("/", s.handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

type GroupData struct {
	Group      Group
	Membership string
}

func renderHTML(w http.ResponseWriter, r *http.Request, tmpl *template.Template, data interface{}) {
	w.Header().Set("Content-Type", "text/html")
	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var group Group
		group.Name = r.PostFormValue("group-name")
		group.Description = r.PostFormValue("description")
		if err := s.store.CreateGroup(loggedInUser, group); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	var data []GroupData
	ownerGroups, err := s.store.GetOwnerGroups(loggedInUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	membershipGroups, err := s.store.GetMembershipGroups(loggedInUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, group := range ownerGroups {
		data = append(data, GroupData{Group: group, Membership: "Owner"})
	}
	for _, group := range membershipGroups {
		data = append(data, GroupData{Group: group, Membership: "Member"})
	}
	indexHTMLContent, err := indexHTML.ReadFile("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl, err := template.New("index").Parse(string(indexHTMLContent))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderHTML(w, r, tmpl, data)
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
