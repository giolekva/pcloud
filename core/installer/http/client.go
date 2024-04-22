package http

import (
	"net/http"
)

type Client interface {
	Get(addr string) (*http.Response, error)
}

type realClient struct{}

func (c realClient) Get(addr string) (*http.Response, error) {
	return http.Get(addr)
}

func NewClient() Client {
	return realClient{}
}
