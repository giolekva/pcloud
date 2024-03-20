package tasks

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/miekg/dns"

	"github.com/giolekva/pcloud/core/installer"
)

func SetupInfra(env Env, startIP net.IP, st *state) []Task {
	t := newLeafTask("Create client", func() error {
		repo, err := st.ssClient.GetRepo("config")
		if err != nil {
			return err
		}
		r := installer.NewRepoIO(repo, st.ssClient.Signer)
		appManager, err := installer.NewAppManager(r, st.nsCreator)
		if err != nil {
			return err
		}
		st.appManager = appManager
		st.appsRepo = installer.NewInMemoryAppRepository(installer.CreateAllApps())
		st.nsGen = installer.NewPrefixGenerator(env.Name + "-")
		st.emptySuffixGen = installer.NewEmptySuffixGenerator()
		return nil
	})
	return []Task{
		CommitEnvironmentConfiguration(env, st),
		&t,
		newConcurrentParentTask(
			"Core services",
			SetupNetwork(env, startIP, st),
			SetupCertificateIssuers(env, st),
			SetupAuth(env, st),
			SetupGroupMemberships(env, st),
			SetupHeadscale(env, startIP, st),
			SetupWelcome(env, st),
			SetupAppStore(env, st),
		),
	}
}

func CommitEnvironmentConfiguration(env Env, st *state) Task {
	t := newLeafTask("Configure environment infrastructure", func() error {
		repo, err := st.ssClient.GetRepo("config")
		if err != nil {
			return err
		}
		r := installer.NewRepoIO(repo, st.ssClient.Signer)
		{
			// TODO(giolekva): private domain can be configurable as well
			config := installer.Config{
				Values: installer.Values{
					PCloudEnvName:   env.PCloudEnvName,
					Id:              env.Name,
					ContactEmail:    env.ContactEmail,
					Domain:          env.Domain,
					PrivateDomain:   fmt.Sprintf("p.%s", env.Domain),
					PublicIP:        st.publicIPs[0].String(),
					NamespacePrefix: fmt.Sprintf("%s-", env.Name),
				},
			}
			if err := r.WriteYaml("config.yaml", config); err != nil {
				return err
			}
		}
		{
			out, err := r.Writer("pcloud-charts.yaml")
			if err != nil {
				return err
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
    branch: main
`, env.Name)
			if err != nil {
				return err
			}
			rootKust, err := r.ReadKustomization("kustomization.yaml")
			if err != nil {
				return err
			}
			rootKust.AddResources("pcloud-charts.yaml")
			if err := r.WriteKustomization("kustomization.yaml", *rootKust); err != nil {
				return err
			}
			r.CommitAndPush("configure charts repo")
		}
		return nil
	})
	return &t
}

func SetupNetwork(env Env, startIP net.IP, st *state) Task {
	t := newLeafTask("Setup network", func() error {
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
			app, err := st.appsRepo.Find("metallb-ipaddresspool")
			if err != nil {
				return err
			}
			if err := st.appManager.Install(app, st.nsGen, installer.NewSuffixGenerator("-ingress-private"), map[string]any{
				"name":       fmt.Sprintf("%s-ingress-private", env.Name),
				"from":       ingressPrivateIP.String(),
				"to":         ingressPrivateIP.String(),
				"autoAssign": false,
				"namespace":  "metallb-system",
			}); err != nil {
				return err
			}
			if err := st.appManager.Install(app, st.nsGen, installer.NewSuffixGenerator("-headscale"), map[string]any{
				"name":       fmt.Sprintf("%s-headscale", env.Name),
				"from":       headscaleIP.String(),
				"to":         headscaleIP.String(),
				"autoAssign": false,
				"namespace":  "metallb-system",
			}); err != nil {
				return err
			}
			if err := st.appManager.Install(app, st.nsGen, st.emptySuffixGen, map[string]any{
				"name":       env.Name,
				"from":       fromIP.String(),
				"to":         toIP.String(),
				"autoAssign": false,
				"namespace":  "metallb-system",
			}); err != nil {
				return err
			}
		}
		{
			app, err := st.appsRepo.Find("private-network")
			if err != nil {
				return err
			}
			if err := st.appManager.Install(app, st.nsGen, st.emptySuffixGen, map[string]any{
				"privateNetwork": map[string]any{
					"hostname": "private-network-proxy",
					"username": "private-network-proxy",
					"ipSubnet": fmt.Sprintf("%s/24", startIP.String()),
				},
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
		app, err := st.appsRepo.Find("certificate-issuer-public")
		if err != nil {
			return err
		}
		if err := st.appManager.Install(app, st.nsGen, st.emptySuffixGen, map[string]any{}); err != nil {
			return err
		}
		return nil
	})
	priv := newLeafTask(fmt.Sprintf("Private p.%s", env.Domain), func() error {
		app, err := st.appsRepo.Find("certificate-issuer-private")
		if err != nil {
			return err
		}
		if err := st.appManager.Install(app, st.nsGen, st.emptySuffixGen, map[string]any{
			"apiConfigMap": map[string]any{
				"name":      "api-config", // TODO(gio): take from global pcloud config
				"namespace": fmt.Sprintf("%s-dns-zone-manager", env.PCloudEnvName),
			},
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask("Configure TLS certificate issuers", &pub, &priv)
}

func SetupAuth(env Env, st *state) Task {
	t := newLeafTask("Setup", func() error {
		app, err := st.appsRepo.Find("core-auth")
		if err != nil {
			return err
		}
		if err := st.appManager.Install(app, st.nsGen, st.emptySuffixGen, map[string]any{
			"subdomain": "test", // TODO(giolekva): make core-auth chart actually use this
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask(
		"Authentication services",
		&t,
		waitForAddr(fmt.Sprintf("https://accounts-ui.%s", env.Domain)),
	)
}

func SetupGroupMemberships(env Env, st *state) Task {
	t := newLeafTask("Setup", func() error {
		app, err := st.appsRepo.Find("memberships")
		if err != nil {
			return err
		}
		if err := st.appManager.Install(app, st.nsGen, st.emptySuffixGen, map[string]any{}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask(
		"Group Membership",
		&t,
		waitForAddr(fmt.Sprintf("https://memberships.p.%s", env.Domain)),
	)
}

func SetupHeadscale(env Env, startIP net.IP, st *state) Task {
	t := newLeafTask("Setup", func() error {
		app, err := st.appsRepo.Find("headscale")
		if err != nil {
			return err
		}
		if err := st.appManager.Install(app, st.nsGen, st.emptySuffixGen, map[string]any{
			"subdomain": "headscale",
			"ipSubnet":  fmt.Sprintf("%s/24", startIP),
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask(
		"Headscale service",
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
		app, err := st.appsRepo.Find("welcome")
		if err != nil {
			return err
		}
		if err := st.appManager.Install(app, st.nsGen, st.emptySuffixGen, map[string]any{
			"repoAddr":      st.ssClient.GetRepoAddress("config"),
			"sshPrivateKey": string(keys.RawPrivateKey()),
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask(
		"Welcome service",
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
		app, err := st.appsRepo.Find("app-manager") // TODO(giolekva): configure
		if err != nil {
			return err
		}
		if err := st.appManager.Install(app, st.nsGen, st.emptySuffixGen, map[string]any{
			"repoAddr":      st.ssClient.GetRepoAddress("config"),
			"sshPrivateKey": string(keys.RawPrivateKey()),
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
