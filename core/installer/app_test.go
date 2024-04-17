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
	values := map[string]any{}
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
