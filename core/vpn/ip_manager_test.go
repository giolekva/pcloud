package vpn

import (
	"log"
	"testing"

	"github.com/giolekva/pcloud/core/vpn/types"
	"inet.af/netaddr"
)

func TestNewGet(t *testing.T) {
	m := NewSequentialIPManager(netaddr.MustParseIP("10.0.0.1"))
	a := types.NewPrivateKey()
	b := types.NewPrivateKey()
	ipA, err := m.New(a.Public())
	if err != nil {
		log.Fatal(err)
	}
	if ipA.String() != "10.0.0.1" {
		t.Fatalf("Expected 10.0.0.1 Got: %s", ipA.String())
	}
	ipA, err = m.Get(a.Public())
	if err != nil {
		log.Fatal(err)
	}
	if ipA.String() != "10.0.0.1" {
		t.Fatalf("Expected 10.0.0.1 Got: %s", ipA.String())
	}
	ipB, err := m.New(b.Public())
	if err != nil {
		log.Fatal(err)
	}
	if ipB.String() != "10.0.0.2" {
		t.Fatalf("Expected 10.0.0.2 Got: %s", ipB.String())
	}
	ipB, err = m.Get(b.Public())
	if err != nil {
		log.Fatal(err)
	}
	if ipB.String() != "10.0.0.2" {
		t.Fatalf("Expected 10.0.0.2 Got: %s", ipB.String())
	}

}

func TestGetNonExistentPublicKey(t *testing.T) {
	m := NewSequentialIPManager(netaddr.MustParseIP("10.0.0.1"))
	a := types.NewPrivateKey()
	if _, err := m.Get(a.Public()); err == nil {
		t.Fatal("Returned IP for non existent public key")
	}
}
