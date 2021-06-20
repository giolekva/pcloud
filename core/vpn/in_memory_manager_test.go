package vpn

import (
	"log"
	"testing"

	"inet.af/netaddr"
	"tailscale.com/ipn/ipnstate"

	"github.com/giolekva/pcloud/core/vpn/engine"
	"github.com/giolekva/pcloud/core/vpn/types"
)

func TestTwoPeers(t *testing.T) {
	ipm := NewSequentialIPManager(netaddr.MustParseIP("10.0.0.1"))
	m := NewInMemoryManager(ipm)
	privKeyA := types.NewPrivateKey()
	a, err := engine.NewFakeWireguardEngine(12345, privKeyA)
	if err != nil {
		t.Fatal(err)
	}
	privKeyB := types.NewPrivateKey()
	b, err := engine.NewFakeWireguardEngine(12346, privKeyB)
	if err != nil {
		t.Fatal(err)
	}
	nma, err := m.RegisterDevice(types.DeviceInfo{
		privKeyA.Public(),
		a.DiscoKey(),
		netaddr.MustParseIPPort("127.0.0.1:12345"),
	})
	if err != nil {
		t.Fatal(err)
	}
	m.AddNetworkMapChangeCallback(privKeyA.Public(), func(nm *types.NetworkMap) {
		log.Printf("a: Received new NetworkMap: %+v\n", nm)
		if err := a.Configure(nm); err != nil {
			t.Fatal(err)
		}
	})
	if err := a.Configure(nma); err != nil {
		t.Fatal(err)
	}
	nmb, err := m.RegisterDevice(types.DeviceInfo{
		privKeyB.Public(),
		b.DiscoKey(),
		netaddr.MustParseIPPort("127.0.0.1:12346"),
	})
	if err != nil {
		t.Fatal(err)
	}
	m.AddNetworkMapChangeCallback(privKeyB.Public(), func(nm *types.NetworkMap) {
		log.Printf("b: Received new NetworkMap: %+v\n", nm)
		if err := b.Configure(nm); err != nil {
			t.Fatal(err)
		}
	})
	if err := b.Configure(nmb); err != nil {
		t.Fatal(err)
	}
	ping := make(chan *ipnstate.PingResult, 2)
	pingCb := func(p *ipnstate.PingResult) {
		ping <- p
	}
	a.Ping(nmb.Self.VPNIP, pingCb)
	b.Ping(nma.Self.VPNIP, pingCb)
	for i := 0; i < 2; i++ {
		p := <-ping
		if p.Err != "" {
			t.Error(p.Err)
		} else {
			log.Printf("Ping received: %+v\n", p)
		}
	}
	if err := m.RemoveDevice(privKeyA.Public()); err != nil {
		t.Fatal(err)
	}
	b.Ping(nma.Self.VPNIP, pingCb)
	p := <-ping
	if p.Err == "" {
		t.Fatalf("Ping received even after removing device: %+v", p)
	}
}
