package engine

import (
	"fmt"
	"log"
	"testing"

	"github.com/giolekva/pcloud/core/vpn/types"

	"inet.af/netaddr"
	"tailscale.com/ipn/ipnstate"
)

type node struct {
	ip      netaddr.IP
	privKey types.PrivateKey
	node    types.Node
	peers   []types.Node
	e       Engine
}

func newNode(ip string, localPort uint16) (n *node, err error) {
	n = &node{
		ip:      netaddr.MustParseIP(ip),
		privKey: types.NewPrivateKey(),
	}
	if n.e, err = NewFakeWireguardEngine(localPort, n.privKey); err != nil {
		return
	}
	n.node = types.Node{
		PublicKey: n.privKey.Public(),
		DiscoKey:  n.e.DiscoKey(),
		IPPort: netaddr.IPPortFrom(
			netaddr.IPv4(127, 0, 0, 1),
			localPort,
		),
		VPNIP: netaddr.MustParseIP(ip),
	}
	return
}

func (n *node) addPeer(x types.Node) {
	n.peers = append(n.peers, x)
}

func (n *node) configure() error {
	return n.e.Configure(&types.NetworkMap{n.node, n.peers})
}

func (n *node) ping(ip string, ch chan<- *ipnstate.PingResult) {
	n.e.Ping(netaddr.MustParseIP(ip), func(p *ipnstate.PingResult) {
		ch <- p
	})
}

func TestTwoPeers(t *testing.T) {
	var a, b *node
	var err error
	if a, err = newNode("10.0.0.1", 1234); err != nil {
		t.Fatal(err)
	}
	if b, err = newNode("10.0.0.2", 1235); err != nil {
		t.Fatal(err)
	}
	a.addPeer(b.node)
	b.addPeer(a.node)
	if err := a.configure(); err != nil {
		t.Fatal(err)
	}
	if err := b.configure(); err != nil {
		t.Fatal(err)
	}
	ping := make(chan *ipnstate.PingResult, 0)
	a.ping("10.0.0.2", ping)
	b.ping("10.0.0.1", ping)
	for i := 0; i < 2; i++ {
		p := <-ping
		if p.Err != "" {
			t.Error(p.Err)
		}
		log.Printf("Ping received: %+v\n", p)
	}
}

func TestTenPeers(t *testing.T) {
	n := 10
	nodes := make([]*node, n)
	ping := make(chan *ipnstate.PingResult, 0)
	for i := 0; i < n; i++ {
		ip := fmt.Sprintf("10.0.0.%d", i+1)
		localPort := uint16(i + 4321)
		var err error
		if nodes[i], err = newNode(ip, localPort); err != nil {
			t.Fatal(err)
		}
		for j := 0; j < i; j++ {
			nodes[i].addPeer(nodes[j].node)
			nodes[j].addPeer(nodes[i].node)
		}
	}
	for i := 0; i < n; i++ {
		if err := nodes[i].configure(); err != nil {
			t.Fatal(err)
		}
		for j := 0; j < i; j++ {
			nodes[i].ping(nodes[j].ip.String(), ping)
			nodes[j].ping(nodes[i].ip.String(), ping)
		}

	}
	for i := 0; i < n*(n-1); i++ {
		p := <-ping
		if p.Err != "" {
			t.Error(p.Err)
		} else {
			log.Printf("Ping received: %+v\n", p)
		}
	}
}
