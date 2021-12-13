package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type VPNApiClient struct {
	addr string
}

type signReq struct {
	Message []byte `json:"message"`
}

type signResp struct {
	Signature []byte `json:"signature"`
}

func (c *VPNApiClient) Sign(message []byte) ([]byte, error) {
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(signReq{message}); err != nil {
		return nil, err
	}
	client := &http.Client{}
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
