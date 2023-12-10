package tasks

import (
	"fmt"
	"net/netip"

	"github.com/miekg/dns"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

type setupInfraAppsTask struct {
	basicTask
	env Env
	st  *state
}

func NewSetupInfraAppsTask(env Env, st *state) Task {
	return &setupInfraAppsTask{
		basicTask: basicTask{
			title: "Configure environment infrastructure",
		},
		env: env,
		st:  st,
	}
}

func (t *setupInfraAppsTask) Start() {
	repo, err := t.st.ssClient.GetRepo("config")
	if err != nil {
		t.callDoneListeners(err)
		return
	}
	if err := t.initNewEnv(t.st.ssClient, installer.NewRepoIO(repo, t.st.ssClient.Signer), t.st.nsCreator, t.env.PCloudEnvName, t.st.publicIPs[0].String()); err != nil {
		t.callDoneListeners(err)
		return
	}
	t.callDoneListeners(nil)
}

func (t *setupInfraAppsTask) initNewEnv(
	ss *soft.Client,
	r installer.RepoIO,
	nsCreator installer.NamespaceCreator,
	pcloudEnvName string,
	pcloudPublicIP string,
) error {
	appManager, err := installer.NewAppManager(r, nsCreator)
	if err != nil {
		return err
	}
	appsRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	// TODO(giolekva): private domain can be configurable as well
	config := installer.Config{
		Values: installer.Values{
			PCloudEnvName:   pcloudEnvName,
			Id:              t.env.Name,
			ContactEmail:    t.env.ContactEmail,
			Domain:          t.env.Domain,
			PrivateDomain:   fmt.Sprintf("p.%s", t.env.Domain),
			PublicIP:        pcloudPublicIP,
			NamespacePrefix: fmt.Sprintf("%s-", t.env.Name),
		},
	}
	if err := r.WriteYaml("config.yaml", config); err != nil {
		return err
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
`, t.env.Name)
		if err != nil {
			return err
		}
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
	nsGen := installer.NewPrefixGenerator(t.env.Name + "-")
	emptySuffixGen := installer.NewEmptySuffixGenerator()
	ingressPrivateIP, err := netip.ParseAddr("10.1.0.1")
	if err != nil {
		return err
	}
	{
		headscaleIP := ingressPrivateIP.Next()
		app, err := appsRepo.Find("metallb-ipaddresspool")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, installer.NewSuffixGenerator("-ingress-private"), map[string]any{
			"Name":       fmt.Sprintf("%s-ingress-private", t.env.Name),
			"From":       ingressPrivateIP.String(),
			"To":         ingressPrivateIP.String(),
			"AutoAssign": false,
			"Namespace":  "metallb-system",
		}); err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, installer.NewSuffixGenerator("-headscale"), map[string]any{
			"Name":       fmt.Sprintf("%s-headscale", t.env.Name),
			"From":       headscaleIP.String(),
			"To":         headscaleIP.String(),
			"AutoAssign": false,
			"Namespace":  "metallb-system",
		}); err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, emptySuffixGen, map[string]any{
			"Name":       t.env.Name,
			"From":       "10.1.0.100", // TODO(gio): auto-generate
			"To":         "10.1.0.254",
			"AutoAssign": false,
			"Namespace":  "metallb-system",
		}); err != nil {
			return err
		}
	}
	{
		app, err := appsRepo.Find("private-network")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, emptySuffixGen, map[string]any{
			"PrivateNetwork": map[string]any{
				"Hostname": "private-network-proxy",
				"Username": "private-network-proxy",
				"IPSubnet": "10.1.0.0/24",
			},
		}); err != nil {
			return err
		}
	}
	{
		app, err := appsRepo.Find("certificate-issuer-public")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, emptySuffixGen, map[string]any{}); err != nil {
			return err
		}
	}
	{
		app, err := appsRepo.Find("certificate-issuer-private")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, emptySuffixGen, map[string]any{
			"APIConfigMap": map[string]any{
				"Name":      "api-config", // TODO(gio): take from global pcloud config
				"Namespace": fmt.Sprintf("%s-dns-zone-manager", pcloudEnvName),
			},
		}); err != nil {
			return err
		}
	}
	{
		app, err := appsRepo.Find("core-auth")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, emptySuffixGen, map[string]any{
			"Subdomain": "test", // TODO(giolekva): make core-auth chart actually use this
		}); err != nil {
			return err
		}
	}
	{
		app, err := appsRepo.Find("headscale")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, emptySuffixGen, map[string]any{
			"Subdomain": "headscale",
		}); err != nil {
			return err
		}
	}
	{
		keys, err := installer.NewSSHKeyPair("welcome")
		if err != nil {
			return err
		}
		user := fmt.Sprintf("%s-welcome", t.env.Name)
		if err := ss.AddUser(user, keys.AuthorizedKey()); err != nil {
			return err
		}
		if err := ss.AddReadWriteCollaborator("config", user); err != nil {
			return err
		}
		app, err := appsRepo.Find("welcome")
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, emptySuffixGen, map[string]any{
			"RepoAddr":      ss.GetRepoAddress("config"),
			"SSHPrivateKey": string(keys.RawPrivateKey()),
		}); err != nil {
			return err
		}
	}
	{
		user := fmt.Sprintf("%s-appmanager", t.env.Name)
		keys, err := installer.NewSSHKeyPair(user)
		if err != nil {
			return err
		}
		if err := ss.AddUser(user, keys.AuthorizedKey()); err != nil {
			return err
		}
		if err := ss.AddReadWriteCollaborator("config", user); err != nil {
			return err
		}
		app, err := appsRepo.Find("app-manager") // TODO(giolekva): configure
		if err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, emptySuffixGen, map[string]any{
			"RepoAddr":      ss.GetRepoAddress("config"),
			"SSHPrivateKey": string(keys.RawPrivateKey()),
		}); err != nil {
			return err
		}
	}
	return nil
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
