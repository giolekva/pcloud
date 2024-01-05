package installer

import (
	"net/netip"
)

type EnvServiceIPs struct {
	ConfigRepo    netip.Addr `json:"configRepo"`
	IngressPublic netip.Addr `json:"ingressPublic"`
	From          netip.Addr `json:"from"`
	To            netip.Addr `json:"to"`
}

type EnvConfig struct {
	Name                      string        `json:"name"`
	PublicIP                  string        `json:"publicIP"`
	NamespacePrefix           string        `json:"namespacePrefix"`
	StorageDir                string        `json:"storageDir"`
	VolumeDefaultReplicaCount int           `json:"volumeDefaultReplicaCount"`
	AdminPublicKey            []byte        `json:"adminPublicKey"`
	ServiceIPs                EnvServiceIPs `json:"serviceIPs"`
}

type Config struct {
	Values Values `json:"input"` // TODO(gio): rename
}

type Values struct {
	PCloudEnvName   string `json:"pcloudEnvName,omitempty"`
	Id              string `json:"id,omitempty"`
	ContactEmail    string `json:"contactEmail,omitempty"`
	Domain          string `json:"domain,omitempty"`
	PrivateDomain   string `json:"privateDomain,omitempty"`
	PublicIP        string `json:"publicIP,omitempty"`
	NamespacePrefix string `json:"namespacePrefix,omitempty"`
	// GandiAPIToken   string `json:"gandiAPIToken,omitempty"`
	// LighthouseAuthUIIP       string `json:"lighthouseAuthUIIP,omitempty"`
	// LighthouseMainIP         string `json:"lighthouseMainIP,omitempty"`
	// LighthouseMainPort       string `json:"lighthouseMainPort,omitempty"`
	// MXHostname               string `json:"mxHostname,omitempty"`
	// MailGatewayAddress       string `json:"mailGatewayAddress,omitempty"`
	// MatrixOAuth2ClientSecret string `json:"matrixOAuth2ClientSecret,omitempty"`
	// MatrixStorageSize        string `json:"matrixStorageSize,omitempty"`
	// PiholeOAuth2ClientSecret string `json:"piholeOAuth2ClientSecret,omitempty"`
	// PiholeOAuth2CookieSecret string `json:"piholeOAuth2CookieSecret,omitempty"`
}
