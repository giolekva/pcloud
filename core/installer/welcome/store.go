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

type LastCommitInfo struct {
	Hash      string
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
	CreateCommit(name, branch, hash, message, status, error string, resources []byte) error
	GetCommitHistory(name, branch string) ([]CommitMeta, error)
	GetCommit(hash string) (Commit, error)
	GetLastCommitInfo(name, branch string) (LastCommitInfo, error)
	GetBranches(name string) ([]string, error)
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
			branch TEXT,
            hash TEXT,
            message TEXT,
            status TEXT,
            error TEXT,
            resources JSONB
		);
		CREATE TABLE IF NOT EXISTS branches (
			app_name TEXT,
			branch TEXT,
            hash TEXT,
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

func (s *storeImpl) CreateCommit(name, branch, hash, message, status, error string, resources []byte) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	query := `INSERT INTO commits (app_name, branch, hash, message, status, error, resources) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.Exec(query, name, branch, hash, message, status, error, resources)
	if err != nil {
		tx.Rollback()
		return err
	}
	branchQuery := `UPDATE branches SET hash = ?, resources = ? WHERE app_name = ? AND branch = ?`
	r, err := tx.Exec(branchQuery, hash, resources, name, branch)
	if err != nil {
		tx.Rollback()
		return err
	}
	if cnt, err := r.RowsAffected(); err != nil {
		tx.Rollback()
		return err
	} else if cnt == 0 {
		branchQuery := `INSERT INTO branches (app_name, branch, hash, resources) VALUES (?, ?, ?, ?)`
		_, err := tx.Exec(branchQuery, name, branch, hash, resources)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func (s *storeImpl) GetCommitHistory(name, branch string) ([]CommitMeta, error) {
	query := `SELECT hash, message, status, error FROM commits WHERE app_name = ? AND branch = ?`
	rows, err := s.db.Query(query, name, branch)
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

func (s *storeImpl) GetLastCommitInfo(name, branch string) (LastCommitInfo, error) {
	query := `SELECT hash, resources FROM branches WHERE app_name = ? AND branch = ?`
	row := s.db.QueryRow(query, name, branch)
	if err := row.Err(); err != nil {
		return LastCommitInfo{}, err
	}
	var ret LastCommitInfo
	var res []byte
	if err := row.Scan(&ret.Hash, &res); err != nil {
		return LastCommitInfo{}, err
	}
	if err := json.NewDecoder(bytes.NewBuffer(res)).Decode(&ret.Resources); err != nil {
		return LastCommitInfo{}, err
	}
	return ret, nil
}

func (s *storeImpl) GetBranches(name string) ([]string, error) {
	query := `SELECT DISTINCT branch FROM commits WHERE app_name = ?`
	rows, err := s.db.Query(query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ret := []string{}
	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		var b string
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		ret = append(ret, b)

	}
	return ret, nil
}
