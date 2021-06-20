package types

import (
	"github.com/tailscale/wireguard-go/wgcfg"
	"inet.af/netaddr"
)

// Private key of the client.
// MUST never leave the device it was generated on.
type PrivateKey wgcfg.PrivateKey

// Corresponding public key of the device.
type PublicKey wgcfg.Key

//Public discovery key of the device.
type DiscoKey wgcfg.Key

type DeviceInfo struct {
	PublicKey PublicKey
	DiscoKey  DiscoKey
	IPPort    netaddr.IPPort
}

// Represents single node in the network.
type Node struct {
	PublicKey     PublicKey
	DiscoKey      DiscoKey
	DiscoEndpoint string
	IPPort        netaddr.IPPort
	VPNIP         netaddr.IP
}

type NetworkMap struct {
	Self  Node
	Peers []Node
}
