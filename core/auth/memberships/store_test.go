package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	if err := store.Init("admin", []string{"admin", "all"}); err != nil {
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
	err = store.Init("admin", []string{"admin", "all"})
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
	req.Header.Set("X-User", "testuser")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	body := rr.Body.String()
	fmt.Println("BODY: ", rr.Header().Get("Location"))
	if rr.Header().Get("Location") != "/group/bb" {
		t.Errorf("handler returned unexpected body: got %v want %v", body, "expected body")
	}
}
