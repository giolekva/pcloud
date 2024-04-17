package installer

import (
	"net"
	"net/netip"
)

type EnvServiceIPs struct {
	ConfigRepo    netip.Addr `json:"configRepo"`
	IngressPublic netip.Addr `json:"ingressPublic"`
	From          netip.Addr `json:"from"`
	To            netip.Addr `json:"to"`
}

type BootstrapConfig struct {
	InfraName                 string        `json:"name"`
	PublicIP                  []net.IP      `json:"publicIP"`
	NamespacePrefix           string        `json:"namespacePrefix"`
	StorageDir                string        `json:"storageDir"`
	VolumeDefaultReplicaCount int           `json:"volumeDefaultReplicaCount"`
	AdminPublicKey            []byte        `json:"adminPublicKey"`
	ServiceIPs                EnvServiceIPs `json:"serviceIPs"`
}

type EnvCIDR struct {
	Name string
	IP   net.IP
}

type EnvCIDRs []EnvCIDR
