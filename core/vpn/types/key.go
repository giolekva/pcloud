package types

import (
	"encoding/hex"
	"fmt"

	"tailscale.com/control/controlclient"
	"tailscale.com/types/key"
)

// Generates new private key.
func NewPrivateKey() PrivateKey {
	return PrivateKey(key.NewPrivate())
}

// Returns public coutnerpart of the given private key.
func (k PrivateKey) Public() PublicKey {
	return PublicKey(key.Private(k).Public())
}

func (k DiscoKey) Endpoint() string {
	discoHex := hex.EncodeToString(k[:])
	return fmt.Sprintf("%s%s", discoHex, controlclient.EndpointDiscoSuffix)
}
