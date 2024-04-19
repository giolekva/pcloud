package installer

import (
	"net"
	"testing"
)

func TestAuthProxyEnabled(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	for _, app := range []string{"rpuppy", "Pi-hole", "url-shortener"} {
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
		env := AppEnvConfig{
			InfraName:       "dodo",
			Id:              "id",
			ContactEmail:    "foo@bar.ge",
			Domain:          "bar.ge",
			PrivateDomain:   "p.bar.ge",
			PublicIP:        []net.IP{net.ParseIP("1.2.3.4")},
			NamespacePrefix: "id-",
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
	for _, app := range []string{"rpuppy", "Pi-hole", "url-shortener"} {
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
		env := AppEnvConfig{
			InfraName:       "dodo",
			Id:              "id",
			ContactEmail:    "foo@bar.ge",
			Domain:          "bar.ge",
			PrivateDomain:   "p.bar.ge",
			PublicIP:        []net.IP{net.ParseIP("1.2.3.4")},
			NamespacePrefix: "id-",
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
	env := AppEnvConfig{
		InfraName:       "dodo",
		Id:              "id",
		ContactEmail:    "foo@bar.ge",
		Domain:          "bar.ge",
		PrivateDomain:   "p.bar.ge",
		PublicIP:        []net.IP{net.ParseIP("1.2.3.4")},
		NamespacePrefix: "id-",
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
	env := AppEnvConfig{
		InfraName:       "dodo",
		Id:              "id",
		ContactEmail:    "foo@bar.ge",
		Domain:          "bar.ge",
		PrivateDomain:   "p.bar.ge",
		PublicIP:        []net.IP{net.ParseIP("1.2.3.4")},
		NamespacePrefix: "id-",
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
	env := AppEnvConfig{
		InfraName:       "dodo",
		Id:              "id",
		ContactEmail:    "foo@bar.ge",
		Domain:          "bar.ge",
		PrivateDomain:   "p.bar.ge",
		PublicIP:        []net.IP{net.ParseIP("1.2.3.4")},
		NamespacePrefix: "id-",
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
	env := InfraConfig{
		Name:                 "dodo",
		PublicIP:             []net.IP{net.ParseIP("1.2.3.4")},
		InfraNamespacePrefix: "id-",
		InfraAdminPublicKey:  []byte("foo"),
	}
	values := map[string]any{
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
	env := AppEnvConfig{
		InfraName:       "dodo",
		Id:              "id",
		ContactEmail:    "foo@bar.ge",
		Domain:          "bar.ge",
		PrivateDomain:   "p.bar.ge",
		PublicIP:        []net.IP{net.ParseIP("1.2.3.4")},
		NamespacePrefix: "id-",
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
		"base.cue": []byte(cueBaseConfig),
		"app.cue":  []byte(contents),
	})
	if err != nil {
		t.Fatal(err)
	}
	release := Release{
		Namespace: "foo",
	}
	env := AppEnvConfig{
		InfraName:       "dodo",
		Id:              "id",
		ContactEmail:    "foo@bar.ge",
		Domain:          "bar.ge",
		PrivateDomain:   "p.bar.ge",
		PublicIP:        []net.IP{net.ParseIP("1.2.3.4")},
		NamespacePrefix: "id-",
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
