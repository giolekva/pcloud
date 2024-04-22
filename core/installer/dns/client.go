package dns

import (
	"net"
)

type Client interface {
	Lookup(host string) ([]net.IP, error)
}

type realClient struct{}

func NewClient() Client {
	return realClient{}
}

func (c realClient) Lookup(host string) ([]net.IP, error) {
	return net.LookupIP(host)
}
