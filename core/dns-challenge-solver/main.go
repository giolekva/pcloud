package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
)

var (
	groupName    = os.Getenv("API_GROUP_NAME")
	resolverName = os.Getenv("RESOLVER_NAME")
)

func main() {
	if groupName == "" {
		panic("API_GROUP_NAME must be specified")
	}
	if resolverName == "" {
		panic("RESOLVER_NAME must be specified")
	}
	cmd.RunWebhookServer(groupName,
		&pcloudDNSProviderSolver{},
	)
}

type ZoneManager interface {
	CreateTextRecord(domain, entry, txt string) error
	DeleteTextRecord(domain, entry, txt string) error
}

type zoneControllerManager struct {
	CreateAddr string
	DeleteAddr string
}

type createTextRecordReq struct {
	Domain string `json:"domain,omitempty"`
	Entry  string `json:"entry,omitempty"`
	Text   string `json:"text,omitempty"`
}

const contentTypeApplicationJSON = "application/json"

func (m *zoneControllerManager) CreateTextRecord(domain, entry, txt string) error {
	req := createTextRecordReq{domain, entry, txt}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return err
	}
	if resp, err := http.Post(m.CreateAddr, contentTypeApplicationJSON, &buf); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		var b strings.Builder
		io.Copy(&b, resp.Body)
		return fmt.Errorf("Create text record failed: %d %s", resp.StatusCode, b.String())
	}
	return nil
}

func (m *zoneControllerManager) DeleteTextRecord(domain, entry, txt string) error {
	req := createTextRecordReq{domain, entry, txt}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return err
	}
	if resp, err := http.Post(m.DeleteAddr, contentTypeApplicationJSON, &buf); err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		var b strings.Builder
		io.Copy(&b, resp.Body)
		return fmt.Errorf("Delete text record failed: %d %s", resp.StatusCode, b.String())
	}
	return nil
}

type pcloudDNSProviderSolver struct {
	// If a Kubernetes 'clientset' is needed, you must:
	// 1. uncomment the additional `client` field in this structure below
	// 2. uncomment the "k8s.io/client-go/kubernetes" import at the top of the file
	// 3. uncomment the relevant code in the Initialize method below
	// 4. ensure your webhook's service account has the required RBAC role
	//    assigned to it for interacting with the Kubernetes APIs you need.
	client *kubernetes.Clientset
}

// customDNSProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
// This typically includes references to Secret resources containing DNS
// provider credentials, in cases where a 'multi-tenant' DNS solver is being
// created.
// If you do *not* require per-issuer or per-certificate configuration to be
// provided to your webhook, you can skip decoding altogether in favour of
// using CLI flags or similar to provide configuration.
// You should not include sensitive information here. If credentials need to
// be used by your provider here, you should reference a Kubernetes Secret
// resource and fetch these credentials using a Kubernetes clientset.
type pcloudDNSProviderConfig struct {
	APIConfigMapName      string `json:"apiConfigMapName,omitempty"`
	APIConfigMapNamespace string `json:"apiConfigMapNamespace,omitempty"`
}

type apiConfig struct {
	CreateAddress string `json:"createTXTAddr,omitempty"`
	DeleteAddress string `json:"deleteTXTAddr,omitempty"`
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (c *pcloudDNSProviderSolver) Name() string {
	return resolverName
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (c *pcloudDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	fmt.Printf("Received challenge %+v\n", ch)
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}
	apiCfg, err := loadAPIConfig(c.client, cfg)
	if err != nil {
		return err
	}
	zm := &zoneControllerManager{apiCfg.CreateAddress, apiCfg.DeleteAddress}
	domain, entry := getDomainAndEntry(ch)
	return zm.CreateTextRecord(domain, entry, ch.Key)
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (c *pcloudDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}
	apiCfg, err := loadAPIConfig(c.client, cfg)
	if err != nil {
		return err
	}
	zm := &zoneControllerManager{apiCfg.CreateAddress, apiCfg.DeleteAddress}
	domain, entry := getDomainAndEntry(ch)
	return zm.DeleteTextRecord(domain, entry, ch.Key)
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (c *pcloudDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	fmt.Println("Initialization start")
	client, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}
	c.client = client
	fmt.Println("Initialization done")
	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (pcloudDNSProviderConfig, error) {
	cfg := pcloudDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

func loadAPIConfig(client *kubernetes.Clientset, cfg pcloudDNSProviderConfig) (apiConfig, error) {
	config, err := client.CoreV1().ConfigMaps(cfg.APIConfigMapNamespace).Get(context.Background(), cfg.APIConfigMapName, metav1.GetOptions{})
	if err != nil {
		return apiConfig{}, fmt.Errorf("unable to get api config map `%s` `%s`; %v", cfg.APIConfigMapName, cfg.APIConfigMapNamespace, err)
	}
	create, ok := config.Data["createTXTAddr"]
	if !ok {
		return apiConfig{}, fmt.Errorf("create address missing")
	}
	delete, ok := config.Data["deleteTXTAddr"]
	if !ok {
		return apiConfig{}, fmt.Errorf("delete address missing")
	}
	return apiConfig{create, delete}, nil
}

func getDomainAndEntry(ch *v1alpha1.ChallengeRequest) (string, string) {
	// Both ch.ResolvedZone and ch.ResolvedFQDN end with a dot: '.'
	entry := strings.TrimSuffix(ch.ResolvedFQDN, ch.ResolvedZone)
	entry = strings.TrimSuffix(entry, ".")
	domain := strings.TrimSuffix(ch.ResolvedZone, ".")
	return domain, entry
}
