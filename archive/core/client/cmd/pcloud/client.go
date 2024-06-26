package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/slackhq/nebula/cert"
	"golang.org/x/crypto/curve25519"
	"sigs.k8s.io/yaml"
)

type VPNClient interface {
	Sign(apiAddr string, message []byte) ([]byte, error)
	Join(apiAddr, hostname string, publicKey, privateKey []byte, message, signature []byte) ([]byte, error)
	Approve(apiAddr, hostname, ipCidr string, encPublicKey, netPublicKey []byte) error
	Get(apiAddr, hostname string, encPrivateKey *rsa.PrivateKey, netPrivateKey []byte) ([]byte, error)
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

func (c *directVPNClient) Join(apiAddr, hostname string, publicKey, privateKey []byte, message, signature []byte) ([]byte, error) {
	req := joinReq{
		message,
		signature,
		hostname,
		cert.MarshalX25519PublicKey(publicKey),
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
	pki["key"] = string(cert.MarshalX25519PrivateKey(privateKey))
	return yaml.Marshal(cfgMap)
}

type approveReq struct {
	EncPublicKey []byte `json:"enc_public_key"`
	Name         string `json:"name"`
	NetPublicKey []byte `json:"net_public_key"`
	IPCidr       string `json:"ip_cidr"`
}

func (c *directVPNClient) Approve(apiAddr, hostname, ipCidr string, encPublicKey, netPublicKey []byte) error {
	req := approveReq{
		encPublicKey,
		hostname,
		cert.MarshalX25519PublicKey(netPublicKey),
		ipCidr,
	}
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(req); err != nil {
		return err
	}
	client := &http.Client{
		// TODO(giolekva): remove, for some reason valid certificates are not accepted on gioui android.
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	_, err := client.Post(apiAddr+"/api/approve", "application/json", &data)
	return err
}

type getResp struct {
	Key   []byte `json:"key"`
	Nonce []byte `json:"nonce"`
	Data  []byte `json:"data"`
}

func (c *directVPNClient) Get(apiAddr, hostname string, encPrivateKey *rsa.PrivateKey, netPrivateKey []byte) ([]byte, error) {
	client := &http.Client{
		// TODO(giolekva): remove, for some reason valid certificates are not accepted on gioui android.
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	r, err := client.Get(apiAddr + "/api/get/" + hostname)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	var resp getResp
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	// TODO(giolekva): encrypt key and nonce together
	key, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, encPrivateKey, resp.Key, []byte(""))
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	nonce, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, encPrivateKey, resp.Nonce, []byte(""))
	if err != nil {
		fmt.Println(1123123)
		fmt.Println(err.Error())
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	cfgYaml, err := aesgcm.Open(nil, nonce, resp.Data, nil)
	if err != nil {
		fmt.Println(22222)
		fmt.Println(err.Error())
		return nil, err
	}
	var cfgMap map[string]interface{}
	if err := yaml.Unmarshal(cfgYaml, &cfgMap); err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	var pki map[string]interface{}
	var ok bool
	if pki, ok = cfgMap["pki"].(map[string]interface{}); !ok {
		panic("Must not reach")
	}
	pki["key"] = string(cert.MarshalX25519PrivateKey(netPrivateKey))
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
