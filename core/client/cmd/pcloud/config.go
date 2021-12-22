package main

type Config struct {
	ApiAddr string `json:"api_addr,omitempty"`
	Enc     struct {
		PublicKey  []byte `json:"public_key,omitempty"`
		PrivateKey []byte `json:"private_key,omitempty"`
	} `json:"encyption,omitempty"`
	Network struct {
		PublicKey  []byte `json:"public_key,omitempty"`
		PrivateKey []byte `json:"private_key,omitempty"`
		Config     []byte `json:"network,omitempty"`
	} `json:"network,omitempty"`
}
