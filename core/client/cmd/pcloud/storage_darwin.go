package main

import "errors"

type darwinStorage struct {
}

func CreateStorage() Storage {
	return &darwinStorage{}
}

func (s *darwinStorage) Get() (Config, error) {
	return nil, errors.New("Not implemented")
}

func (s *darwinStorage) Store(config Config) error {
	return errors.New("Not implemented")
}
