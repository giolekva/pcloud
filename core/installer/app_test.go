package installer

import (
	"net"
	"testing"
)

var env = EnvConfig{
	InfraName:       "dodo",
	Id:              "id",
	ContactEmail:    "foo@bar.ge",
	Domain:          "bar.ge",
	PrivateDomain:   "p.bar.ge",
	PublicIP:        []net.IP{net.ParseIP("1.2.3.4")},
	NameserverIP:    []net.IP{net.ParseIP("1.2.3.4")},
	NamespacePrefix: "id-",
	Network: EnvNetwork{
		DNS:            net.ParseIP("1.1.1.1"),
		DNSInClusterIP: net.ParseIP("2.2.2.2"),
		Ingress:        net.ParseIP("3.3.3.3"),
		Headscale:      net.ParseIP("4.4.4.4"),
		ServicesFrom:   net.ParseIP("5.5.5.5"),
		ServicesTo:     net.ParseIP("6.6.6.6"),
	},
}

func TestAuthProxyEnabled(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	for _, app := range []string{"rpuppy", "pi-hole", "url-shortener"} {
		a, err := FindEnvApp(r, app)
		if err != nil {
			t.Fatal(err)
		}
		if a == nil {
			t.Fatal("returned app is nil")
		}
		release := Release{
			Namespace: "foo",
		}
		values := map[string]any{
			"network":   "Public",
			"subdomain": "woof",
			"auth": map[string]any{
				"enabled": true,
				"groups":  "a,b",
			},
		}
		rendered, err := a.Render(release, env, values)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rendered.Resources {
			t.Log(string(r))
		}
	}
}

func TestAuthProxyDisabled(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	for _, app := range []string{"rpuppy", "pi-hole", "url-shortener"} {
		a, err := FindEnvApp(r, app)
		if err != nil {
			t.Fatal(err)
		}
		if a == nil {
			t.Fatal("returned app is nil")
		}
		release := Release{
			Namespace: "foo",
		}
		values := map[string]any{
			"network":   "Public",
			"subdomain": "woof",
			"auth": map[string]any{
				"enabled": false,
			},
		}
		rendered, err := a.Render(release, env, values)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rendered.Resources {
			t.Log(string(r))
		}
	}
}

func TestGroupMemberships(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	a, err := FindEnvApp(r, "memberships")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("returned app is nil")
	}
	release := Release{
		Namespace: "foo",
	}
	values := map[string]any{
		"authGroups": "foo,bar",
	}
	rendered, err := a.Render(release, env, values)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rendered.Resources {
		t.Log(string(r))
	}
}

func TestGerrit(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	a, err := FindEnvApp(r, "gerrit")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("returned app is nil")
	}
	release := Release{
		Namespace: "foo",
	}
	values := map[string]any{
		"subdomain": "gerrit",
		"network":   "Private",
		"key": map[string]any{
			"public":  "foo",
			"private": "bar",
		},
		"sshPort": 22,
	}
	rendered, err := a.Render(release, env, values)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rendered.Resources {
		t.Log(string(r))
	}
}

func TestJenkins(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	a, err := FindEnvApp(r, "jenkins")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("returned app is nil")
	}
	release := Release{
		Namespace: "foo",
	}
	values := map[string]any{
		"subdomain": "jenkins",
		"network":   "Private",
	}
	rendered, err := a.Render(release, env, values)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rendered.Resources {
		t.Log(string(r))
	}
}

func TestIngressPublic(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	a, err := FindInfraApp(r, "ingress-public")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("returned app is nil")
	}
	release := Release{
		Namespace: "foo",
	}
	infra := InfraConfig{
		Name:                 "dodo",
		PublicIP:             []net.IP{net.ParseIP("1.2.3.4")},
		InfraNamespacePrefix: "id-",
		InfraAdminPublicKey:  []byte("foo"),
	}
	values := map[string]any{
		"sshPrivateKey": "private",
	}
	rendered, err := a.Render(release, infra, values)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rendered.Resources {
		t.Log(string(r))
	}
}

func TestPrivateNetwork(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	a, err := FindEnvApp(r, "private-network")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("returned app is nil")
	}
	release := Release{
		Namespace: "foo",
	}
	values := map[string]any{
		"privateNetwork": map[string]any{
			"hostname": "foo",
			"username": "bar",
			"ipSubnet": "123123",
		},
		"sshPrivateKey": "private",
	}
	rendered, err := a.Render(release, env, values)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rendered.Resources {
		t.Log(string(r))
	}
}

func TestAppPackages(t *testing.T) {
	contents, err := valuesTmpls.ReadFile("values-tmpl/rpuppy.cue")
	if err != nil {
		t.Fatal(err)
	}
	app, err := NewCueEnvApp(CueAppData{
		"base.cue":   []byte(cueBaseConfig),
		"app.cue":    []byte(contents),
		"global.cue": []byte(cueEnvAppGlobal),
	})
	if err != nil {
		t.Fatal(err)
	}
	release := Release{
		Namespace: "foo",
	}
	values := map[string]any{
		"network":   "Public",
		"subdomain": "woof",
		"auth": map[string]any{
			"enabled": true,
			"groups":  "a,b",
		},
	}
	rendered, err := app.Render(release, env, values)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rendered.Resources {
		t.Log(string(r))
	}
	for _, r := range rendered.Data {
		t.Log(string(r))
	}
}

func TestDNSGateway(t *testing.T) {
	contents, err := valuesTmpls.ReadFile("values-tmpl/dns-gateway.cue")
	if err != nil {
		t.Fatal(err)
	}
	app, err := NewCueInfraApp(CueAppData{
		"base.cue":   []byte(cueBaseConfig),
		"app.cue":    []byte(contents),
		"global.cue": []byte(cueInfraAppGlobal),
	})
	if err != nil {
		t.Fatal(err)
	}
	release := Release{
		Namespace:     "foo",
		AppInstanceId: "dns-gateway",
		RepoAddr:      "ssh://192.168.100.210:22/config",
		AppDir:        "/infrastructure/gns-gateway",
	}
	infra := InfraConfig{
		Name:                 "dodo",
		PublicIP:             []net.IP{net.ParseIP("135.181.48.180"), net.ParseIP("65.108.39.172")},
		InfraNamespacePrefix: "dodo-",
		InfraAdminPublicKey:  []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC/ZRj0QJ0j+3udh0ANN9mJyEzrATZIOAHfNikDMpSHqrVbPZqpeHGbdYrSksCvXPXfissIZoYU4CCXX007jY0W6e1mPf1nObYh2eUT1dHo/8UtGaf9nYk+kEGU/k3utN4Uzkxa13IFh9pYERX+o0Ad3X5wh0vi5hjOBAJVKOCD9d3aipeR9piUb+qrkFDXf9fozMFn7D9nALkpJBVuGxwl/76f8K6hRxBEmPqZwIMfklzX15nRdLEcsGFJpYLYXsonbr1P3moMJFBBbQFv6M6JO9rrwA+swXpWMoScI7m/nziSEPLAb+ziv+/OyhqzeC9CQner73V0m8+2DmtcgTuSe1qHRtOScPyIjBfxoXaUx1IUkgq1NXt8k+EBO2mxnVpKdyDCvwT1Tb7088P8f8cSLtUOmUdEiAhB8bfQFprzm2KrlufenfhMvdvQPU4VfWlkQ4smLYt2yVaaXoxZMy5yD3X6LFurNXwee/Gn6di+DWqsASAOsmpsNgSCGhT8wxM= lekva@gl-mbp-m1-max.local"),
	}
	values := map[string]any{
		"servers": []EnvDNS{EnvDNS{"v1.dodo.cloud", "10.0.1.2"}},
	}
	rendered, err := app.Render(release, infra, values)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rendered.Resources {
		t.Log(string(r))
	}
	for _, r := range rendered.Data {
		t.Log(string(r))
	}
}
