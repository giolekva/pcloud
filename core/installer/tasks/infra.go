package tasks

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

var initGroups = []string{"admin"}

func CreateRepoClient(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Create repo client", func() error {
		r, err := st.ssClient.GetRepo("config")
		if err != nil {
			return err
		}
		appManager, err := installer.NewAppManager(r, st.nsCreator, st.jc, st.hf, "/apps")
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

func SetupInfra(env installer.EnvConfig, st *state) Task {
	tasks := []Task{
		SetupNetwork(env, st),
		SetupCertificateIssuers(env, st),
		SetupAuth(env, st),
		SetupGroupMemberships(env, st),
		SetupWelcome(env, st),
		SetupAppStore(env, st),
		SetupLauncher(env, st),
	}
	if env.PrivateDomain != "" {
		tasks = append(tasks, SetupHeadscale(env, st))
	}
	return newConcurrentParentTask(
		"Setup core services",
		true,
		tasks...,
	)
}

func CommitEnvironmentConfiguration(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("commit config", func() error {
		r, err := st.ssClient.GetRepo("config")
		if err != nil {
			return err
		}
		r.Do(func(r soft.RepoFS) (string, error) {
			if err := soft.WriteYaml(r, "config.yaml", env); err != nil {
				return "", err
			}
			rootKust, err := soft.ReadKustomization(r, "kustomization.yaml")
			if err != nil {
				return "", err
			}
			if err := soft.WriteYaml(r, "kustomization.yaml", rootKust); err != nil {
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
	Domain  string   `json:"domain"`
	Groups  []string `json:"groups"`
}

func ConfigureFirstAccount(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Configure first account settings", func() error {
		r, err := st.ssClient.GetRepo("config")
		if err != nil {
			return err
		}
		return r.Do(func(r soft.RepoFS) (string, error) {
			fa := firstAccount{false, env.Domain, initGroups}
			if err := soft.WriteYaml(r, "first-account.yaml", fa); err != nil {
				return "", err
			}
			return "first account membership configuration", nil
		})
	})
	return &t
}

func SetupNetwork(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Setup networks", func() error {
		{
			app, err := installer.FindEnvApp(st.appsRepo, "metallb-ipaddresspool")
			if err != nil {
				return err
			}
			{
				instanceId := fmt.Sprintf("%s-ingress-private", app.Slug())
				appDir := fmt.Sprintf("/apps/%s", instanceId)
				namespace := fmt.Sprintf("%s%s-ingress-private", env.NamespacePrefix, app.Namespace())
				if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
					"name":       fmt.Sprintf("%s-ingress-private", env.Id),
					"from":       env.Network.Ingress.String(),
					"to":         env.Network.Ingress.String(),
					"autoAssign": false,
					"namespace":  "metallb-system",
				}); err != nil {
					return err
				}
			}
			{
				instanceId := fmt.Sprintf("%s-headscale", app.Slug())
				appDir := fmt.Sprintf("/apps/%s", instanceId)
				namespace := fmt.Sprintf("%s%s-ingress-private", env.NamespacePrefix, app.Namespace())
				if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
					"name":       fmt.Sprintf("%s-headscale", env.Id),
					"from":       env.Network.Headscale.String(),
					"to":         env.Network.Headscale.String(),
					"autoAssign": false,
					"namespace":  "metallb-system",
				}); err != nil {
					return err
				}
			}
			{
				instanceId := app.Slug()
				appDir := fmt.Sprintf("/apps/%s", instanceId)
				namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
				if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
					"name":       env.Id,
					"from":       env.Network.ServicesFrom.String(),
					"to":         env.Network.ServicesTo.String(),
					"autoAssign": false,
					"namespace":  "metallb-system",
				}); err != nil {
					return err
				}
			}
		}
		if env.PrivateDomain != "" {
			keys, err := installer.NewSSHKeyPair("port-allocator")
			if err != nil {
				return err
			}
			user := fmt.Sprintf("%s-port-allocator", env.Id)
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
			instanceId := app.Slug()
			appDir := fmt.Sprintf("/apps/%s", instanceId)
			namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
			if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
				"privateNetwork": map[string]any{
					"hostname": "private-network-proxy",
					"username": "private-network-proxy",
					"ipSubnet": fmt.Sprintf("%s.0/24", strings.Join(strings.Split(env.Network.DNS.String(), ".")[:3], ".")),
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

func SetupCertificateIssuers(env installer.EnvConfig, st *state) Task {
	pub := newLeafTask(fmt.Sprintf("Public %s", env.Domain), func() error {
		app, err := installer.FindEnvApp(st.appsRepo, "certificate-issuer-public")
		if err != nil {
			return err
		}
		instanceId := app.Slug()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"network": "Public",
		}); err != nil {
			return err
		}
		return nil
	})
	tasks := []Task{&pub}
	if env.PrivateDomain != "" {
		priv := newLeafTask(fmt.Sprintf("Private p.%s", env.Domain), func() error {
			app, err := installer.FindEnvApp(st.appsRepo, "certificate-issuer-private")
			if err != nil {
				return err
			}
			instanceId := app.Slug()
			appDir := fmt.Sprintf("/apps/%s", instanceId)
			namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
			if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{}); err != nil {
				return err
			}
			return nil
		})
		tasks = append(tasks, &priv)
	}
	return newSequentialParentTask("Configure TLS certificate issuers", false, tasks...)
}

func SetupAuth(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Setup", func() error {
		app, err := installer.FindEnvApp(st.appsRepo, "core-auth")
		if err != nil {
			return err
		}
		instanceId := app.Slug()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"network":   "Public",
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
		waitForAddr(st.httpClient, fmt.Sprintf("https://accounts-ui.%s", env.Domain)),
	)
}

func SetupGroupMemberships(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Setup", func() error {
		app, err := installer.FindEnvApp(st.appsRepo, "memberships")
		if err != nil {
			return err
		}
		instanceId := app.Slug()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		network := "Public"
		if env.PrivateDomain != "" {
			network = "Private"
		}
		if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"network":    network,
			"authGroups": strings.Join(initGroups, ","),
		}); err != nil {
			return err
		}
		return nil
	})
	var addr string
	if env.PrivateDomain != "" {
		addr = fmt.Sprintf("https://memberships.%s", env.PrivateDomain)
	} else {
		addr = fmt.Sprintf("https://memberships.%s", env.Domain)
	}
	return newSequentialParentTask(
		"Group membership",
		false,
		&t,
		waitForAddr(st.httpClient, addr),
	)
}

func SetupLauncher(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Setup", func() error {
		user := fmt.Sprintf("%s-launcher", env.Id)
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
		app, err := installer.FindEnvApp(st.appsRepo, "launcher")
		if err != nil {
			return err
		}
		instanceId := app.Slug()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"network":       "Public",
			"repoAddr":      st.ssClient.GetRepoAddress("config"),
			"sshPrivateKey": string(keys.RawPrivateKey()),
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask(
		"Launcher",
		false,
		&t,
		waitForAddr(st.httpClient, fmt.Sprintf("https://launcher.%s", env.Domain)),
	)
}

func SetupHeadscale(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Setup", func() error {
		app, err := installer.FindEnvApp(st.appsRepo, "headscale")
		if err != nil {
			return err
		}
		instanceId := app.Slug()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"network":   "Public",
			"subdomain": "headscale",
			"ipSubnet":  fmt.Sprintf("%s/24", env.Network.DNS.String()),
		}); err != nil {
			return err
		}
		return nil
	})
	return newSequentialParentTask(
		"Setup mesh VPN",
		false,
		&t,
		waitForAddr(st.httpClient, fmt.Sprintf("https://headscale.%s/apple", env.Domain)),
	)
}

func SetupWelcome(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Setup", func() error {
		keys, err := installer.NewSSHKeyPair("welcome")
		if err != nil {
			return err
		}
		user := fmt.Sprintf("%s-welcome", env.Id)
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
		instanceId := app.Slug()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"network":       "Public",
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
		waitForAddr(st.httpClient, fmt.Sprintf("https://welcome.%s", env.Domain)),
	)
}

func SetupAppStore(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Setup", func() error {
		user := fmt.Sprintf("%s-appmanager", env.Id)
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
		instanceId := app.Slug()
		appDir := fmt.Sprintf("/apps/%s", instanceId)
		namespace := fmt.Sprintf("%s%s", env.NamespacePrefix, app.Namespace())
		network := "Public"
		if env.PrivateDomain != "" {
			network = "Private"
		}
		if _, err := st.appManager.Install(app, instanceId, appDir, namespace, map[string]any{
			"network":       network,
			"repoAddr":      st.ssClient.GetRepoAddress("config"),
			"sshPrivateKey": string(keys.RawPrivateKey()),
			"authGroups":    strings.Join(initGroups, ","),
		}); err != nil {
			return err
		}
		return nil
	})
	var addr string
	if env.PrivateDomain != "" {
		addr = fmt.Sprintf("https://apps.%s", env.PrivateDomain)
	} else {
		addr = fmt.Sprintf("https://apps.%s", env.Domain)
	}
	return newSequentialParentTask(
		"Application marketplace",
		false,
		&t,
		waitForAddr(st.httpClient, addr),
	)
}

// TODO(gio-dns): remove
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
