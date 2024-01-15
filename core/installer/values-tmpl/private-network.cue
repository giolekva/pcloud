input: {
	privateNetwork: {
		hostname: string
		username: string
		ipSubnet: string // TODO(gio): use cidr type
	}
}

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
						"metallb.universe.tf/address-pool": _ingressPrivate
					}
				}
				ingressClassByName: true
				ingressClassResource: {
					name: _ingressPrivate
					enabled: true
					default: false
					controllerValue: "k8s.io/\(_ingressPrivate)"
				}
				extraArgs: {
					"default-ssl-certificate": "\(_ingressPrivate)/cert-wildcard.\(global.privateDomain)"
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
}
