package appmanager

import (
	"encoding/gob"
	"os"
)

type App struct {
	Namespace string
	Triggers  *Triggers
}

// TODO(giolekva): add interface
type Manager struct {
	Apps map[string]App
}

func NewEmptyManager() *Manager {
	return &Manager{make(map[string]App)}
}

func LoadManagerStateFromFile(path string) (*Manager, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewEmptyManager(), nil
		}
		return nil, err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var m Manager
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func StoreManagerStateToFile(m *Manager, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	if err := enc.Encode(*m); err != nil {
		return err
	}
	return nil
}
