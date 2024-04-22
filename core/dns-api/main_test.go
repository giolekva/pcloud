package main

import (
	"testing"
)

func TestDNSSecKey(t *testing.T) {
	k, err := getDNSSecKey(osFS{"/tmp"}, "foo.bar.ge")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(k)
}
