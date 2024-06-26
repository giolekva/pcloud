package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"time"

	"github.com/slackhq/nebula/cert"
	"golang.org/x/crypto/curve25519"
)

type CertificateAuthority struct {
	PrivateKey  []byte `json:"private_key"`
	Certificate []byte `json:"certificate"`
}

func CreateCertificateAuthority(name string) (*CertificateAuthority, error) {
	t := time.Now().Add(time.Duration(-1 * time.Second))
	rawPub, rawPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	nc := cert.NebulaCertificate{
		Details: cert.NebulaCertificateDetails{
			Name:      name,
			NotBefore: t,
			NotAfter:  t.Add(time.Duration(8760 * time.Hour)),
			PublicKey: rawPub,
			IsCA:      true,
		},
	}
	if err := nc.Sign(rawPriv); err != nil {
		return nil, err
	}
	certSerialized, err := nc.MarshalToPEM()
	if err != nil {
		return nil, err
	}
	privKeySerialzied := cert.MarshalEd25519PrivateKey(rawPriv)
	return &CertificateAuthority{
		privKeySerialzied,
		certSerialized,
	}, nil
}

type NebulaNode struct {
	PrivateKey  []byte `json:"private_key,omitempty"`
	Certificate []byte `json:"certificate"`
}

func SignNebulaNode(rawCAPrivateKey []byte, rawCACert []byte, nodeName string, nodePublicKey []byte, ip *net.IPNet) (*NebulaNode, error) {
	caKey, _, err := cert.UnmarshalEd25519PrivateKey(rawCAPrivateKey)
	if err != nil {
		return nil, err
	}
	caCert, _, err := cert.UnmarshalNebulaCertificateFromPEM(rawCACert)
	if err != nil {
		return nil, err
	}

	if err := caCert.VerifyPrivateKey(caKey); err != nil {
		return nil, err
	}
	issuer, err := caCert.Sha256Sum()
	if err != nil {
		return nil, err
	}
	if caCert.Expired(time.Now()) {
		return nil, errors.New("ca certificate is expired")
	}
	var pub, priv []byte
	if nodePublicKey != nil {
		var err error
		pub, _, err = cert.UnmarshalX25519PublicKey(nodePublicKey)
		if err != nil {
			return nil, err
		}
	} else {
		var rawPriv []byte
		var err error
		pub, rawPriv, err = x25519Keypair()
		if err != nil {
			return nil, err
		}
		priv = cert.MarshalX25519PrivateKey(rawPriv)
	}
	t := time.Now().Add(time.Duration(-1 * time.Second))
	nc := cert.NebulaCertificate{
		Details: cert.NebulaCertificateDetails{
			Name:      nodeName,
			Ips:       []*net.IPNet{ip},
			NotBefore: t,
			NotAfter:  caCert.Details.NotAfter.Add(time.Duration(-1 * time.Second)),
			PublicKey: pub,
			IsCA:      false,
			Issuer:    issuer,
		},
	}
	if err := nc.CheckRootConstrains(caCert); err != nil {
		return nil, err
	}
	if err := nc.Sign(caKey); err != nil {
		return nil, err
	}
	certSerialized, err := nc.MarshalToPEM()
	if err != nil {
		return nil, err
	}
	return &NebulaNode{
		PrivateKey:  priv,
		Certificate: certSerialized,
	}, nil
}

func x25519Keypair() ([]byte, []byte, error) {
	var pubkey, privkey [32]byte
	if _, err := io.ReadFull(rand.Reader, privkey[:]); err != nil {
		return nil, nil, err
	}
	curve25519.ScalarBaseMult(&pubkey, &privkey)
	return pubkey[:], privkey[:], nil
}
