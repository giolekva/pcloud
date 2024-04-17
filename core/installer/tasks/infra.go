package tasks

import (
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/miekg/dns"

	"github.com/giolekva/pcloud/core/installer"
)

var initGroups = []string{"admin"}

func CreateRepoClient(env Env, st *state) Task {
	t := newLeafTask("Create repo client", func() error {
		repo, err := st.ssClient.GetRepo("config")
		if err != nil {
			return err
		}
		r, err := installer.NewRepoIO(repo, st.ssClient.Signer)
		if err != nil {
			return err
		}
		appManager, err := installer.NewAppManager(r, st.nsCreator)
		if err != nil {
			return err
		}
		st.appManager = appManager
		st.appsRepo = installer.NewInMemoryAppRepository(installer.CreateAllApps())
		return nil
	})
	t.beforeStart = func() {
		st.infoListener("Setting up core infrastructure services.")
	}
	return &t
}

func SetupInfra(env Env, startIP net.IP, st *state) Task {
	return newConcurrentParentTask(
		"Setup core services",
		true,
		SetupNetwork(env, startIP, st),
		SetupCertificateIssuers(env, st),
		SetupAuth(env, st),
		SetupGroupMemberships(env, st),
		SetupHeadscale(env, startIP, st),
		SetupWelcome(env, st),
		SetupAppStore(env, st),
	)
}

func CommitEnvironmentConfiguration(env Env, st *state) Task {
	t := newLeafTask("commit config", func() error {
		repo, err := st.ssClient.GetRepo("config")
		if err != nil {
			return err
		}
		r, err := installer.NewRepoIO(repo, st.ssClient.Signer)
		if err != nil {
			return err
		}
		r.Atomic(func(r installer.RepoFS) (string, error) {
			{
				// TODO(giolekva): private domain can be configurable as well
				config := installer.AppEnvConfig{
					Id:              env.Name,
					InfraName:       env.PCloudEnvName,
					Domain:          env.Domain,
					PrivateDomain:   fmt.Sprintf("p.%s", env.Domain),
					ContactEmail:    env.ContactEmail,
					PublicIP:        st.publicIPs,
					NamespacePrefix: fmt.Sprintf("%s-", env.Name),
				}
				if err := installer.WriteYaml(r, "config.yaml", config); err != nil {
					return "", err
				}
			}
			out, err := r.Writer("pcloud-charts.yaml")
			if err != nil {
				return "", err
			}
			defer out.Close()
			_, err = fmt.Fprintf(out, `
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: pcloud
  namespace: %s
spec:
  interval: 1m0s
  url: https://github.com/giolekva/pcloud
  ref:
    branch: app-config-test
`, env.Name)
			if err != nil {
				return "", err
			}
			rootKust, err := installer.ReadKustomization(r, "kustomization.yaml")
			if err != nil {
				return "", err
			}
			rootKust.AddResources("pcloud-charts.yaml")
			if err := installer.WriteYaml(r, "kustomization.yaml", rootKust); err != nil {
				return "", err
			}
			return "configure charts repo", nil
		})
		return nil
	})
	return &t
}

type firstAccount struct {
	Created bool     `json:"created"`
	Groups  []string `json:"groups"`
}

func ConfigureFirstAccount(env Env, st *state) Task {
	t := newLeafTask("Configure first account settings", func() error {
		repo, err := st.ssClient.GetRepo("config")
		if err != nil {
			return err
		}
		r, err := installer.NewRepoIO(repo, st.ssClient.Signer)
		if err != nil {
			return err
		}
		return r.Atomic(func(r installer.RepoFS) (string, error) {
			fa := firstAccount{false, initGroups}
			if err := installer.WriteYaml(r, "first-account.yaml", fa); err != nil {
				return "", err
			}
			return "first account membership configuration", nil
		})
	})
	return &t
}

func SetupNetwork(env Env, startIP net.IP, st *state) Task {
	t := newLeafTask("Setup private and public networks", func() error {
		startAddr, err := netip.ParseAddr(startIP.String())
		if err != nil {
			return err
		}
		if !startAddr.Is4() {
			return fmt.Errorf("Expected IPv4, got %s instead", startAddr)
		}
		addr := startAddr.AsSlice()
		if addr[3] != 0 {
			return fmt.Errorf("Expected last byte to be zero, got %d instead", addr[3])
		}
		addr[3] = 10
		fromIP, ok := netip.AddrFromSlice(addr)
		if !ok {
			return fmt.Errorf("Must not reach")
		}
		addr[3] = 254
		toIP, ok := netip.AddrFromSlice(addr)
		if !ok {
			return fmt.Errorf("Must not reach")
		}
		{
			ingressPrivateIP := startAddr
			headscaleIP := ingressPrivateIP.Next()
			app, err := installer.FindEnvApp(st.appsRepo, "metallb-ipaddresspool")
			if err != nil {
				return err
			}
			{
				instanceId := fmt.Sprintf("%s-ingress-private", app.Name())
				appDir := fmt.Sprintf("/apps/%s", instanceId)
				namespace := fmt.Sprintf("%s%s-ingress-private", env.NamespacePrefix, app.Namespace())
				if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
					"name":       fmt.Sprintf("%s-ingress-private", env.Name),
					"from":       ingressPrivateIP.String(),
					"to":         ingressPrivateIP.String(),
					"autoAssign": false,
					"namespace":  "metallb-system",
				}); err != nil {
					return err
				}
			}
			{
				instanceId := fmt.Sprintf("%s-headscale", app.Name())
				appDir := fmt.Sprintf("/apps/%s", instanceId)
				namespace := fmt.Sprintf("%s%s-ingress-private", env.NamespacePrefix, app.Namespace())
				if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
					"name":       fmt.Sprintf("%s-headscale", env.Name),
					"from":       headscaleIP.String(),
					"to":         headscaleIP.String(),
					"autoAssign": false,
					"namespace":  "metallb-system",
				}); err != nil {
					return err
				}
			}
			{
				instanceId := app.Name()
				appDir := fmt.Sprintf("/apps/%s", instanceId)
				namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
				if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
					"name":       env.Name,
					"from":       fromIP.String(),
					"to":         toIP.String(),
					"autoAssign": false,
					"namespace":  "metallb-system",
				}); err != nil {
					return err
				}
			}
		}
		{
			keys, err := installer.NewSSHKeyPair("port-allocator")
			if err != nil {
				return err
			}
			user := fmt.Sprintf("%s-port-allocator", env.Name)
			if err := st.ssClient.AddUser(user, keys.AuthorizedKey()); err != nil {
				return err
			}
			if err := st.ssClient.AddReadWriteCollaborator("config", user); err != nil {
				return err
			}
			app, err := installer.FindEnvApp(st.appsRepo, "private-network")
			if err != nil {
				return err
			}
			instanceId := app.Name()
			appDir := fmt.Sprintf("/apps/%s", instanceId)
			namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
			if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
				"privateNetwork": map[string]any{
					"hostname": "private-network-proxy",
					"username": "private-network-proxy",
					"ipSubnet": fmt.Sprintf("%s/24", startIP.String()),
				},
				"sshPrivateKey": string(keys.RawPrivateKey()),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return &t
}

func SetupCertificateIssuers(env Env, st *state) Task {
	pub := newLeafTask(fmt.Sprintf("Public %s", env.Domain), func() error {
		app, err := installer.FindEnvApp(st.appsRepo, "certificate-issuer-public")
		if err != nil {
			return err
		}
		instanceId := app.Name()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{}); err != nil {
			return err
		}
		return nil
	})
	priv := newLeafTask(fmt.Sprintf("Private p.%s", env.Domain), func() error {
		app, err := installer.FindEnvApp(st.appsRepo, "certificate-issuer-private")
		if err != nil {
			return err
		}
		instanceId := app.Name()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"apiConfigMap": map[string]any{
				"name":      "api-config", // TODO(gio): take from global pcloud config
				"namespace": fmt.Sprintf("%s-dns-zone-manager", env.PCloudEnvName),
			},
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask("Configure TLS certificate issuers", false, &pub, &priv)
}

func SetupAuth(env Env, st *state) Task {
	t := newLeafTask("Setup", func() error {
		app, err := installer.FindEnvApp(st.appsRepo, "core-auth")
		if err != nil {
			return err
		}
		instanceId := app.Name()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"subdomain": "test", // TODO(giolekva): make core-auth chart actually use this
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask(
		"Authentication services",
		false,
		&t,
		waitForAddr(fmt.Sprintf("https://accounts-ui.%s", env.Domain)),
	)
}

func SetupGroupMemberships(env Env, st *state) Task {
	t := newLeafTask("Setup", func() error {
		app, err := installer.FindEnvApp(st.appsRepo, "memberships")
		if err != nil {
			return err
		}
		instanceId := app.Name()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"authGroups": strings.Join(initGroups, ","),
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask(
		"Group membership",
		false,
		&t,
		waitForAddr(fmt.Sprintf("https://memberships.p.%s", env.Domain)),
	)
}

func SetupHeadscale(env Env, startIP net.IP, st *state) Task {
	t := newLeafTask("Setup", func() error {
		app, err := installer.FindEnvApp(st.appsRepo, "headscale")
		if err != nil {
			return err
		}
		instanceId := app.Name()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"subdomain": "headscale",
			"ipSubnet":  fmt.Sprintf("%s/24", startIP),
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask(
		"Setup mesh VPN",
		false,
		&t,
		waitForAddr(fmt.Sprintf("https://headscale.%s/apple", env.Domain)),
	)
}

func SetupWelcome(env Env, st *state) Task {
	t := newLeafTask("Setup", func() error {
		keys, err := installer.NewSSHKeyPair("welcome")
		if err != nil {
			return err
		}
		user := fmt.Sprintf("%s-welcome", env.Name)
		if err := st.ssClient.AddUser(user, keys.AuthorizedKey()); err != nil {
			return err
		}
		if err := st.ssClient.AddReadWriteCollaborator("config", user); err != nil {
			return err
		}
		app, err := installer.FindEnvApp(st.appsRepo, "welcome")
		if err != nil {
			return err
		}
		instanceId := app.Name()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"repoAddr":      st.ssClient.GetRepoAddress("config"),
			"sshPrivateKey": string(keys.RawPrivateKey()),
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask(
		"Welcome service",
		false,
		&t,
		waitForAddr(fmt.Sprintf("https://welcome.%s", env.Domain)),
	)
}

func SetupAppStore(env Env, st *state) Task {
	t := newLeafTask("Application marketplace", func() error {
		user := fmt.Sprintf("%s-appmanager", env.Name)
		keys, err := installer.NewSSHKeyPair(user)
		if err != nil {
			return err
		}
		if err := st.ssClient.AddUser(user, keys.AuthorizedKey()); err != nil {
			return err
		}
		if err := st.ssClient.AddReadWriteCollaborator("config", user); err != nil {
			return err
		}
		app, err := installer.FindEnvApp(st.appsRepo, "app-manager") // TODO(giolekva): configure
		if err != nil {
			return err
		}
		instanceId := app.Name()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"repoAddr":      st.ssClient.GetRepoAddress("config"),
			"sshPrivateKey": string(keys.RawPrivateKey()),
			"authGroups":    strings.Join(initGroups, ","),
		}); err != nil {
			return err
		}
		return nil
	})
	return &t
}

type DNSSecKey struct {
	Basename string `json:"basename,omitempty"`
	Key      []byte `json:"key,omitempty"`
	Private  []byte `json:"private,omitempty"`
	DS       []byte `json:"ds,omitempty"`
}

func newDNSSecKey(zone string) (DNSSecKey, error) {
	key := &dns.DNSKEY{
		Hdr:       dns.RR_Header{Name: dns.Fqdn(zone), Class: dns.ClassINET, Ttl: 3600, Rrtype: dns.TypeDNSKEY},
		Algorithm: dns.ECDSAP256SHA256, Flags: 257, Protocol: 3,
	}
	priv, err := key.Generate(256)
	if err != nil {
		return DNSSecKey{}, err
	}
	return DNSSecKey{
		Basename: fmt.Sprintf("K%s+%03d+%05d", key.Header().Name, key.Algorithm, key.KeyTag()),
		Key:      []byte(key.String()),
		Private:  []byte(key.PrivateKeyString(priv)),
		DS:       []byte(key.ToDS(dns.SHA256).String()),
	}, nil
}
