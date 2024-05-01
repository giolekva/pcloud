import (
	"encoding/base64"
)

input: {
	privateNetwork: {
		hostname: string
		username: string
		ipSubnet: string // TODO(gio): use cidr type
	}
	sshPrivateKey: string
}

name: "private-network"
namespace: "ingress-private"

images: {
	"ingress-nginx": {
		registry: "registry.k8s.io"
		repository: "ingress-nginx"
		name: "controller"
		tag: "v1.8.0"
		pullPolicy: "IfNotPresent"
	}
	"tailscale-proxy": {
		repository: "tailscale"
		name: "tailscale"
		tag: "v1.42.0"
		pullPolicy: "IfNotPresent"
	}
	portAllocator: {
		repository: "giolekva"
		name: "port-allocator"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	"ingress-nginx": {
		chart: "charts/ingress-nginx"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
	"tailscale-proxy": {
		chart: "charts/tailscale-proxy"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
	portAllocator: {
		chart: "charts/port-allocator"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	"ingress-nginx": {
		chart: charts["ingress-nginx"]
		values: {
			fullnameOverride: "\(global.id)-nginx-private"
			controller: {
				service: {
					enabled: true
					type: "LoadBalancer"
					annotations: {
						"metallb.universe.tf/address-pool": ingressPrivate
					}
				}
				ingressClassByName: true
				ingressClassResource: {
					name: ingressPrivate
					enabled: true
					default: false
					controllerValue: "k8s.io/\(ingressPrivate)"
				}
				config: {
					"proxy-body-size": "200M" // TODO(giolekva): configurable
					"force-ssl-redirect": "true"
					"server-snippet": """
					more_clear_headers "X-Frame-Options";
					"""
				}
				extraArgs: {
					"default-ssl-certificate": "\(ingressPrivate)/cert-wildcard.\(global.privateDomain)"
				}
				admissionWebhooks: {
					enabled: false
				}
				image: {
					registry: images["ingress-nginx"].registry
					image: images["ingress-nginx"].imageName
					tag: images["ingress-nginx"].tag
					pullPolicy: images["ingress-nginx"].pullPolicy
				}
			}
		}
	}
	"tailscale-proxy": {
		chart: charts["tailscale-proxy"]
		values: {
			hostname: input.privateNetwork.hostname
			apiServer: "http://headscale-api.\(global.namespacePrefix)app-headscale.svc.cluster.local"
			loginServer: "https://headscale.\(global.domain)" // TODO(gio): take headscale subdomain from configuration
			ipSubnet: input.privateNetwork.ipSubnet
			username: input.privateNetwork.username // TODO(gio): maybe install headscale-user chart separately?
			preAuthKeySecret: "headscale-preauth-key"
			image: {
				repository: images["tailscale-proxy"].fullName
				tag: images["tailscale-proxy"].tag
				pullPolicy: images["tailscale-proxy"].pullPolicy
			}
		}
	}
	"port-allocator": {
		chart: charts.portAllocator
		values: {
			repoAddr: release.repoAddr
			sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
			ingressNginxPath: "\(release.appDir)/resources/ingress-nginx.yaml"
			image: {
				repository: images.portAllocator.fullName
				tag: images.portAllocator.tag
				pullPolicy: images.portAllocator.pullPolicy
			}
		}
	}
}
