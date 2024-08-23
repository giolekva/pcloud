import (
	// "encoding/base64"
)

input: {
	cluster: #Cluster
	vpnUser: string
	vpnProxyHostname: string
	vpnAuthKey: string @role(VPNAuthKey) @usernameField(vpnUser)
	// TODO(gio): support port allocator
}

name: "Cluster Network"
namespace: "cluster-network"

out: {
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
		// portAllocator: {
		// 	repository: "giolekva"
		// 	name: "port-allocator"
		// 	tag: "latest"
		// 	pullPolicy: "Always"
		// }
	}

	charts: {
		"access-secrets": {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/access-secrets"
		}
		"ingress-nginx": {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/ingress-nginx"
		}
		"tailscale-proxy": {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/tailscale-proxy"
		}
		// portAllocator: {
		// 	kind: "GitRepository"
		// 	address: "https://code.v1.dodo.cloud/helm-charts"
		// 	branch: "main"
		// 	path: "charts/port-allocator"
		// }
	}

	helm: {
		_fullnameOverride: "\(global.id)-nginx-cluster-\(input.cluster.name)"
		"access-secrets": {
			chart: charts["access-secrets"]
			values: {
				serviceAccountName: _fullnameOverride
			}
		}
		"ingress-nginx": {
			chart: charts["ingress-nginx"]
			dependsOn: [{
				name: "access-secrets"
				namespace: release.namespace
			}]
			values: {
				fullnameOverride: _fullnameOverride
				controller: {
					service: enabled: false
					ingressClassByName: true
					ingressClassResource: {
						name: input.cluster.ingressClassName
						enabled: true
						default: false
						controllerValue: "k8s.io/\(input.cluster.name)"
					}
					config: {
						"proxy-body-size": "200M" // TODO(giolekva): configurable
						"force-ssl-redirect": "true"
						"server-snippet": """
						more_clear_headers "X-Frame-Options";
						"""
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
					extraContainers: [{
						name: "proxy"
						image: images["tailscale-proxy"].fullNameWithTag
						env: [{
							name: "TS_AUTHKEY"
							value: input.vpnAuthKey
					    }, {
							name: "TS_HOSTNAME"
							value: input.vpnProxyHostname
						}, {
							name: "TS_EXTRA_ARGS"
							value: "--login-server=https://headscale.\(global.domain)"
						}]
  				    }]
				}
			}
		}
		// "port-allocator": {
		// 	chart: charts.portAllocator
		// 	values: {
		// 		repoAddr: release.repoAddr
		// 		sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
		// 		ingressNginxPath: "\(release.appDir)/resources/ingress-nginx.yaml"
		// 		image: {
		// 			repository: images.portAllocator.fullName
		// 			tag: images.portAllocator.tag
		// 			pullPolicy: images.portAllocator.pullPolicy
		// 		}
		// 	}
		// }
	}
}
