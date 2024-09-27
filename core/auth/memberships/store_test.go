package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestInitSuccess(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Init("admin", "admin@admin", []string{"admin", "all"}); err != nil {
		t.Fatal(err)
	}
	groups, err := store.GetGroupsOwnedBy("admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("Expected two groups, got: %s", groups)
	}
	groups, err = store.GetGroupsUserBelongsTo("admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("Expected two groups, got: %s", groups)
	}
}

func TestInitFailure(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
        INSERT INTO groups (name, description)
        VALUES
            ('a', 'xxx'),
            ('b', 'yyy');
        `)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Init("admin", "admin", []string{"admin", "all"})
	if err == nil {
		t.Fatal("initialisation did not fail")
	} else if err.Error() != "Store already initialised" {
		t.Fatalf("Expected initialisation error, got: %s", err.Error())
	}
}

func TestGetAllTransitiveGroupsForGroup(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
        INSERT INTO groups (name, description)
        VALUES
            ('a', 'xxx'),
            ('b', 'yyy');

        INSERT INTO group_to_group (child_group, parent_group)
        VALUES
            ('a', 'b'),
            ('b', 'a');
        `)
	if err != nil {
		t.Fatal(err)
	}
	groups, err := store.GetAllTransitiveGroupsForGroup("a")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("Expected exactly one transitive group, got: %s", groups)
	}
	expected := Group{"b", "yyy"}
	if groups[0] != expected {
		t.Fatalf("Expected %s, got: %s", expected, groups[0])
	}
}

func TestGetAllTransitiveGroupsForUser(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
        INSERT INTO groups (name, description)
        VALUES
            ('a', 'xxx'),
            ('b', 'yyy'),
            ('c', 'zzz');

        INSERT INTO group_to_group (child_group, parent_group)
        VALUES
            ('a', 'c'),
            ('b', 'c');
        INSERT INTO user_to_group (username, group_name)
        VALUES
            ('u', 'a'),
            ('u', 'b');
        `)
	if err != nil {
		t.Fatal(err)
	}
	groups, err := store.GetAllTransitiveGroupsForUser("u")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 3 {
		t.Fatalf("Expected exactly one transitive group, got: %s", groups)
	}
}

func TestParentAndChildGroupCases(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
	INSERT INTO groups (name, description)
	VALUES
		('a', 'xxx'),
		('b', 'yyy');
	`)
	if err != nil {
		t.Fatal(err)
	}
	err = store.AddChildGroup("a", "a")
	if err == nil || err.Error() != "Parent and child groups can not have same name" {
		t.Fatalf("Expected error, got: %v", err)
	}
	if err := store.AddChildGroup("a", "b"); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestRemoveChildGroupHandler(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS groups (
			name TEXT PRIMARY KEY,
			description TEXT
		);
		INSERT INTO groups (name, description)
        VALUES
            ('bb', 'desc'),
			('aa', 'desc');
		CREATE TABLE IF NOT EXISTS owners (
			username TEXT,
			group_name TEXT,
			FOREIGN KEY(group_name) REFERENCES groups(name),
			UNIQUE (username, group_name)
		);
		INSERT INTO owners (username, group_name)
        VALUES
            ('testuser', 'bb');
		CREATE TABLE IF NOT EXISTS group_to_group (
			parent_group TEXT,
			child_group TEXT
		);
        INSERT INTO group_to_group (parent_group, child_group)
        VALUES
            ('bb', 'aa');
        `)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{
		store:         store,
		syncAddresses: make(map[string]struct{}),
		mu:            sync.Mutex{},
	}
	router := mux.NewRouter()
	router.HandleFunc("/group/{parent-group}/remove-child-group/{child-group}", server.removeChildGroupHandler).Methods(http.MethodPost)
	req, err := http.NewRequest("POST", "/group/bb/remove-child-group/aa", nil)
	req.Header.Set("X-Forwarded-User", "testuser")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	if rr.Header().Get("Location") != "/group/bb" {
		t.Errorf("handler returned wrong Location header: got %v want %v", rr.Header().Get("Location"), "/group/bb")
	}
}

func TestFilterUsersByGroupHandler(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS groups (
			name TEXT PRIMARY KEY,
			description TEXT
		);
		INSERT INTO groups (name, description)
        VALUES
            ('a', 'a'),
			('b', 'b'),
			('c', 'c'),
			('d', 'd'),
			('e', 'e'),
			('f', 'f');
		CREATE TABLE IF NOT EXISTS owners (
			username TEXT,
			group_name TEXT,
			FOREIGN KEY(group_name) REFERENCES groups(name),
			UNIQUE (username, group_name)
		);
		INSERT INTO owners (username, group_name)
        VALUES
            ('testuser1', 'a'),
			('testuser2', 'd');
		CREATE TABLE IF NOT EXISTS group_to_group (
			parent_group TEXT,
			child_group TEXT
		);
        INSERT INTO group_to_group (parent_group, child_group)
        VALUES
            ('a', 'b'),
			('b', 'c'),
			('d', 'e'),
			('e', 'f');
		CREATE TABLE IF NOT EXISTS user_to_group (
			username TEXT,
			group_name TEXT,
			FOREIGN KEY(group_name) REFERENCES groups(name),
			UNIQUE (username, group_name)
		);
        INSERT INTO user_to_group (username, group_name)
        VALUES
            ('u1', 'a'),
			('u2', 'b'),
			('u3', 'e'),
			('u4', 'f'),
			('u5', 'f'),
			('u6', 'd'),
			('u7', 'd');
		CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			email TEXT,
			UNIQUE (email)
		);
		INSERT INTO users (username, email)
		VALUES
			('u1','u1@d.d'),
			('u2','u2@d.d'),
			('u3','u3@d.d'),
			('u4','u4@d.d'),
			('u5','u5@d.d'),
			('u6','u6@d.d'),
			('u7','u7@d.d');
		CREATE TABLE IF NOT EXISTS user_ssh_keys (
			username TEXT,
			ssh_key TEXT,
			UNIQUE (ssh_key),
			FOREIGN KEY(username) REFERENCES users(username)
		);
		INSERT INTO user_ssh_keys (username, ssh_key)
		VALUES
			('u1','ssh1'),
			('u1','ssh1-1'),
			('u2','ssh2'),
			('u3','ssh3'),
			('u4','ssh4'),
			('u5','ssh5'),
			('u6','ssh6'),
			('u7','ssh7');
        `)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{
		store:         store,
		syncAddresses: make(map[string]struct{}),
		mu:            sync.Mutex{},
	}
	router := mux.NewRouter()
	// case when group present or exist
	router.HandleFunc("/api/users", server.apiGetAllUsers).Methods(http.MethodGet)
	req, err := http.NewRequest("GET", "/api/users?groups=b,e,t", nil)
	req.Header.Set("X-Forwarded-User", "testuser1")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	expected := []User{
		{"u1", "u1@d.d", []string{"ssh1", "ssh1-1"}},
		{"u2", "u2@d.d", []string{"ssh2"}},
		{"u3", "u3@d.d", []string{"ssh3"}},
		{"u6", "u6@d.d", []string{"ssh6"}},
		{"u7", "u7@d.d", []string{"ssh7"}},
	}

	var actual []User
	err = json.NewDecoder(rr.Body).Decode(&actual)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", actual, expected)
	}

	// case when no group present
	req, err = http.NewRequest("GET", "/api/users?groups=", nil)
	req.Header.Set("X-Forwarded-User", "testuser1")
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	expected = []User{
		{"u1", "u1@d.d", []string{"ssh1", "ssh1-1"}},
		{"u2", "u2@d.d", []string{"ssh2"}},
		{"u3", "u3@d.d", []string{"ssh3"}},
		{"u4", "u4@d.d", []string{"ssh4"}},
		{"u5", "u5@d.d", []string{"ssh5"}},
		{"u6", "u6@d.d", []string{"ssh6"}},
		{"u7", "u7@d.d", []string{"ssh7"}},
	}
	err = json.NewDecoder(rr.Body).Decode(&actual)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", actual, expected)
	}

	// case when wrong groups
	req, err = http.NewRequest("GET", "/api/users?groups=x,y", nil)
	req.Header.Set("X-Forwarded-User", "testuser1")
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	expected = []User{}
	err = json.NewDecoder(rr.Body).Decode(&actual)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", actual, expected)
	}
}
