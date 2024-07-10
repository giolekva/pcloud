package welcome

import (
	"database/sql"
	"errors"

	"github.com/ncruces/go-sqlite3"

	"github.com/giolekva/pcloud/core/installer/soft"
)

const (
	errorUniqueConstraintViolation = 2067
)

var (
	ErrorAlreadyExists = errors.New("already exists")
)

type Commit struct {
	Hash    string
	Message string
}

type Store interface {
	CreateUser(username string, password []byte) error
	GetUserPassword(username string) ([]byte, error)
	GetApps() ([]string, error)
	GetUserApps(username string) ([]string, error)
	CreateApp(name, username string) error
	GetAppOwner(name string) (string, error)
	CreateCommit(name, hash, message string) error
	GetCommitHistory(name string) ([]Commit, error)
}

func NewStore(cf soft.RepoIO, db *sql.DB) (Store, error) {
	s := &storeImpl{cf, db}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

type storeImpl struct {
	cf soft.RepoIO
	db *sql.DB
}

func (s *storeImpl) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
            password BLOB
		);
		CREATE TABLE IF NOT EXISTS apps (
			name TEXT PRIMARY KEY,
            username TEXT
		);
		CREATE TABLE IF NOT EXISTS commits (
			app_name TEXT,
            hash TEXT,
            message TEXT
		);
	`)
	return err

}

func (s *storeImpl) CreateUser(username string, password []byte) error {
	query := `INSERT INTO users (username, password) VALUES (?, ?)`
	_, err := s.db.Exec(query, username, password)
	if err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		if ok && sqliteErr.ExtendedCode() == errorUniqueConstraintViolation {
			return ErrorAlreadyExists
		}
	}
	return err
}

func (s *storeImpl) GetUserPassword(username string) ([]byte, error) {
	query := `SELECT password FROM users WHERE username = ?`
	row := s.db.QueryRow(query, username)
	if err := row.Err(); err != nil {
		return nil, err
	}
	ret := []byte{}
	if err := row.Scan(&ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *storeImpl) CreateApp(name, username string) error {
	query := `INSERT INTO apps (name, username) VALUES (?, ?)`
	_, err := s.db.Exec(query, name, username)
	return err
}

func (s *storeImpl) GetAppOwner(name string) (string, error) {
	query := `SELECT username FROM apps WHERE name = ?`
	row := s.db.QueryRow(query, name)
	if err := row.Err(); err != nil {
		return "", err
	}
	var ret string
	if err := row.Scan(&ret); err != nil {
		return "", err
	}
	return ret, nil
}

func (s *storeImpl) GetApps() ([]string, error) {
	query := `SELECT name FROM apps`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ret := []string{}
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		ret = append(ret, name)

	}
	return ret, nil
}

func (s *storeImpl) GetUserApps(username string) ([]string, error) {
	query := `SELECT name FROM apps WHERE username = ?`
	rows, err := s.db.Query(query, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ret := []string{}
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		ret = append(ret, name)

	}
	return ret, nil
}

func (s *storeImpl) CreateCommit(name, hash, message string) error {
	query := `INSERT INTO commits (app_name, hash, message) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, name, hash, message)
	return err
}

func (s *storeImpl) GetCommitHistory(name string) ([]Commit, error) {
	query := `SELECT hash, message FROM commits WHERE app_name = ?`
	rows, err := s.db.Query(query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ret := []Commit{}
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		var c Commit
		if err := rows.Scan(&c.Hash, &c.Message); err != nil {
			return nil, err
		}
		ret = append(ret, c)

	}
	return ret, nil
}
