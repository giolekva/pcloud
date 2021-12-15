package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
)

type VPNClient interface {
	Address() string
	Sign(message []byte) ([]byte, error)
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
		fmt.Println(111111)
		return nil, err
	}
	resp := &signResp{}
	if err := json.NewDecoder(r.Body).Decode(resp); err != nil {
		return nil, err
	}
	return resp.Signature, nil
}
