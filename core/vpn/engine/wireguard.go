package engine

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/giolekva/pcloud/core/vpn/types"

	"github.com/tailscale/wireguard-go/wgcfg"
	"inet.af/netaddr"
	"tailscale.com/control/controlclient"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/types/wgkey"
	"tailscale.com/wgengine"
	"tailscale.com/wgengine/router"
)

// Wireguard specific implementation of the Engine interface.
type WireguardEngine struct {
	wg      wgengine.Engine
	port    uint16
	privKey types.PrivateKey
}

// Creates Wireguard engine.
func NewWireguardEngine(tunName string, port uint16, privKey types.PrivateKey) (Engine, error) {
	e, err := wgengine.NewUserspaceEngine(log.Printf, tunName, port)
	if err != nil {
		return nil, err
	}
	return &WireguardEngine{
		wg:      e,
		port:    port,
		privKey: privKey,
	}, nil
}

// Used for unit testing.
func NewFakeWireguardEngine(port uint16, privKey types.PrivateKey) (Engine, error) {
	e, err := wgengine.NewFakeUserspaceEngine(log.Printf, port, nil)
	if err != nil {
		return nil, err
	}
	return &WireguardEngine{
		wg:      e,
		port:    port,
		privKey: privKey,
	}, nil
}

func genWireguardConf(privKey types.PrivateKey, port uint16, netMap *types.NetworkMap) *wgcfg.Config {
	c := &wgcfg.Config{
		Name:       "foo",
		PrivateKey: wgcfg.PrivateKey(privKey),
		Addresses: []netaddr.IPPrefix{netaddr.IPPrefix{
			IP:   netMap.Self.VPNIP,
			Bits: 32, // TODO(giolekva): adapt for IPv6
		}},
		ListenPort: port,
		Peers:      make([]wgcfg.Peer, 0, len(netMap.Peers)),
	}
	for _, peer := range netMap.Peers {
		c.Peers = append(c.Peers, wgcfg.Peer{
			PublicKey: wgcfg.Key(peer.PublicKey),
			AllowedIPs: []netaddr.IPPrefix{netaddr.IPPrefix{
				IP:   peer.VPNIP,
				Bits: 32,
			}},
			Endpoints:           peer.DiscoEndpoint,
			PersistentKeepalive: 15, // TODO(giolekva): make it configurable
		})
	}
	return c
}

func genRouterConf(netMap *types.NetworkMap) *router.Config {
	c := &router.Config{
		LocalAddrs: []netaddr.IPPrefix{netaddr.IPPrefix{
			IP:   netMap.Self.VPNIP,
			Bits: 32,
		}},
		Routes: make([]netaddr.IPPrefix, 0, len(netMap.Peers)),
	}
	for _, peer := range netMap.Peers {
		c.Routes = append(c.Routes, netaddr.IPPrefix{
			IP:   peer.VPNIP,
			Bits: 32,
		})
	}
	return c
}

func genTailNetMap(privKey types.PrivateKey, port uint16, netMap *types.NetworkMap) *controlclient.NetworkMap {
	fmt.Println(netMap.Self.IPPort.String())
	c := &controlclient.NetworkMap{
		SelfNode: &tailcfg.Node{
			ID:       0, // TODO(giolekva): maybe IDs should be stored server side.
			StableID: "0",
			Name:     "0",
			Key:      tailcfg.NodeKey(netMap.Self.PublicKey),
			DiscoKey: tailcfg.DiscoKey(netMap.Self.DiscoKey),
			Addresses: []netaddr.IPPrefix{netaddr.IPPrefix{
				IP:   netMap.Self.VPNIP,
				Bits: 32,
			}},
			AllowedIPs: make([]netaddr.IPPrefix, 0, len(netMap.Peers)),
			Endpoints:  []string{netMap.Self.IPPort.String()},
			KeepAlive:  true, // TODO(giolekva): make it configurable
		},
		NodeKey:    tailcfg.NodeKey(netMap.Self.PublicKey),
		PrivateKey: wgkey.Private(privKey),
		Name:       "0",
		Addresses: []netaddr.IPPrefix{netaddr.IPPrefix{
			IP:   netMap.Self.VPNIP,
			Bits: 32,
		}},
		LocalPort: port,
		Peers:     make([]*tailcfg.Node, 0, len(netMap.Peers)),
	}
	for i, peer := range netMap.Peers {
		c.Peers = append(c.Peers, &tailcfg.Node{
			ID:       tailcfg.NodeID(i + 1),
			StableID: tailcfg.StableNodeID(fmt.Sprintf("%d", i+1)),
			Name:     fmt.Sprintf("%d", i+1),
			Key:      tailcfg.NodeKey(peer.PublicKey),
			DiscoKey: tailcfg.DiscoKey(peer.DiscoKey),
			Addresses: []netaddr.IPPrefix{netaddr.IPPrefix{
				IP:   peer.VPNIP,
				Bits: 32,
			}},
			AllowedIPs: []netaddr.IPPrefix{netaddr.IPPrefix{
				IP:   netMap.Self.VPNIP,
				Bits: 32,
			}},
			Endpoints: []string{peer.IPPort.String()},
			KeepAlive: true,
		})
	}
	return c
}

func (e *WireguardEngine) Configure(netMap *types.NetworkMap) error {
	err := e.wg.Reconfig(
		genWireguardConf(e.privKey, e.port, netMap),
		genRouterConf(netMap))
	if err != nil {
		return err
	}
	e.wg.SetNetworkMap(genTailNetMap(e.privKey, e.port, netMap))
	e.wg.RequestStatus()
	return err
}

func (e *WireguardEngine) DiscoKey() types.DiscoKey {
	return types.DiscoKey(e.wg.DiscoPublicKey())
}

func (e *WireguardEngine) DiscoEndpoint() string {
	k := e.DiscoKey()
	discoHex := hex.EncodeToString(k[:])
	return fmt.Sprintf("%s%s", discoHex, controlclient.EndpointDiscoSuffix)
}

func (e *WireguardEngine) Ping(ip netaddr.IP, cb func(*ipnstate.PingResult)) {
	e.wg.Ping(ip, cb)
}
