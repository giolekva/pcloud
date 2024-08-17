package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/gorilla/mux"
)

var port = flag.Int("port", 8080, "Port to listen on")
var apiPort = flag.Int("api-port", 8081, "Port to listen on for API requests")
var dbPath = flag.String("db-path", "memberships.db", "Path to SQLite file")

//go:embed memberships-tmpl/*
var tmpls embed.FS

//go:embed stat
var staticResources embed.FS

type Store interface {
	// Initializes store with admin user and their groups.
	Init(user, email string, groups []string) error
	CreateGroup(owner string, group Group) error
	AddChildGroup(parent, child string) error
	AddOwnerGroup(owned_group, owner_group string) error
	DoesGroupExist(group string) (bool, error)
	GetGroupsOwnedBy(user string) ([]Group, error)
	GetGroupsUserBelongsTo(user string) ([]Group, error)
	IsGroupOwner(user, group string) (bool, error)
	IsMemberOfOwnerGroup(user, group string) (bool, error)
	AddGroupMember(user, group string) error
	AddGroupOwner(user, group string) error
	GetGroupOwners(group string) ([]string, error)
	GetGroupOwnerGroups(group string) ([]Group, error)
	GetGroupMembers(group string) ([]string, error)
	GetGroupDescription(group string) (string, error)
	GetAllTransitiveGroupsForUser(user string) ([]Group, error)
	GetGroupsGroupBelongsTo(group string) ([]Group, error)
	GetDirectChildrenGroups(group string) ([]Group, error)
	GetAllTransitiveGroupsForGroup(group string) ([]Group, error)
	RemoveFromGroupToGroup(parent, child string) error
	RemoveUserFromTable(username, groupName, tableName string) error
	GetAllGroups() ([]Group, error)
	GetUsers(username []string) ([]User, error)
	GetUser(username string) (User, error)
	AddSSHKeyForUser(username, sshKey string) error
	RemoveSSHKeyForUser(username, sshKey string) error
	CreateUser(user, email string) error
}

type Server struct {
	store         Store
	syncAddresses map[string]struct{}
	mu            sync.Mutex
}

type Group struct {
	Name        string
	Description string
}

type User struct {
	Username      string   `json:"username"`
	Email         string   `json:"email"`
	SSHPublicKeys []string `json:"sshPublicKeys,omitempty"`
}

type SQLiteStore struct {
	db *sql.DB
}

const (
	ErrorUniqueConstraintViolation     = 2067
	ErrorConstraintPrimaryKeyViolation = 1555
)

func NewSQLiteStore(db *sql.DB) (*SQLiteStore, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS groups (
			name TEXT PRIMARY KEY,
			description TEXT
		);
		CREATE TABLE IF NOT EXISTS owners (
			username TEXT,
			group_name TEXT,
			FOREIGN KEY(group_name) REFERENCES groups(name),
			UNIQUE (username, group_name)
		);
		CREATE TABLE IF NOT EXISTS owner_groups (
			owner_group TEXT,
			owned_group TEXT,
			FOREIGN KEY(owner_group) REFERENCES groups(name),
			FOREIGN KEY(owned_group) REFERENCES groups(name),
			UNIQUE (owner_group, owned_group)
		);
		CREATE TABLE IF NOT EXISTS group_to_group (
			parent_group TEXT,
			child_group TEXT,
			FOREIGN KEY(parent_group) REFERENCES groups(name),
			FOREIGN KEY(child_group) REFERENCES groups(name),
			UNIQUE (parent_group, child_group)
		);
		CREATE TABLE IF NOT EXISTS user_to_group (
			username TEXT,
			group_name TEXT,
			FOREIGN KEY(group_name) REFERENCES groups(name),
			UNIQUE (username, group_name)
		);
		CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			email TEXT,
			UNIQUE (email)
		);
		CREATE TABLE IF NOT EXISTS user_ssh_keys (
			username TEXT,
			ssh_key TEXT,
			UNIQUE (ssh_key),
			FOREIGN KEY(username) REFERENCES users(username)
		);`)
	if err != nil {
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Init(user, email string, groups []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	row := tx.QueryRow("SELECT COUNT(*) FROM groups")
	var count int
	if err := row.Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("Store already initialised")
	}
	query := `INSERT INTO users (username, email) VALUES (?, ?)`
	if _, err := tx.Exec(query, user, email); err != nil {
		return err
	}
	for _, g := range groups {
		query = `INSERT INTO groups (name, description) VALUES (?, '')`
		if _, err := tx.Exec(query, g); err != nil {
			return err
		}
		query = `INSERT INTO owners (username, group_name) VALUES (?, ?)`
		if _, err := tx.Exec(query, user, g); err != nil {
			return err
		}
		query = `INSERT INTO user_to_group (username, group_name) VALUES (?, ?)`
		if _, err := tx.Exec(query, user, g); err != nil {
			return err
		}
	}
	return tx.Commit()
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

func (s *SQLiteStore) GetGroupsOwnedBy(user string) ([]Group, error) {
	query := `
        SELECT groups.name, groups.description
        FROM groups
        JOIN owners ON groups.name = owners.group_name
        WHERE owners.username = ?`
	return s.queryGroups(query, user)
}

func (s *SQLiteStore) GetGroupsUserBelongsTo(user string) ([]Group, error) {
	query := `
        SELECT groups.name, groups.description
        FROM groups
        JOIN user_to_group ON groups.name = user_to_group.group_name
        WHERE user_to_group.username = ?`
	return s.queryGroups(query, user)
}

func (s *SQLiteStore) CreateGroup(owner string, group Group) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	query := `INSERT INTO groups (name, description) VALUES (?, ?)`
	if _, err := tx.Exec(query, group.Name, group.Description); err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		if ok && sqliteErr.ExtendedCode() == ErrorConstraintPrimaryKeyViolation {
			return fmt.Errorf("Group with the name %s already exists", group.Name)
		}
		return err
	}
	query = `INSERT INTO owners (username, group_name) VALUES (?, ?)`
	if _, err := tx.Exec(query, owner, group.Name); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) IsGroupOwner(user, group string) (bool, error) {
	query := `
        SELECT EXISTS (
            SELECT 1
            FROM owners
            WHERE username = ? AND group_name = ?
        )`
	var exists bool
	if err := s.db.QueryRow(query, user, group).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (s *SQLiteStore) AddGroupMember(user, group string) error {
	_, err := s.db.Exec(`INSERT INTO user_to_group (username, group_name) VALUES (?, ?)`, user, group)
	if err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		if ok && sqliteErr.ExtendedCode() == ErrorUniqueConstraintViolation {
			return fmt.Errorf("%s is already a member of group %s", user, group)
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) AddGroupOwner(user, group string) error {
	_, err := s.db.Exec(`INSERT INTO owners (username, group_name) VALUES (?, ?)`, user, group)
	if err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		if ok && sqliteErr.ExtendedCode() == ErrorUniqueConstraintViolation {
			return fmt.Errorf("%s is already an owner of group %s", user, group)
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) getUsersByGroup(table, group string) ([]string, error) {
	query := fmt.Sprintf("SELECT username FROM %s WHERE group_name = ?", table)
	rows, err := s.db.Query(query, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		users = append(users, username)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *SQLiteStore) GetGroupOwners(group string) ([]string, error) {
	return s.getUsersByGroup("owners", group)
}

func (s *SQLiteStore) GetGroupMembers(group string) ([]string, error) {
	return s.getUsersByGroup("user_to_group", group)
}

func (s *SQLiteStore) GetGroupDescription(group string) (string, error) {
	var description string
	query := `SELECT description FROM groups WHERE name = ?`
	if err := s.db.QueryRow(query, group).Scan(&description); err != nil {
		return "", err
	}
	return description, nil
}

func (s *SQLiteStore) DoesGroupExist(group string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM groups WHERE name = ?)`
	var exists bool
	if err := s.db.QueryRow(query, group).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (s *SQLiteStore) AddChildGroup(parent, child string) error {
	if parent == child {
		return fmt.Errorf("Parent and child groups can not have same name")
	}
	exists, err := s.DoesGroupExist(parent)
	if err != nil {
		return fmt.Errorf("Error checking parent group existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("Parent group with name %s does not exist", parent)
	}
	exists, err = s.DoesGroupExist(child)
	if err != nil {
		return fmt.Errorf("Error checking child group existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("Child group with name %s does not exist", child)
	}
	parentGroups, err := s.GetAllTransitiveGroupsForGroup(parent)
	if err != nil {
		return err
	}
	for _, group := range parentGroups {
		if group.Name == child {
			return fmt.Errorf("Circular reference detected: group %s is already a parent of group %s", child, parent)
		}
	}
	_, err = s.db.Exec(`INSERT INTO group_to_group (parent_group, child_group) VALUES (?, ?)`, parent, child)
	if err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		if ok && sqliteErr.ExtendedCode() == ErrorUniqueConstraintViolation {
			return fmt.Errorf("Child group name %s already exists in group %s", child, parent)
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) GetAllTransitiveGroupsForUser(user string) ([]Group, error) {
	if groups, err := s.GetGroupsUserBelongsTo(user); err != nil {
		return nil, err
	} else {
		visited := map[string]struct{}{}
		return s.getAllParentGroupsRecursive(groups, visited)
	}
}

func (s *SQLiteStore) GetAllTransitiveGroupsForGroup(group string) ([]Group, error) {
	if p, err := s.GetGroupsGroupBelongsTo(group); err != nil {
		return nil, err
	} else {
		// Mark initial group as visited
		visited := map[string]struct{}{
			group: struct{}{},
		}
		return s.getAllParentGroupsRecursive(p, visited)
	}
}

func (s *SQLiteStore) getAllParentGroupsRecursive(groups []Group, visited map[string]struct{}) ([]Group, error) {
	var ret []Group
	for _, g := range groups {
		if _, ok := visited[g.Name]; ok {
			continue
		}
		visited[g.Name] = struct{}{}
		ret = append(ret, g)
		if p, err := s.GetGroupsGroupBelongsTo(g.Name); err != nil {
			return nil, err
		} else if res, err := s.getAllParentGroupsRecursive(p, visited); err != nil {
			return nil, err
		} else {
			ret = append(ret, res...)
		}
	}
	return ret, nil
}

func (s *SQLiteStore) GetGroupsGroupBelongsTo(group string) ([]Group, error) {
	query := `
        SELECT groups.name, groups.description
        FROM groups
        JOIN group_to_group ON groups.name = group_to_group.parent_group
        WHERE group_to_group.child_group = ?`
	rows, err := s.db.Query(query, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var parentGroups []Group
	for rows.Next() {
		var parentGroup Group
		if err := rows.Scan(&parentGroup.Name, &parentGroup.Description); err != nil {
			return nil, err
		}
		parentGroups = append(parentGroups, parentGroup)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return parentGroups, nil
}

func (s *SQLiteStore) GetDirectChildrenGroups(group string) ([]Group, error) {
	query := `
        SELECT groups.name, groups.description
        FROM groups
        JOIN group_to_group ON groups.name = group_to_group.child_group
        WHERE group_to_group.parent_group = ?`
	rows, err := s.db.Query(query, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var childrenGroups []Group
	for rows.Next() {
		var childGroup Group
		if err := rows.Scan(&childGroup.Name, &childGroup.Description); err != nil {
			return nil, err
		}
		childrenGroups = append(childrenGroups, childGroup)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return childrenGroups, nil
}

func (s *SQLiteStore) RemoveFromGroupToGroup(parent, child string) error {
	query := `DELETE FROM group_to_group WHERE parent_group = ? AND child_group = ?`
	rowDeleted, err := s.db.Exec(query, parent, child)
	if err != nil {
		return err
	}
	rowDeletedNumber, err := rowDeleted.RowsAffected()
	if err != nil {
		return err
	}
	if rowDeletedNumber == 0 {
		return fmt.Errorf("Pair of parent '%s' and child '%s' groups not found", parent, child)
	}
	return nil
}

func (s *SQLiteStore) RemoveUserFromTable(username, groupName, tableName string) error {
	if tableName == "owners" {
		owners, err := s.GetGroupOwners(groupName)
		if err != nil {
			return err
		}
		if len(owners) == 1 {
			return fmt.Errorf("Cannot remove the last owner of the group")
		}
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE username = ? AND group_name = ?", tableName)
	rowDeleted, err := s.db.Exec(query, username, groupName)
	if err != nil {
		return err
	}
	rowDeletedNumber, err := rowDeleted.RowsAffected()
	if err != nil {
		return err
	}
	if rowDeletedNumber == 0 {
		return fmt.Errorf("Pair of group '%s' and user '%s' not found", groupName, username)
	}
	return nil
}

func (s *SQLiteStore) AddOwnerGroup(owner_group, owned_group string) error {
	if owned_group == owner_group {
		return fmt.Errorf("Group can not own itself")
	}
	exists, err := s.DoesGroupExist(owned_group)
	if err != nil {
		return fmt.Errorf("Error checking owned group existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("Owned group with name %s does not exist", owned_group)
	}
	exists, err = s.DoesGroupExist(owner_group)
	if err != nil {
		return fmt.Errorf("Error checking owner group existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("Owner group with name %s does not exist", owner_group)
	}
	_, err = s.db.Exec(`INSERT INTO owner_groups (owner_group, owned_group) VALUES (?, ?)`, owner_group, owned_group)
	if err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		if ok && sqliteErr.ExtendedCode() == ErrorUniqueConstraintViolation {
			return fmt.Errorf("Group named %s is already owner of a group %s", owner_group, owned_group)
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) GetGroupOwnerGroups(group string) ([]Group, error) {
	query := `
        SELECT groups.name, groups.description
        FROM groups
        JOIN owner_groups ON groups.name = owner_groups.owner_group
        WHERE owner_groups.owned_group = ?`
	return s.queryGroups(query, group)
}

func (s *SQLiteStore) IsMemberOfOwnerGroup(user, group string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM owner_groups
			INNER JOIN user_to_group ON owner_groups.owner_group = user_to_group.group_name
			WHERE owner_groups.owned_group = ? AND user_to_group.username = ?)`
	var exists bool
	err := s.db.QueryRow(query, group, user).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (s *SQLiteStore) GetAllGroups() ([]Group, error) {
	query := `SELECT name, description FROM groups`
	return s.queryGroups(query)
}

func (s *SQLiteStore) AddSSHKeyForUser(username, sshKey string) error {
	_, err := s.db.Exec(`INSERT INTO user_ssh_keys (username, ssh_key) VALUES (?, ?)`, username, sshKey)
	if err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		if ok && sqliteErr.ExtendedCode() == ErrorUniqueConstraintViolation {
			return fmt.Errorf("%s such SSH public key already exists", sshKey)
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) RemoveSSHKeyForUser(username, sshKey string) error {
	_, err := s.db.Exec(`DELETE FROM user_ssh_keys WHERE username = ? AND ssh_key = ?`, username, sshKey)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) GetUsers(usernames []string) ([]User, error) {
	var rows *sql.Rows
	var err error
	query := `
		SELECT users.username, users.email, GROUP_CONCAT(user_ssh_keys.ssh_key, ',')
		FROM users
		LEFT JOIN user_ssh_keys ON users.username = user_ssh_keys.username`
	var args []interface{}
	if usernames != nil {
		if len(usernames) == 0 {
			return []User{}, nil
		}
		query += " WHERE users.username IN ("
		placeholders := strings.Repeat("?,", len(usernames)-1) + "?"
		query += placeholders + ") "
		for _, username := range usernames {
			args = append(args, username)
		}
	}
	query += " GROUP BY users.username"
	rows, err = s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var userInfos []User
	for rows.Next() {
		var username, email string
		var sshKeys sql.NullString
		if err := rows.Scan(&username, &email, &sshKeys); err != nil {
			return nil, err
		}
		user := User{
			Username: username,
			Email:    email,
		}
		if sshKeys.Valid {
			user.SSHPublicKeys = strings.Split(sshKeys.String, ",")
		}
		userInfos = append(userInfos, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return userInfos, nil
}

func (s *SQLiteStore) GetUser(username string) (User, error) {
	var user User
	user.Username = username
	query := `
		SELECT users.email, GROUP_CONCAT(user_ssh_keys.ssh_key, ',')
		FROM users
		LEFT JOIN user_ssh_keys ON users.username = user_ssh_keys.username
		WHERE users.username = ?
		GROUP BY users.username
	`
	row := s.db.QueryRow(query, username)
	var sshKeys sql.NullString
	err := row.Scan(&user.Email, &sshKeys)
	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, fmt.Errorf("no user found with username %s", username)
		}
		return User{}, err
	}
	if sshKeys.Valid {
		user.SSHPublicKeys = strings.Split(sshKeys.String, ",")
	}
	return user, nil
}

func (s *SQLiteStore) CreateUser(user, email string) error {
	_, err := s.db.Exec(`INSERT INTO users (username, email) VALUES (?, ?)`, user, email)
	if err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		if ok {
			if sqliteErr.ExtendedCode() == ErrorUniqueConstraintViolation {
				if strings.Contains(err.Error(), "UNIQUE constraint failed: users.username") {
					return fmt.Errorf("username %s already exists", user)
				}
				if strings.Contains(err.Error(), "UNIQUE constraint failed: users.email") {
					return fmt.Errorf("email %s already exists", email)
				}
			}
		}
		return err
	}
	return nil
}

func getLoggedInUser(r *http.Request) (string, error) {
	if user := r.Header.Get("X-User"); user != "" {
		return user, nil
	} else {
		return "", fmt.Errorf("unauthenticated")
	}
	// return "tabo", nil
}

type Status int

const (
	Owner Status = iota
	Member
)

func (s *Server) Start() error {
	e := make(chan error)
	go func() {
		r := mux.NewRouter()
		r.PathPrefix("/stat/").Handler(http.FileServer(http.FS(staticResources)))
		r.HandleFunc("/group/{group-name}/add-user/", s.addUserToGroupHandler).Methods(http.MethodPost)
		r.HandleFunc("/group/{parent-group}/add-child-group", s.addChildGroupHandler).Methods(http.MethodPost)
		r.HandleFunc("/group/{owned-group}/add-owner-group", s.addOwnerGroupHandler).Methods(http.MethodPost)
		r.HandleFunc("/group/{parent-group}/remove-child-group/{child-group}", s.removeChildGroupHandler).Methods(http.MethodPost)
		r.HandleFunc("/group/{group-name}/remove-owner/{username}", s.removeOwnerFromGroupHandler).Methods(http.MethodPost)
		r.HandleFunc("/group/{group-name}/remove-member/{username}", s.removeMemberFromGroupHandler).Methods(http.MethodPost)
		r.HandleFunc("/group/{group-name}", s.groupHandler)
		r.HandleFunc("/user/{username}/ssh-key", s.addSSHKeyForUserHandler).Methods(http.MethodPost)
		r.HandleFunc("/user/{username}/remove-ssh-key", s.removeSSHKeyForUserHandler).Methods(http.MethodPost)
		r.HandleFunc("/user/{username}", s.userHandler)
		r.HandleFunc("/create-group", s.createGroupHandler).Methods(http.MethodPost)
		r.HandleFunc("/", s.homePageHandler)
		e <- http.ListenAndServe(fmt.Sprintf(":%d", *port), r)
	}()
	go func() {
		r := mux.NewRouter()
		r.HandleFunc("/api/init", s.apiInitHandler)
		r.HandleFunc("/api/user/{username}", s.apiMemberOfHandler)
		r.HandleFunc("/api/users", s.apiGetAllUsers).Methods(http.MethodGet)
		r.HandleFunc("/api/users", s.apiCreateUser).Methods(http.MethodPost)
		e <- http.ListenAndServe(fmt.Sprintf(":%d", *apiPort), r)
	}()
	return <-e
}

type GroupData struct {
	Group      Group
	Membership string
}

func (s *Server) checkIsOwner(w http.ResponseWriter, user, group string) error {
	isOwner, err := s.store.IsGroupOwner(user, group)
	if err != nil {
		return err
	}
	if isOwner {
		return nil
	}
	// TODO(dtabidze): right now this only checks if user is member of just one lvl upper group. should add transitive group check.
	isMemberOfOwnerGroup, err := s.store.IsMemberOfOwnerGroup(user, group)
	if err != nil {
		return err
	}
	if !isMemberOfOwnerGroup {
		return fmt.Errorf("You are not the owner or a member of any owner group of the group %s", group)
	}
	return nil
}

type templates struct {
	group *template.Template
	user  *template.Template
}

func parseTemplates(fs embed.FS) (templates, error) {
	base, err := template.ParseFS(fs, "memberships-tmpl/base.html")
	if err != nil {
		return templates{}, err
	}
	parse := func(path string) (*template.Template, error) {
		if b, err := base.Clone(); err != nil {
			return nil, err
		} else {
			return b.ParseFS(fs, path)
		}
	}
	user, err := parse("memberships-tmpl/user.html")
	if err != nil {
		return templates{}, err
	}
	group, err := parse("memberships-tmpl/group.html")
	if err != nil {
		return templates{}, err
	}
	return templates{group, user}, nil
}

func (s *Server) homePageHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/user/"+loggedInUser, http.StatusSeeOther)
}

type UserPageData struct {
	OwnerGroups      []Group
	MembershipGroups []Group
	TransitiveGroups []Group
	LoggedInUserPage bool
	CurrentUser      string
	SSHPublicKeys    []string
	Email            string
	ErrorMessage     string
}

func (s *Server) userHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	errorMsg := r.URL.Query().Get("errorMessage")
	vars := mux.Vars(r)
	user := strings.ToLower(vars["username"])
	// TODO(dtabidze): should check if username exists or not.
	loggedInUserPage := loggedInUser == user
	ownerGroups, err := s.store.GetGroupsOwnedBy(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	membershipGroups, err := s.store.GetGroupsUserBelongsTo(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	transitiveGroups, err := s.store.GetAllTransitiveGroupsForUser(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userInfo, err := s.store.GetUser(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := UserPageData{
		OwnerGroups:      ownerGroups,
		MembershipGroups: membershipGroups,
		TransitiveGroups: transitiveGroups,
		LoggedInUserPage: loggedInUserPage,
		CurrentUser:      user,
		SSHPublicKeys:    userInfo.SSHPublicKeys,
		Email:            userInfo.Email,
		ErrorMessage:     errorMsg,
	}
	templates, err := parseTemplates(tmpls)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := templates.user.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) createGroupHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var group Group
	group.Name = r.PostFormValue("group-name")
	if err := isValidGroupName(group.Name); err != nil {
		// http.Error(w, err.Error(), http.StatusBadRequest)
		redirectURL := fmt.Sprintf("/user/%s?errorMessage=%s", loggedInUser, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	group.Description = r.PostFormValue("description")
	if err := s.store.CreateGroup(loggedInUser, group); err != nil {
		// http.Error(w, err.Error(), http.StatusInternalServerError)
		redirectURL := fmt.Sprintf("/user/%s?errorMessage=%s", loggedInUser, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type GroupPageData struct {
	GroupName        string
	Description      string
	Owners           []string
	Members          []string
	AllGroups        []Group
	TransitiveGroups []Group
	ChildGroups      []Group
	OwnerGroups      []Group
	ErrorMessage     string
}

func (s *Server) groupHandler(w http.ResponseWriter, r *http.Request) {
	_, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	errorMsg := r.URL.Query().Get("errorMessage")
	vars := mux.Vars(r)
	groupName := vars["group-name"]
	exists, err := s.store.DoesGroupExist(groupName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		errorMsg = fmt.Sprintf("group with the name '%s' not found", groupName)
		http.Error(w, errorMsg, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	owners, err := s.store.GetGroupOwners(groupName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	members, err := s.store.GetGroupMembers(groupName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	description, err := s.store.GetGroupDescription(groupName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	allGroups, err := s.store.GetAllGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	transitiveGroups, err := s.store.GetAllTransitiveGroupsForGroup(groupName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	childGroups, err := s.store.GetDirectChildrenGroups(groupName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ownerGroups, err := s.store.GetGroupOwnerGroups(groupName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := GroupPageData{
		GroupName:        groupName,
		Description:      description,
		Owners:           owners,
		Members:          members,
		AllGroups:        allGroups,
		TransitiveGroups: transitiveGroups,
		ChildGroups:      childGroups,
		OwnerGroups:      ownerGroups,
		ErrorMessage:     errorMsg,
	}
	templates, err := parseTemplates(tmpls)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := templates.group.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) removeChildGroupHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	parentGroup := vars["parent-group"]
	childGroup := vars["child-group"]
	if err := isValidGroupName(parentGroup); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := isValidGroupName(childGroup); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.checkIsOwner(w, loggedInUser, parentGroup); err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", parentGroup, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}
	err = s.store.RemoveFromGroupToGroup(parentGroup, childGroup)
	if err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", parentGroup, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/group/"+parentGroup, http.StatusSeeOther)
}

func (s *Server) removeOwnerFromGroupHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	username := vars["username"]
	groupName := vars["group-name"]
	tableName := "owners"
	if err := isValidGroupName(groupName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.checkIsOwner(w, loggedInUser, groupName); err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", groupName, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}
	err = s.store.RemoveUserFromTable(username, groupName, tableName)
	if err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", groupName, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/group/"+groupName, http.StatusSeeOther)
}

func (s *Server) removeMemberFromGroupHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	username := vars["username"]
	groupName := vars["group-name"]
	tableName := "user_to_group"
	if err := isValidGroupName(groupName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.checkIsOwner(w, loggedInUser, groupName); err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", groupName, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}
	err = s.store.RemoveUserFromTable(username, groupName, tableName)
	if err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", groupName, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/group/"+groupName, http.StatusSeeOther)
}

func (s *Server) addUserToGroupHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	groupName := vars["group-name"]
	if err := isValidGroupName(groupName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	username := strings.ToLower(r.FormValue("username"))
	if username == "" {
		http.Error(w, "Username parameter is required", http.StatusBadRequest)
		return
	}
	status, err := convertStatus(r.FormValue("status"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.checkIsOwner(w, loggedInUser, groupName); err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", groupName, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}
	switch status {
	case Owner:
		err = s.store.AddGroupOwner(username, groupName)
	case Member:
		err = s.store.AddGroupMember(username, groupName)
	default:
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}
	if err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", groupName, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/group/"+groupName, http.StatusSeeOther)
}

func (s *Server) addChildGroupHandler(w http.ResponseWriter, r *http.Request) {
	// TODO(dtabidze): In future we might need to make one group OWNER of another and not just a member.
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	parentGroup := vars["parent-group"]
	if err := isValidGroupName(parentGroup); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	childGroup := r.FormValue("child-group")
	if err := isValidGroupName(childGroup); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.checkIsOwner(w, loggedInUser, parentGroup); err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", parentGroup, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}
	if err := s.store.AddChildGroup(parentGroup, childGroup); err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", parentGroup, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/group/"+parentGroup, http.StatusSeeOther)
}

func (s *Server) addOwnerGroupHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	ownedGroup := vars["owned-group"]
	if err := isValidGroupName(ownedGroup); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ownerGroup := r.FormValue("owner-group")
	if err := isValidGroupName(ownerGroup); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.checkIsOwner(w, loggedInUser, ownedGroup); err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", ownedGroup, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}
	if err := s.store.AddOwnerGroup(ownerGroup, ownedGroup); err != nil {
		redirectURL := fmt.Sprintf("/group/%s?errorMessage=%s", ownedGroup, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/group/"+ownedGroup, http.StatusSeeOther)
}

func (s *Server) addSSHKeyForUserHandler(w http.ResponseWriter, r *http.Request) {
	defer s.pingAllSyncAddresses()
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	username := vars["username"]
	if loggedInUser != username {
		http.Error(w, "You are not allowed to add SSH key for someone else", http.StatusUnauthorized)
		return
	}
	sshKey := r.FormValue("ssh-key")
	if sshKey == "" {
		http.Error(w, "SSH key not present", http.StatusBadRequest)
		return
	}
	if err := s.store.AddSSHKeyForUser(username, sshKey); err != nil {
		redirectURL := fmt.Sprintf("/user/%s?errorMessage=%s", loggedInUser, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/user/"+loggedInUser, http.StatusSeeOther)
}

func (s *Server) removeSSHKeyForUserHandler(w http.ResponseWriter, r *http.Request) {
	defer s.pingAllSyncAddresses()
	loggedInUser, err := getLoggedInUser(r)
	if err != nil {
		http.Error(w, "User Not Logged In", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	username := vars["username"]
	if loggedInUser != username {
		http.Error(w, "You are not allowed to remove SSH key for someone else", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	sshKey := r.FormValue("ssh-key")
	if sshKey == "" {
		http.Error(w, "SSH key not present", http.StatusBadRequest)
		return
	}
	if err := s.store.RemoveSSHKeyForUser(username, sshKey); err != nil {
		redirectURL := fmt.Sprintf("/user/%s?errorMessage=%s", loggedInUser, url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/user/"+loggedInUser, http.StatusSeeOther)
}

type initRequest struct {
	User   string   `json:"user"`
	Email  string   `json:"email"`
	Groups []string `json:"groups"`
}

func (s *Server) apiInitHandler(w http.ResponseWriter, r *http.Request) {
	var req initRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.Init(req.User, req.Email, req.Groups); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type userInfo struct {
	MemberOf []string `json:"memberOf"`
}

func (s *Server) apiMemberOfHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user, ok := vars["username"]
	if !ok || user == "" {
		http.Error(w, "Username parameter is required", http.StatusBadRequest)
		return
	}
	user = strings.ToLower(user)
	transitiveGroups, err := s.store.GetAllTransitiveGroupsForUser(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var groupNames []string
	for _, group := range transitiveGroups {
		groupNames = append(groupNames, group.Name)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userInfo{groupNames}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) apiGetAllUsers(w http.ResponseWriter, r *http.Request) {
	defer s.pingAllSyncAddresses()
	selfAddress := r.FormValue("selfAddress")
	if selfAddress != "" {
		s.addSyncAddress(selfAddress)
	}
	var users []User
	var err error
	groups := r.FormValue("groups")
	if groups == "" {
		users, err = s.store.GetUsers(nil)
	} else {
		uniqueUsers := make(map[string]struct{})
		g := strings.Split(groups, ",")
		uniqueTG := make(map[string]struct{})
		for _, group := range g {
			uniqueTG[group] = struct{}{}
			trGroups, err := s.store.GetAllTransitiveGroupsForGroup(group)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for _, tg := range trGroups {
				uniqueTG[tg.Name] = struct{}{}
			}
		}
		for group := range uniqueTG {
			u, err := s.store.GetGroupMembers(group)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for _, user := range u {
				uniqueUsers[user] = struct{}{}
			}
		}
		usernames := make([]string, 0, len(uniqueUsers))
		for username := range uniqueUsers {
			usernames = append(usernames, username)
		}
		users, err = s.store.GetUsers(usernames)
	}
	if err != nil {
		http.Error(w, "Failed to retrieve user infos", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type createUserRequest struct {
	User  string `json:"user"`
	Email string `json:"email"`
}

func (s *Server) apiCreateUser(w http.ResponseWriter, r *http.Request) {
	defer s.pingAllSyncAddresses()
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.User == "" {
		http.Error(w, "Username cannot be empty", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		http.Error(w, "Email cannot be empty", http.StatusBadRequest)
		return
	}
	if err := s.store.CreateUser(strings.ToLower(req.User), strings.ToLower(req.Email)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) pingAllSyncAddresses() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for address := range s.syncAddresses {
		resp, err := http.Get(address)
		if err != nil {
			log.Printf("Failed to ping %s: %v", address, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("Ping to %s returned status %d", address, resp.StatusCode)
		}
	}
}

func (s *Server) addSyncAddress(address string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncAddresses[address] = struct{}{}
}

func convertStatus(status string) (Status, error) {
	switch status {
	case "Owner":
		return Owner, nil
	case "Member":
		return Member, nil
	default:
		return Owner, fmt.Errorf("invalid status: %s", status)
	}
}

func isValidGroupName(group string) error {
	if strings.TrimSpace(group) == "" {
		return fmt.Errorf("Group name can't be empty or contain only whitespaces")
	}
	validGroupName := regexp.MustCompile(`^[a-z0-9\-_:.\/ ]+$`)
	if !validGroupName.MatchString(group) {
		return fmt.Errorf("Group name should contain only lowercase letters, digits, -, _, :, ., /")
	}
	return nil
}

func main() {
	flag.Parse()
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		panic(err)
	}
	store, err := NewSQLiteStore(db)
	if err != nil {
		panic(err)
	}
	s := Server{
		store:         store,
		syncAddresses: make(map[string]struct{}),
		mu:            sync.Mutex{},
	}
	log.Fatal(s.Start())
}
