package types

import "tailscale.com/types/key"

// Generates new private key.
func NewPrivateKey() PrivateKey {
	return PrivateKey(key.NewPrivate())
}

// Returns public coutnerpart of the given private key.
func (k PrivateKey) Public() PublicKey {
	return PublicKey(key.Private(k).Public())
}
