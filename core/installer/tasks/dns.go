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

type Check func(ch Check) error

func SetupZoneTask(env Env, ingressIP net.IP, st *state) Task {
	return newSequentialParentTask(
		"Configure DNS",
		true,
		CreateZoneRecords(env.Domain, st.publicIPs, ingressIP, env, st),
		WaitToPropagate(env.Domain, st.publicIPs),
	)
}

func CreateZoneRecords(
	name string,
	expected []net.IP,
	ingressIP net.IP,
	env Env,
	st *state,
) Task {
	t := newLeafTask("Generate and publish DNS records", func() error {
		repo, err := st.ssClient.GetRepo("config")
		if err != nil {
			return err
		}
		r := installer.NewRepoIO(repo, st.ssClient.Signer)
		{
			key, err := newDNSSecKey(env.Domain)
			if err != nil {
				return err
			}
			out, err := r.Writer("dns-zone.yaml")
			if err != nil {
				return err
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
  privateIP: {{ .ingressIP }}
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
				return err
			}
			if err := dnsZoneTmpl.Execute(out, map[string]any{
				"namespace": env.Name,
				"zone":      env.Domain,
				"dnssec":    key,
				"publicIPs": st.publicIPs,
				"ingressIP": ingressIP.String(),
			}); err != nil {
				return err
			}
			rootKust, err := r.ReadKustomization("kustomization.yaml")
			if err != nil {
				return err
			}
			rootKust.AddResources("dns-zone.yaml")
			if err := r.WriteKustomization("kustomization.yaml", *rootKust); err != nil {
				return err
			}
			if err := r.CommitAndPush("configure dns zone"); err != nil {
				return err
			}
		}
		return nil
	})
	return &t
}

func WaitToPropagate(
	name string,
	expected []net.IP,
) Task {
	t := newLeafTask("Wait to propagate", func() error {
		ctx := context.TODO()
		gotExpectedIPs := func(actual []net.IP) bool {
			for _, a := range actual {
				found := false
				for _, e := range expected {
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
		check := func(check Check) error {
			addrs, err := net.LookupIP(name)
			fmt.Printf("DNS LOOKUP: %+v\n", addrs)
			if err == nil && gotExpectedIPs(addrs) {
				return err
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
				return check(check)
			}
		}
		return check(check)
	})
	return &t
}
