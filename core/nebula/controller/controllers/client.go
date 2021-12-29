package controllers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
)

type createCAReq struct {
	Name string `json:"name"`
}

type createCAResp struct {
	PrivateKey  []byte `json:"private_key"`
	Certificate []byte `json:"certificate"`
}

func CreateCertificateAuthority(apiAddr, name string) ([]byte, []byte, error) {
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(createCAReq{name}); err != nil {
		return nil, nil, err
	}
	client := &http.Client{
		// TODO(giolekva): remove, for some reason valid certificates are not accepted on gioui android.
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Post(apiAddr+"/api/process/ca", "application/json", &data)
	if err != nil {
		return nil, nil, err
	}
	var ret createCAResp
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, nil, err
	}
	return ret.PrivateKey, ret.Certificate, nil
}

type signNodeReq struct {
	CAPrivateKey  []byte `json:"ca_private_key"`
	CACert        []byte `json:"ca_certificate"`
	NodeName      string `json:"node_name"`
	NodePublicKey []byte `json:"node_public_key,omitempty"`
	NodeIPCidr    string `json:"node_ip_cidr"`
}

type signNodeResp struct {
	PrivateKey  []byte `json:"private_key,omitempty"`
	Certificate []byte `json:"certificate"`
}

func SignNebulaNode(apiAddr string, caPrivateKey, caCert []byte, nodeName string, nodePublicKey []byte, nodeIp string) ([]byte, []byte, error) {
	req := signNodeReq{
		caPrivateKey,
		caCert,
		nodeName,
		nodePublicKey,
		nodeIp,
	}
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(req); err != nil {
		return nil, nil, err
	}
	client := &http.Client{
		// TODO(giolekva): remove, for some reason valid certificates are not accepted on gioui android.
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Post(apiAddr+"/api/process/node", "application/json", &data)
	if err != nil {
		return nil, nil, err
	}
	var ret signNodeResp
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, nil, err
	}
	return ret.PrivateKey, ret.Certificate, nil
}
