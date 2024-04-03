package installer

import (
	"testing"
)

func TestAuthProxyEnabled(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	for _, app := range []string{"rpuppy", "Pi-hole", "url-shortener"} {
		a, err := r.Find(app)
		if err != nil {
			t.Fatal(err)
		}
		if a == nil {
			t.Fatal("returned app is nil")
		}
		d := Derived{
			Release: Release{
				Namespace: "foo",
			},
			Global: Values{
				PCloudEnvName:   "dodo",
				Id:              "id",
				ContactEmail:    "foo@bar.ge",
				Domain:          "bar.ge",
				PrivateDomain:   "p.bar.ge",
				PublicIP:        "1.2.3.4",
				NamespacePrefix: "id-",
			},
			Values: map[string]any{
				"network": map[string]any{
					"name":              "Public",
					"ingressClass":      "dodo-ingress-public",
					"certificateIssuer": "id-public",
					"domain":            "bar.ge",
				},
				"subdomain": "woof",
				"auth": map[string]any{
					"enabled": true,
					"groups":  "a,b",
				},
			},
		}
		rendered, err := a.Render(d)
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
		a, err := r.Find(app)
		if err != nil {
			t.Fatal(err)
		}
		if a == nil {
			t.Fatal("returned app is nil")
		}
		d := Derived{
			Release: Release{
				Namespace: "foo",
			},
			Global: Values{
				PCloudEnvName:   "dodo",
				Id:              "id",
				ContactEmail:    "foo@bar.ge",
				Domain:          "bar.ge",
				PrivateDomain:   "p.bar.ge",
				PublicIP:        "1.2.3.4",
				NamespacePrefix: "id-",
			},
			Values: map[string]any{
				"network": map[string]any{
					"name":              "Public",
					"ingressClass":      "dodo-ingress-public",
					"certificateIssuer": "id-public",
					"domain":            "bar.ge",
				},
				"subdomain": "woof",
				"auth": map[string]any{
					"enabled": false,
				},
			},
		}
		rendered, err := a.Render(d)
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
	a, err := r.Find("memberships")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("returned app is nil")
	}
	d := Derived{
		Release: Release{
			Namespace: "foo",
		},
		Global: Values{
			PCloudEnvName:   "dodo",
			Id:              "id",
			ContactEmail:    "foo@bar.ge",
			Domain:          "bar.ge",
			PrivateDomain:   "p.bar.ge",
			PublicIP:        "1.2.3.4",
			NamespacePrefix: "id-",
		},
		Values: map[string]any{},
	}
	rendered, err := a.Render(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rendered.Resources {
		t.Log(string(r))
	}
}

func TestGerrit(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	a, err := r.Find("gerrit")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("returned app is nil")
	}
	d := Derived{
		Release: Release{
			Namespace: "foo",
		},
		Global: Values{
			PCloudEnvName:   "dodo",
			Id:              "id",
			ContactEmail:    "foo@bar.ge",
			Domain:          "bar.ge",
			PrivateDomain:   "p.bar.ge",
			PublicIP:        "1.2.3.4",
			NamespacePrefix: "id-",
		},
		Values: map[string]any{
			"subdomain": "gerrit",
			"network": map[string]any{
				"name":             "Private",
				"ingressClass":     "id-ingress-private",
				"domain":           "p.bar.ge",
				"allocatePortAddr": "http://foo.bar/api/allocate",
			},
			"key": map[string]any{
				"public":  "foo",
				"private": "bar",
			},
			"sshPort": 22,
		},
	}
	rendered, err := a.Render(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rendered.Resources {
		t.Log(string(r))
	}
}
