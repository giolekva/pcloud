package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/dns"
)

type Check func(ch Check) error

func SetupZoneTask(env installer.EnvConfig, mgr *installer.InfraAppManager, st *state) Task {
	ret := newSequentialParentTask(
		"Configure DNS",
		true,
		SetupDNSServer(env, st),
		WaitToPropagate(st.dnsClient, env.Domain, env.PublicIP),
	)
	ret.beforeStart = func() {
		st.infoListener(fmt.Sprintf("Generating DNS zone records for %s", env.Domain))
	}
	ret.afterDone = func() {
		st.infoListener("DNS zone records have been propagated.")
	}
	return ret
}

func join[T fmt.Stringer](items []T, sep string) string {
	var tmp []string
	for _, i := range items {
		tmp = append(tmp, i.String())
	}
	return strings.Join(tmp, ",")
}

func SetupDNSServer(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Start up DNS server", func() error {
		addressPool := fmt.Sprintf("%s-dns", env.Id)
		{
			app, err := installer.FindEnvApp(st.appsRepo, "env-dns")
			if err != nil {
				return err
			}
			instanceId := app.Slug()
			appDir := fmt.Sprintf("/apps/%s", instanceId)
			namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
			if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
				"addressPool":  addressPool,
				"inClusterIP":  env.Network.DNSInClusterIP.String(),
				"publicIP":     join(env.PublicIP, ","),
				"privateIP":    env.Network.Ingress.String(),
				"nameserverIP": join(env.NameserverIP, ","),
			}); err != nil {
				return err
			}
		}
		{
			app, err := installer.FindInfraApp(st.appsRepo, "dns-gateway")
			if err != nil {
				return err
			}
			cfg, err := st.infraAppManager.FindInstance("dns-gateway")
			if err != nil {
				return err
			}
			serversJSON, ok := cfg.Values["servers"]
			if !ok {
				serversJSON = []installer.EnvDNS{}
			}
			serversTmp, err := json.Marshal(serversJSON)
			if err != nil {
				return err
			}
			servers := []installer.EnvDNS{}
			if err := json.Unmarshal(serversTmp, &servers); err != nil {
				return err
			}
			servers = append(servers, installer.EnvDNS{
				env.Domain,
				env.Network.DNSInClusterIP.String(),
			})
			if _, err := st.infraAppManager.Update(app, "dns-gateway", map[string]any{
				"servers": servers,
			}); err != nil {
				return err
			}
		}
		{
			for {
				if _, err := st.dnsFetcher.Fetch(fmt.Sprintf("http://dns-api.%sdns.svc.cluster.local/records-to-publish", env.NamespacePrefix)); err != nil {
					time.Sleep(5 * time.Second)
				} else {
					break
				}
			}
		}
		return nil
	})
	return &t
}

func WaitToPropagate(
	client dns.Client,
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
			addrs, err := client.Lookup(name)
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
