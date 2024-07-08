package welcome

import (
	"database/sql"

	"github.com/giolekva/pcloud/core/installer/soft"
)

type Commit struct {
	Hash    string
	Message string
}

type Store interface {
	GetApps() ([]string, error)
	CreateApp(name string) error
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
		CREATE TABLE IF NOT EXISTS apps (
			name TEXT PRIMARY KEY
		);
		CREATE TABLE IF NOT EXISTS commits (
			app_name TEXT,
            hash TEXT,
            message TEXT
		);
	`)
	return err

}

func (s *storeImpl) CreateApp(name string) error {
	query := `INSERT INTO apps (name) VALUES (?)`
	_, err := s.db.Exec(query, name)
	return err
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
