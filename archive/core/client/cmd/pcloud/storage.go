package main

type Storage interface {
	Get() (Config, error)
	Store(Config) error
}
