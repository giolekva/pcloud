package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

type FS interface {
	Exists(path string) (bool, error)
	Reader(path string) (io.ReadCloser, error)
	Writer(path string) (io.WriteCloser, error)
	Read(path string) (string, error)
	Write(path string, data string) error
	AbsolutePath(path string) string
}

type osFS struct {
	root string
}

func (f osFS) AbsolutePath(path string) string {
	return filepath.Join(f.root, path)
}

func (f osFS) Exists(path string) (bool, error) {
	_, err := os.Stat(f.AbsolutePath(path))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, nil // TODO(gio): return err
}

func (f osFS) Reader(path string) (io.ReadCloser, error) {
	return os.Open(f.AbsolutePath(path))
}

func (f osFS) Writer(path string) (io.WriteCloser, error) {
	return os.Create(f.AbsolutePath(path))
}

func (f osFS) Read(path string) (string, error) {
	r, err := f.Reader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	d, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(d), err
}

func (f osFS) Write(path string, data string) error {
	w, err := f.Writer(path)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.WriteString(w, data)
	return err
}
