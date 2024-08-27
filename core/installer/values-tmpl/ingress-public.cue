import (
	"encoding/base64"
)

input: {
	sshPrivateKey: string
}

name: "ingress-public"
namespace: "ingress-public"

out: {
	images: {
		ingressNginx: {
			registry: "registry.k8s.io"
			repository: "ingress-nginx"
			name: "controller"
			tag: "v1.8.0"
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
		ingressNginx: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/ingress-nginx"
		}
		portAllocator: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/port-allocator"
		}
	}

	helm: {
		"ingress-public": {
			chart: charts.ingressNginx
			values: {
				fullnameOverride: "\(global.pcloudEnvName)-ingress-public"
				controller: {
					kind: "Deployment"
					replicaCount: 1 // TODO(gio): configurable
					topologySpreadConstraints: [{
						labelSelector: {
							matchLabels: {
								"app.kubernetes.io/instance": "ingress-public"
							}
						}
						maxSkew: 1
						topologyKey: "kubernetes.io/hostname"
						whenUnsatisfiable: "DoNotSchedule"
					}]
					hostNetwork: false
					hostPort: enabled: false
					updateStrategy: {
						type: "RollingUpdate"
						rollingUpdate: {
							maxSurge: "100%"
							maxUnavailable: "30%"
						}
					}
					service: {
						enabled: true
						type: "NodePort"
						nodePorts: {
							http: 80
							https: 443
							tcp: {
								"53": 53
							}
							udp: {
								"53": 53
							}
						}
					}
					ingressClassByName: true
					ingressClassResource: {
						name: networks.public.ingressClass
						enabled: true
						default: false
						controllerValue: "k8s.io/\(networks.public.ingressClass)"
					}
					config: {
						"proxy-body-size": "200M" // TODO(giolekva): configurable
						"server-snippet": """
						more_clear_headers "X-Frame-Options";
						"""
					}
					image: {
						registry: images.ingressNginx.registry
						image: images.ingressNginx.imageName
						tag: images.ingressNginx.tag
						pullPolicy: images.ingressNginx.pullPolicy
					}
				}
				tcp: {
					"53": "\(global.pcloudEnvName)-dns-gateway/coredns:53"
				}
				udp: {
					"53": "\(global.pcloudEnvName)-dns-gateway/coredns:53"
				}
			}
		}
		"port-allocator": {
			chart: charts.portAllocator
			values: {
				repoAddr: release.repoAddr
				sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
				ingressNginxPath: "\(release.appDir)/ingress-public.yaml"
				image: {
					repository: images.portAllocator.fullName
					tag: images.portAllocator.tag
					pullPolicy: images.portAllocator.pullPolicy
				}
			}
		}
	}
}
