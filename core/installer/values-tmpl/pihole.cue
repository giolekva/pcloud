input: {
	network: #Network
	subdomain: string
}

_domain: "\(input.subdomain).\(input.network.domain)"

readme: "Installs pihole at https://\(_domain)"

images: {
	pihole: {
		repository: "pihole"
		name: "pihole"
		tag: "v5.8.1"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	pihole: {
		chart: "charts/pihole"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	pihole: {
		chart: charts.pihole
		values: {
			domain: _domain
			pihole: {
				fullnameOverride: "pihole"
				persistentVolumeClaim: { // TODO(gio): create volume separately as a dependency
					enabled: true
					size: "5Gi"
				}
				admin: {
					enabled: false
				}
				ingress: {
					enabled: false
				}
				serviceDhcp: {
					enabled: false
				}
				serviceDns: {
					type: "ClusterIP"
				}
				serviceWeb: {
					type: "ClusterIP"
					http: {
						enabled: true
					}
					https: {
						enabled: false
					}
				}
				virtualHost: _domain
				resources: {
					requests: {
						cpu: "250m"
						memory: "100M"
					}
					limits: {
						cpu: "500m"
						memory: "250M"
					}
				}
				image: {
					repository: images.pihole.fullName
					tag: images.pihole.tag
					pullPolicy: images.pihole.pullPolicy
				}
			}
			oauth2: {
				secretName: "oauth2-secret"
				configName: "oauth2-proxy"
				hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc"
			}
			hydraPublic: "https://hydra.\(global.domain)"
			profileUrl: "https://accounts-ui.\(global.domain)"
			ingressClassName: input.network.ingressClass
			certificateIssuer: input.network.certificateIssuer
		}
	}
}
