package vpn

import (
	"fmt"

	"github.com/giolekva/pcloud/core/vpn/types"

	"inet.af/netaddr"
)

// TODO(giolekva): Add Disable method which marks given IP as non-usable for future.
// It will be used when devices get removed from the network, in which case IP should not be reused for safety reasons.
type IPManager interface {
	New(pubKey types.PublicKey) (netaddr.IP, error)
	Get(pubKey types.PublicKey) (netaddr.IP, error)
}

type SequentialIPManager struct {
	cur     netaddr.IP
	keyToIP map[types.PublicKey]netaddr.IP
}

func NewSequentialIPManager(start netaddr.IP) IPManager {
	return &SequentialIPManager{
		cur:     start,
		keyToIP: make(map[types.PublicKey]netaddr.IP),
	}
}

func (m *SequentialIPManager) New(pubKey types.PublicKey) (netaddr.IP, error) {
	ip := m.cur
	if _, ok := m.keyToIP[pubKey]; ok {
		return netaddr.IP{}, fmt.Errorf("Device with public key %s has already been assigned IP", pubKey)
	}
	m.keyToIP[pubKey] = ip
	m.cur = m.cur.Next()
	return ip, nil
}

func (m *SequentialIPManager) Get(pubKey types.PublicKey) (netaddr.IP, error) {
	if ip, ok := m.keyToIP[pubKey]; ok {
		return ip, nil
	}
	return netaddr.IP{}, fmt.Errorf("Device with public key %s pubKey does not have VPN IP assigned.", pubKey)
}
