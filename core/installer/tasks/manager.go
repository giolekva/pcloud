package tasks

import (
	"fmt"
)

type TaskManager interface {
	Add(name string, task Task) error
	Get(name string) (Task, error)
}

type TaskMap struct {
	t map[string]Task
}

func NewTaskMap() *TaskMap {
	return &TaskMap{make(map[string]Task)}
}

func (m *TaskMap) Add(name string, task Task) error {
	if _, ok := m.t[name]; ok {
		return fmt.Errorf("already exists")
	}
	m.t[name] = task
	return nil
}

func (m *TaskMap) Get(name string) (Task, error) {
	if t, ok := m.t[name]; ok {
		return t, nil
	} else {
		return nil, fmt.Errorf("does not exist")
	}
}
