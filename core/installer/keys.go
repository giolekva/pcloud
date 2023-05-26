package installer

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

func GenerateSSHKeys() (string, string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	privEnc, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", err
	}
	privPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privEnc,
		},
	)
	pubKey, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", "", err
	}
	return string(ssh.MarshalAuthorizedKey(pubKey)), string(privPem), nil
}
