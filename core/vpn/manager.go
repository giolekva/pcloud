package vpn

import (
	"github.com/giolekva/pcloud/core/vpn/types"
)

type NetworkMapChangeCallback func(*types.NetworkMap)

// Manager interface manages mesh VPN configuration for all the devices registed by all users.
// It does enforce device to device ACLs but delegates user authorization to the client.
type Manager interface {
	// Registers new device with given public key and name.
	// New device is isolated from the rest of the network until it is explicitely added to
	// an existing group.
	RegisterDevice(name string, pubKey types.PublicKey) error
	// Completely removes device with given public key from the network.
	RemoveDevice(pubKey types.PublicKey) error
	// Creates new group with given name and returns it's id.
	// Name does not have to be unique.
	CreateGroup(name string) (types.GroupID, error)
	// Deletes group with given id.
	DeleteGroup(id types.GroupID) error
	// Adds device with given public key to the group and returns updated network configuration.
	AddDeviceToGroup(pubKey types.PublicKey, id types.GroupID) (*types.NetworkMap, error)
	// Removes device from the group and returns updated network configuration.
	RemoveDeviceFromGroup(pubKey types.PublicKey, id types.GroupID) (*types.NetworkMap, error)
	// Returns network configuration for a device with give public key.
	// Result of this call must be encrypted with the same public key before
	// sending it back to the client, so only the owner of it's corresponding
	// private key is able to decrypt and use it.
	GetNetworkMap(pubKey types.PublicKey) (*types.NetworkMap, error)
	// AddNetworkMapChangeCallback can be used to receive new network configurations
	// for a device with given public key.
	AddNetworkMapChangeCallback(pubKey types.PublicKey, cb NetworkMapChangeCallback) error
}
