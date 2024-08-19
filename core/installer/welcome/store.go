package welcome

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/ncruces/go-sqlite3"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

const (
	errorConstraintPrimaryKeyViolation = 1555
)

var (
	ErrorAlreadyExists = errors.New("already exists")
)

type CommitMeta struct {
	Status  string
	Error   string
	Hash    string
	Message string
}

type Commit struct {
	CommitMeta
	Resources installer.ReleaseResources
}

type Store interface {
	CreateUser(username string, password []byte, network string) error
	GetUserPassword(username string) ([]byte, error)
	GetUserNetwork(username string) (string, error)
	GetApps() ([]string, error)
	GetUserApps(username string) ([]string, error)
	CreateApp(name, username string) error
	GetAppOwner(name string) (string, error)
	CreateCommit(name, hash, message, status, error string, resources []byte) error
	GetCommitHistory(name string) ([]CommitMeta, error)
	GetCommit(hash string) (Commit, error)
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
            password BLOB,
            network TEXT
		);
		CREATE TABLE IF NOT EXISTS apps (
			name TEXT PRIMARY KEY,
            username TEXT
		);
		CREATE TABLE IF NOT EXISTS commits (
			app_name TEXT,
            hash TEXT,
            message TEXT,
            status TEXT,
            error TEXT,
            resources JSONB
		);
	`)
	return err

}

func (s *storeImpl) CreateUser(username string, password []byte, network string) error {
	query := `INSERT INTO users (username, password, network) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, username, password, network)
	if err != nil {
		sqliteErr, ok := err.(*sqlite3.Error)
		if ok && sqliteErr.ExtendedCode() == errorConstraintPrimaryKeyViolation {
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

func (s *storeImpl) GetUserNetwork(username string) (string, error) {
	query := `SELECT network FROM users WHERE username = ?`
	row := s.db.QueryRow(query, username)
	if err := row.Err(); err != nil {
		return "", err
	}
	var ret string
	if err := row.Scan(&ret); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return "", nil
		}
		return "", err
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

func (s *storeImpl) CreateCommit(name, hash, message, status, error string, resources []byte) error {
	query := `INSERT INTO commits (app_name, hash, message, status, error, resources) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, name, hash, message, status, error, resources)
	return err
}

func (s *storeImpl) GetCommitHistory(name string) ([]CommitMeta, error) {
	query := `SELECT hash, message, status, error FROM commits WHERE app_name = ?`
	rows, err := s.db.Query(query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ret := []CommitMeta{}
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		var c CommitMeta
		if err := rows.Scan(&c.Hash, &c.Message, &c.Status, &c.Error); err != nil {
			return nil, err
		}
		ret = append(ret, c)

	}
	return ret, nil
}

func (s *storeImpl) GetCommit(hash string) (Commit, error) {
	query := `SELECT hash, message, status, error, resources FROM commits WHERE hash = ?`
	row := s.db.QueryRow(query, hash)
	if err := row.Err(); err != nil {
		return Commit{}, err
	}
	var ret Commit
	var c Commit
	var res []byte
	if err := row.Scan(&c.Hash, &c.Message, &c.Status, &c.Error, &res); err != nil {
		return Commit{}, err
	}
	if err := json.NewDecoder(bytes.NewBuffer(res)).Decode(&ret.Resources); err != nil {
		return Commit{}, err
	}
	return ret, nil
}
