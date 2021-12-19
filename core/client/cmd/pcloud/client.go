package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/slackhq/nebula/cert"
	"golang.org/x/crypto/curve25519"
	"sigs.k8s.io/yaml"
)

type VPNClient interface {
	Sign(apiAddr string, message []byte) ([]byte, error)
	Join(apiAddr, hostname string, message, signature []byte) ([]byte, error)
}

type directVPNClient struct {
	addr string
}

func NewDirectVPNClient(addr string) VPNClient {
	return &directVPNClient{addr}
}

type signReq struct {
	Message []byte `json:"message"`
}

type signResp struct {
	Signature []byte `json:"signature"`
}

func (c *directVPNClient) Sign(apiAddr string, message []byte) ([]byte, error) {
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(signReq{message}); err != nil {
		return nil, err
	}
	client := &http.Client{
		// TODO(giolekva): remove, for some reason valid certificates are not accepted on gioui android.
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	r, err := client.Post(apiAddr+"/api/sign", "application/json", &data)
	if err != nil {
		return nil, err
	}
	resp := &signResp{}
	if err := json.NewDecoder(r.Body).Decode(resp); err != nil {
		return nil, err
	}
	return resp.Signature, nil
}

type joinReq struct {
	Message   []byte `json:"message"`
	Signature []byte `json:"signature"`
	Name      string `json:"name"`
	PublicKey []byte `json:"public_key"`
	IPCidr    string `json:"ip_cidr"`
}

type joinResp struct {
	cfgYamlB64 string
}

func (c *directVPNClient) Join(apiAddr, hostname string, message, signature []byte) ([]byte, error) {
	pubKey, privKey, err := x25519Keypair()
	if err != nil {
		return nil, err
	}
	req := joinReq{
		message,
		signature,
		hostname,
		cert.MarshalX25519PublicKey(pubKey),
		"111.0.0.13/24",
	}
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(req); err != nil {
		return nil, err
	}
	client := &http.Client{
		// TODO(giolekva): remove, for some reason valid certificates are not accepted on gioui android.
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	r, err := client.Post(apiAddr+"/api/join", "application/json", &data)
	if err != nil {
		return nil, err
	}
	var cfgYamlB bytes.Buffer
	_, err = io.Copy(&cfgYamlB,
		base64.NewDecoder(base64.StdEncoding, r.Body))
	if err != nil {
		return nil, err
	}
	cfgYaml := cfgYamlB.Bytes()
	var cfgMap map[string]interface{}
	if err := yaml.Unmarshal(cfgYaml, &cfgMap); err != nil {
		return nil, err
	}
	var pki map[string]interface{}
	var ok bool
	if pki, ok = cfgMap["pki"].(map[string]interface{}); !ok {
		panic("Must not reach")
	}
	pki["key"] = string(cert.MarshalX25519PrivateKey(privKey))
	return yaml.Marshal(cfgMap)
}

func x25519Keypair() ([]byte, []byte, error) {
	var pubkey, privkey [32]byte
	if _, err := io.ReadFull(rand.Reader, privkey[:]); err != nil {
		return nil, nil, err
	}
	curve25519.ScalarBaseMult(&pubkey, &privkey)
	return pubkey[:], privkey[:], nil
}
