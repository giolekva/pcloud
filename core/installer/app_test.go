package installer

import (
	_ "embed"
	"fmt"
	"net"
	"testing"

	"cuelang.org/go/cue/errors"
)

var (
	env = EnvConfig{
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

	infraNetworks = []InfraNetwork{
		{
			Name:               "Public",
			IngressClass:       fmt.Sprintf("%s-ingress-public", env.InfraName),
			CertificateIssuer:  fmt.Sprintf("%s-public", env.Id),
			AllocatePortAddr:   fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/allocate", env.InfraName),
			ReservePortAddr:    fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/reserve", env.InfraName),
			DeallocatePortAddr: fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/remove", env.InfraName),
		},
	}

	networks = []Network{
		{
			Name:               "Public",
			IngressClass:       fmt.Sprintf("%s-ingress-public", env.InfraName),
			CertificateIssuer:  fmt.Sprintf("%s-public", env.Id),
			Domain:             env.Domain,
			AllocatePortAddr:   fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/allocate", env.InfraName),
			ReservePortAddr:    fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/reserve", env.InfraName),
			DeallocatePortAddr: fmt.Sprintf("http://port-allocator.%s-ingress-public.svc.cluster.local/api/remove", env.InfraName),
		},
		{
			Name:               "Private",
			IngressClass:       fmt.Sprintf("%s-ingress-private", env.Id),
			Domain:             env.PrivateDomain,
			AllocatePortAddr:   fmt.Sprintf("http://port-allocator.%s-ingress-private.svc.cluster.local/api/allocate", env.Id),
			ReservePortAddr:    fmt.Sprintf("http://port-allocator.%s-ingress-private.svc.cluster.local/api/reserve", env.Id),
			DeallocatePortAddr: fmt.Sprintf("http://port-allocator.%s-ingress-private.svc.cluster.local/api/remove", env.Id),
		},
	}

	clusters = []Cluster{
		{
			Name: "default",
		},
		{
			Name:             "io",
			IngressClassName: "io",
			Kubeconfig:       "kubeconfig",
		},
	}
)

func TestAuthProxyEnabled(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	for _, app := range []string{"rpuppy"} {
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
		rendered, err := a.Render(release, env, networks, nil, values, nil, nil)
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
	for _, app := range []string{"rpuppy"} {
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
		rendered, err := a.Render(release, env, networks, nil, values, nil, nil)
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
		"network":    "Public",
		"authGroups": "foo,bar",
	}
	rendered, err := a.Render(release, env, networks, nil, values, nil, nil)
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
	rendered, err := a.Render(release, env, networks, nil, values, nil, nil)
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
	rendered, err := a.Render(release, env, networks, nil, values, nil, nil)
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
	rendered, err := a.Render(release, infra, infraNetworks, values, nil)
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
	rendered, err := a.Render(release, env, networks, nil, values, nil, nil)
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
	rendered, err := app.Render(release, env, networks, nil, values, nil, nil)
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

func TestAppPackagesRemoteCluster(t *testing.T) {
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
			"enabled": false,
		},
		"cluster": "io",
	}
	rendered, err := app.Render(release, env, networks, clusters, values, nil, nil)
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
	rendered, err := app.Render(release, infra, infraNetworks, values, nil)
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

var dodoAppRemoteClusterCue = `
app: {
	type: "golang:1.22.0"
	run: "main.go"
	ingress: {
		network: "private"
		subdomain: "testapp"
		auth: enabled: false
	}
	dev: {
		enabled: false
	}
    cluster: "io"
}`

func TestDodoAppRemoteCluster(t *testing.T) {
	app, err := NewDodoApp([]byte(dodoAppRemoteClusterCue))
	if err != nil {
		for _, e := range errors.Errors(err) {
			t.Log(e)
		}
		t.Fatal(err)
	}

	release := Release{
		Namespace:     "foo",
		AppInstanceId: "foo-bar",
		RepoAddr:      "ssh://192.168.100.210:22/config",
		AppDir:        "/foo/bar",
	}
	keyGen := testKeyGen{}
	r, err := app.Render(release, env, networks, clusters, map[string]any{
		"repoAddr":       "1",
		"repoPublicAddr": "2",
		"managerAddr":    "3",
		"appId":          "4",
		"branch":         "5",
		"sshPrivateKey":  "6",
	}, nil, keyGen)
	if err != nil {
		for _, e := range errors.Errors(err) {
			for _, f := range errors.Errors(e) {
				for _, g := range errors.Errors(f) {
					t.Log(g)
				}
			}
		}
		t.Fatal(err)
	}
	t.Log(string(r.Raw))
}

var dodoAppDevDisabledCue = `
app: {
	type: "golang:1.22.0"
	run: "main.go"
	ingress: {
		network: "private"
		subdomain: "testapp"
		auth: enabled: false
	}
	dev: {
		enabled: false
	}
}`

var dodoAppDevEnabledCue = `
app: {
	type: "golang:1.20.0"
	run: "main.go"
	ingress: {
		network: "private"
		subdomain: "testapp"
		auth: enabled: false
	}
	dev: {
		enabled: true
		username: "gio"
	}
    volumes: {
      data: size: "5Gi"
    }
    postgresql: {
      db: size: "10Gi"
    }
}`

func TestDodoAppDevDisabled(t *testing.T) {
	app, err := NewDodoApp([]byte(dodoAppDevDisabledCue))
	if err != nil {
		for _, e := range errors.Errors(err) {
			t.Log(e)
		}
		t.Fatal(err)
	}

	release := Release{
		Namespace:     "foo",
		AppInstanceId: "foo-bar",
		RepoAddr:      "ssh://192.168.100.210:22/config",
		AppDir:        "/foo/bar",
	}
	keyGen := testKeyGen{}
	r, err := app.Render(release, env, networks, nil, map[string]any{
		"repoAddr":       "",
		"repoPublicAddr": "",
		"managerAddr":    "",
		"appId":          "",
		"branch":         "",
		"sshPrivateKey":  "",
	}, nil, keyGen)
	if err != nil {
		for _, e := range errors.Errors(err) {
			for _, f := range errors.Errors(e) {
				for _, g := range errors.Errors(f) {
					t.Log(g)
				}
			}
		}
		t.Fatal(err)
	}
	t.Log(string(r.Raw))
}

func TestDodoAppDevEnabled(t *testing.T) {
	app, err := NewDodoApp([]byte(dodoAppDevEnabledCue))
	if err != nil {
		for _, e := range errors.Errors(err) {
			t.Log(e)
		}
		t.Fatal(err)
	}

	release := Release{
		Namespace:     "foo",
		AppInstanceId: "foo-bar",
		RepoAddr:      "ssh://192.168.100.210:22/config",
		AppDir:        "/foo/bar",
	}
	keyGen := testKeyGen{}
	r, err := app.Render(release, env, networks, nil, map[string]any{
		"repoAddr":       "",
		"repoPublicAddr": "",
		"managerAddr":    "",
		"appId":          "",
		"branch":         "",
		"sshPrivateKey":  "",
		"username":       "",
	}, nil, keyGen)
	if err != nil {
		for _, e := range errors.Errors(err) {
			t.Log(e)
		}
		t.Fatal(err)
	}
	t.Log(string(r.Raw))
}

func TestDodoAppInstance(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	a, err := FindEnvApp(r, "dodo-app-instance")
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
		"repoAddr":         "",
		"repoPublicAddr":   "",
		"repoHost":         "",
		"branch":           "",
		"gitRepoPublicKey": "",
		"username":         "",
	}
	rendered, err := a.Render(release, env, networks, nil, values, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rendered.Resources {
		t.Log(string(r))
	}
}

func TestDodoApp(t *testing.T) {
	contents, err := valuesTmpls.ReadFile("values-tmpl/dodo-app.cue")
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
	t.Log(app.Schema())
}

func TestAppVirtualMachine(t *testing.T) {
	contents, err := valuesTmpls.ReadFile("values-tmpl/virtual-machine.cue")
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
		"name":     "foo",
		"username": "bar",
		"cpuCores": 1,
		"memory":   "1Gi",
	}
	rendered, err := app.Render(release, env, networks, nil, values, nil, nil)
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
