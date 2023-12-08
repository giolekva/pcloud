package welcome

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	htemplate "html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/netip"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/charmbracelet/keygen"
	"github.com/gorilla/mux"
	"github.com/miekg/dns"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

//go:embed env-tmpl
var filesTmpls embed.FS

//go:embed create-env.html
var createEnvFormHtml string

//go:embed env-created.html
var envCreatedHtml string

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusAccepted Status = "ACCEPTED"
)

// TODO(giolekva): add CreatedAt and ValidUntil
type invitation struct {
	Token  string `json:"token"`
	Status Status `json:"status"`
}

type EnvServer struct {
	port          int
	ss            *soft.Client
	repo          installer.RepoIO
	nsCreator     installer.NamespaceCreator
	nameGenerator installer.NameGenerator
}

func NewEnvServer(port int, ss *soft.Client, repo installer.RepoIO, nsCreator installer.NamespaceCreator, nameGenerator installer.NameGenerator) *EnvServer {
	return &EnvServer{
		port,
		ss,
		repo,
		nsCreator,
		nameGenerator,
	}
}

func (s *EnvServer) Start() {
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticAssets)))
	r.Path("/").Methods("GET").HandlerFunc(s.createEnvForm)
	r.Path("/").Methods("POST").HandlerFunc(s.createEnv)
	r.Path("/create-invitation").Methods("GET").HandlerFunc(s.createInvitation)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil))
}

func (s *EnvServer) createEnvForm(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte(createEnvFormHtml)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *EnvServer) createInvitation(w http.ResponseWriter, r *http.Request) {
	invitations, err := s.readInvitations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	token, err := installer.NewFixedLengthRandomNameGenerator(100).Generate() // TODO(giolekva): use cryptographic tokens
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return

	}
	invitations = append(invitations, invitation{token, StatusActive})
	if err := s.writeInvitations(invitations); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write([]byte("OK")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type createEnvReq struct {
	Name           string
	ContactEmail   string `json:"contactEmail"`
	Domain         string `json:"domain"`
	AdminPublicKey string `json:"adminPublicKey"`
	SecretToken    string `json:"secretToken"`
}

func (s *EnvServer) readInvitations() ([]invitation, error) {
	r, err := s.repo.Reader("invitations")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return make([]invitation, 0), nil
		}
		return nil, err
	}
	defer r.Close()
	dec := json.NewDecoder(r)
	invitations := make([]invitation, 0)
	for {
		var i invitation
		if err := dec.Decode(&i); err == io.EOF {
			break
		}
		invitations = append(invitations, i)
	}
	return invitations, nil
}

func (s *EnvServer) writeInvitations(invitations []invitation) error {
	w, err := s.repo.Writer("invitations")
	if err != nil {
		return err
	}
	defer w.Close()
	enc := json.NewEncoder(w)
	for _, i := range invitations {
		if err := enc.Encode(i); err != nil {
			return err
		}
	}
	return s.repo.CommitAndPush("Generated new invitation")
}

func extractRequest(r *http.Request) (createEnvReq, error) {
	var req createEnvReq
	if err := func() error {
		var err error
		if err = r.ParseForm(); err != nil {
			return err
		}
		if req.SecretToken, err = getFormValue(r.PostForm, "secret-token"); err != nil {
			return err
		}
		if req.Domain, err = getFormValue(r.PostForm, "domain"); err != nil {
			return err
		}
		if req.ContactEmail, err = getFormValue(r.PostForm, "contact-email"); err != nil {
			return err
		}
		if req.AdminPublicKey, err = getFormValue(r.PostForm, "admin-public-key"); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return createEnvReq{}, err
		}
	}
	return req, nil
}

func (s *EnvServer) acceptInvitation(token string) error {
	invitations, err := s.readInvitations()
	if err != nil {
		return err
	}
	found := false
	for i := range invitations {
		if invitations[i].Token == token && invitations[i].Status == StatusActive {
			invitations[i].Status = StatusAccepted
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Invitation not found")
	}
	return s.writeInvitations(invitations)
}

func (s *EnvServer) createEnv(w http.ResponseWriter, r *http.Request) {
	req, err := extractRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var env installer.EnvConfig
	cr, err := s.repo.Reader("config.yaml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cr.Close()
	if err := installer.ReadYaml(cr, &env); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.acceptInvitation(req.SecretToken); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if name, err := s.nameGenerator.Generate(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		req.Name = name
	}
	appsRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	ssApp, err := appsRepo.Find("soft-serve")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ssAdminKeys, err := installer.NewSSHKeyPair(fmt.Sprintf("%s-config-repo-admin-keys", req.Name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ssKeys, err := installer.NewSSHKeyPair(fmt.Sprintf("%s-config-repo-keys", req.Name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ssValues := map[string]any{
		"ChartRepositoryNamespace": env.Name,
		"ServiceType":              "ClusterIP",
		"PrivateKey":               string(ssKeys.RawPrivateKey()),
		"PublicKey":                string(ssKeys.RawAuthorizedKey()),
		"AdminKey":                 string(ssAdminKeys.RawAuthorizedKey()),
		"Ingress": map[string]any{
			"Enabled": false,
		},
	}
	derived := installer.Derived{
		Global: installer.Values{
			Id:            req.Name,
			PCloudEnvName: env.Name,
		},
		Release: installer.Release{
			Namespace: req.Name,
		},
		Values: ssValues,
	}
	if err := s.nsCreator.Create(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.repo.InstallApp(*ssApp, filepath.Join("/environments", req.Name, "config-repo"), ssValues, derived); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	k := installer.NewKustomization()
	k.AddResources("config-repo")
	if err := s.repo.WriteKustomization(filepath.Join("/environments", req.Name, "kustomization.yaml"), k); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ssClient, err := soft.WaitForClient(
		fmt.Sprintf("soft-serve.%s.svc.cluster.local:%d", req.Name, 22),
		ssAdminKeys.RawPrivateKey(),
		log.Default())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := ssClient.AddPublicKey("admin", req.AdminPublicKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := ssClient.RemovePublicKey("admin", string(ssAdminKeys.RawAuthorizedKey())); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}()
	fluxUserName := fmt.Sprintf("flux-%s", req.Name)
	keys, err := installer.NewSSHKeyPair(fluxUserName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	{
		if err := ssClient.AddRepository("config"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		repo, err := ssClient.GetRepo("config")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		repoIO := installer.NewRepoIO(repo, ssClient.Signer)
		if err := repoIO.WriteCommitAndPush("README.md", fmt.Sprintf("# %s PCloud environment", req.Name), "readme"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := ssClient.AddUser(fluxUserName, keys.AuthorizedKey()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := ssClient.AddReadOnlyCollaborator("config", fluxUserName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	{
		repo, err := ssClient.GetRepo("config")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := initNewEnv(ssClient, installer.NewRepoIO(repo, ssClient.Signer), s.nsCreator, req, env.Name, env.PublicIP); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	{
		ssPublicKeys, err := ssClient.GetPublicKeys()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := addNewEnv(
			s.repo,
			req,
			strings.Split(ssClient.Addr, ":")[0],
			keys,
			ssPublicKeys,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	tmpl, err := htemplate.New("response").Parse(envCreatedHtml)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, map[string]any{
		"Domain": req.Domain,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

func initNewEnv(
	ss *soft.Client,
	r installer.RepoIO,
	nsCreator installer.NamespaceCreator,
	req createEnvReq,
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
			Id:              req.Name,
			ContactEmail:    req.ContactEmail,
			Domain:          req.Domain,
			PrivateDomain:   fmt.Sprintf("p.%s", req.Domain),
			PublicIP:        pcloudPublicIP,
			NamespacePrefix: fmt.Sprintf("%s-", req.Name),
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
`, req.Name)
		if err != nil {
			return err
		}
	}
	{
		key, err := newDNSSecKey(req.Domain)
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
  privateIP: 10.1.0.1
  publicIPs:
  - 135.181.48.180
  - 65.108.39.172
  nameservers:
  - 135.181.48.180
  - 65.108.39.172
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
			"namespace": req.Name,
			"zone":      req.Domain,
			"dnssec":    key,
		}); err != nil {
			return err
		}
	}
	rootKust := installer.NewKustomization()
	rootKust.AddResources("pcloud-charts.yaml", "dns-zone.yaml", "apps")
	if err := r.WriteKustomization("kustomization.yaml", rootKust); err != nil {
		return err
	}
	appsKust := installer.NewKustomization()
	if err := r.WriteKustomization("apps/kustomization.yaml", appsKust); err != nil {
		return err
	}
	r.CommitAndPush("initialize config")
	nsGen := installer.NewPrefixGenerator(req.Name + "-")
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
			"Name":       fmt.Sprintf("%s-ingress-private", req.Name),
			"From":       ingressPrivateIP.String(),
			"To":         ingressPrivateIP.String(),
			"AutoAssign": false,
			"Namespace":  "metallb-system",
		}); err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, installer.NewSuffixGenerator("-headscale"), map[string]any{
			"Name":       fmt.Sprintf("%s-headscale", req.Name),
			"From":       headscaleIP.String(),
			"To":         headscaleIP.String(),
			"AutoAssign": false,
			"Namespace":  "metallb-system",
		}); err != nil {
			return err
		}
		if err := appManager.Install(*app, nsGen, emptySuffixGen, map[string]any{
			"Name":       req.Name,
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
		user := fmt.Sprintf("%s-welcome", req.Name)
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
		user := fmt.Sprintf("%s-appmanager", req.Name)
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

func addNewEnv(
	repoIO installer.RepoIO,
	req createEnvReq,
	repoHost string,
	keys *keygen.KeyPair,
	configRepoPublicKeys []string,
) error {
	kust, err := repoIO.ReadKustomization("environments/kustomization.yaml")
	if err != nil {
		return err
	}
	kust.AddResources(req.Name)
	tmpls, err := template.ParseFS(filesTmpls, "env-tmpl/*.yaml")
	if err != nil {
		return err
	}
	var knownHosts bytes.Buffer
	for _, key := range configRepoPublicKeys {
		fmt.Fprintf(&knownHosts, "%s %s\n", repoHost, key)
	}
	for _, tmpl := range tmpls.Templates() {
		dstPath := path.Join("environments", req.Name, tmpl.Name())
		dst, err := repoIO.Writer(dstPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		if err := tmpl.Execute(dst, map[string]string{
			"Name":       req.Name,
			"PrivateKey": base64.StdEncoding.EncodeToString(keys.RawPrivateKey()),
			"PublicKey":  base64.StdEncoding.EncodeToString(keys.RawAuthorizedKey()),
			"RepoHost":   repoHost,
			"RepoName":   "config",
			"KnownHosts": base64.StdEncoding.EncodeToString(knownHosts.Bytes()),
		}); err != nil {
			return err
		}
	}
	if err := repoIO.WriteKustomization("environments/kustomization.yaml", *kust); err != nil {
		return err
	}
	if err := repoIO.CommitAndPush(fmt.Sprintf("%s: initialize environment", req.Name)); err != nil {
		return err
	}
	return nil
}
