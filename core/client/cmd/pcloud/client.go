package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"golang.org/x/crypto/curve25519"
)

type VPNClient interface {
	Address() string
	Sign(message []byte) ([]byte, error)
	Join(apiAddr string, message, signature []byte) (interface{}, error)
}

type directVPNClient struct {
	addr string
}

func NewDirectVPNClient(addr string) VPNClient {
	return &directVPNClient{addr}
}

func (c *directVPNClient) Address() string {
	return c.addr
}

type signReq struct {
	Message []byte `json:"message"`
}

type signResp struct {
	Signature []byte `json:"signature"`
}

func (c *directVPNClient) Sign(message []byte) ([]byte, error) {
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
	r, err := client.Post(c.addr+"/api/sign", "application/json", &data)
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
}

func (c *directVPNClient) Join(apiAddr string, message, signature []byte) (interface{}, error) {
	if c.addr != "" {
		return nil, errors.New("Already joined")
	}
	c.addr = apiAddr
	pubKey, _, err := x25519Keypair()
	if err != nil {
		return nil, err
	}
	req := joinReq{
		message,
		signature,
		"test",
		pubKey,
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
	r, err := client.Post(c.addr+"/api/join", "application/json", &data)
	if err != nil {
		return nil, err
	}
	resp := &joinResp{}
	if err := json.NewDecoder(r.Body).Decode(resp); err != nil {
		return nil, err
	}
	return nil, nil
}

func x25519Keypair() ([]byte, []byte, error) {
	var pubkey, privkey [32]byte
	if _, err := io.ReadFull(rand.Reader, privkey[:]); err != nil {
		return nil, nil, err
	}
	curve25519.ScalarBaseMult(&pubkey, &privkey)
	return pubkey[:], privkey[:], nil
}
