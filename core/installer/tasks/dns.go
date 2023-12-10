package tasks

import (
	"context"
	"fmt"
	"net"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"

	"github.com/giolekva/pcloud/core/installer"
)

type dnsResolver struct {
	basicTask
	name     string
	expected []net.IP
	ctx      context.Context
	env      Env
	st       *state
}

func NewDNSResolverTask(
	name string,
	expected []net.IP,
	ctx context.Context,
	env Env,
	st *state,
) Task {
	return &dnsResolver{
		basicTask: basicTask{
			title: "Configure DNS",
		},
		name:     name,
		expected: expected,
		ctx:      ctx,
		env:      env,
		st:       st,
	}
}

func (t *dnsResolver) Start() {
	repo, err := t.st.ssClient.GetRepo("config")
	if err != nil {
		t.callDoneListeners(err)
		return
	}
	r := installer.NewRepoIO(repo, t.st.ssClient.Signer)
	{
		key, err := newDNSSecKey(t.env.Domain)
		if err != nil {
			t.callDoneListeners(err)
			return
		}
		out, err := r.Writer("dns-zone.yaml")
		if err != nil {
			t.callDoneListeners(err)
			return
		}
		defer out.Close()
		dnsZoneTmpl, err := template.New("config").Funcs(sprig.TxtFuncMap()).Parse(`
apiVersion: dodo.cloud.dodo.cloud/v1
kind: DNSZone
metadata:
  name: dns-zone
  namespace: {{ .namespace }}
spec:
  zone: {{ .zone }}
  privateIP: 10.1.0.1
  publicIPs:
{{ range .publicIPs }}
  - {{ .String }}
{{ end }}
  nameservers:
{{ range .publicIPs }}
  - {{ .String }}
{{ end }}
  dnssec:
    enabled: true
    secretName: dnssec-key
---
apiVersion: v1
kind: Secret
metadata:
  name: dnssec-key
  namespace: {{ .namespace }}
type: Opaque
data:
  basename: {{ .dnssec.Basename | b64enc }}
  key: {{ .dnssec.Key | toString | b64enc }}
  private: {{ .dnssec.Private | toString | b64enc }}
  ds: {{ .dnssec.DS | toString | b64enc }}
`)
		if err != nil {
			t.callDoneListeners(err)
			return
		}
		if err := dnsZoneTmpl.Execute(out, map[string]any{
			"namespace": t.env.Name,
			"zone":      t.env.Domain,
			"dnssec":    key,
			"publicIPs": t.st.publicIPs,
		}); err != nil {
			t.callDoneListeners(err)
			return
		}
		rootKust := installer.NewKustomization()
		rootKust.AddResources("dns-zone.yaml")
		if err := r.WriteKustomization("kustomization.yaml", rootKust); err != nil {
			t.callDoneListeners(err)
			return
		}
		r.CommitAndPush("configure dns zone")
	}

	gotExpectedIPs := func(actual []net.IP) bool {
		for _, a := range actual {
			found := false
			for _, e := range t.expected {
				if a.Equal(e) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	check := func(check Check) {
		addrs, err := net.LookupIP(t.name)
		if err == nil && gotExpectedIPs(addrs) {
			t.callDoneListeners(nil)
			return
		}
		select {
		case <-t.ctx.Done():
			t.callDoneListeners(fmt.Errorf("deadline exceeded"))
			return
		case <-time.After(5 * time.Second):
			check(check)
		}
	}
	check(check)
}

type Check func(ch Check)
