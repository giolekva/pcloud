package main

import (
	"database/sql"
	"testing"

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
	} else if err.Error() != "store already initialised" {
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
	if err := store.AddChildGroup("a", "a"); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
