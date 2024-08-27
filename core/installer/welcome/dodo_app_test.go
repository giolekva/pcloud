package welcome

import (
	"testing"
)

func TestCreateDevBranch(t *testing.T) {
	cfg := []byte(`
app: {
	type: "golang:1.22.0"
	run: "main.go"
	ingress: {
		network: "private"
		subdomain: "testapp"
		auth: enabled: false
	}
}`)
	network, newCfg, err := createDevBranchAppConfig(cfg, "foo", "bar")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(network)
	t.Log(string(newCfg))
}
