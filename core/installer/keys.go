package installer

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

type KeyPair struct {
	Public  string
	Private string
}

func NewSSHKeyPair() (KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, err
	}
	privEnc, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return KeyPair{}, err
	}
	privPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privEnc,
		},
	)
	pubKey, err := ssh.NewPublicKey(pub)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{
		Public:  string(ssh.MarshalAuthorizedKey(pubKey)),
		Private: string(privPem),
	}, nil
}
