package engine

import (
	"inet.af/netaddr"
	"tailscale.com/ipn/ipnstate"

	"github.com/giolekva/pcloud/core/vpn/types"
)

// Abstracts away communication with host OS needed to setup netfwork interfaces
// for VPN.
type Engine interface {
	// Reconfigures local network interfaces in accordance to the given VPN
	// layout.
	Configure(netMap *types.NetworkMap) error
	// Unique public discovery key of the current device.
	DiscoKey() types.DiscoKey
	// Sends ping to the given IP address and invokes callback with results.
	Ping(ip netaddr.IP, cb func(*ipnstate.PingResult))
}
