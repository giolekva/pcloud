package installer

import (
	"github.com/charmbracelet/keygen"
)

func NewSSHKeyPair(path string) (*keygen.KeyPair, error) {
	return keygen.New(path, keygen.WithKeyType(keygen.Ed25519))
}

func NewECDSASSHKeyPair(path string) (*keygen.KeyPair, error) {
	return keygen.New(path, keygen.WithKeyType(keygen.ECDSA))
}
