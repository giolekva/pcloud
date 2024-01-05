#Input: {
	network: #Network
	subdomain: string
}

input: #Input

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

// TODO(gio): import

#Network: {
	name: string
	ingressClass: string
	certificateIssuer: string | *""
	domain: string
}

#Image: {
	registry: string | *"docker.io"
	repository: string
	name: string
	tag: string
	pullPolicy: string // TODO(gio): remove?
	fullName: "\(registry)/\(repository)/\(name)"
	fullNameWithTag: "\(fullName):\(tag)"
}

#Chart: {
	chart: string
	sourceRef: #SourceRef
}

#SourceRef: {
	kind: "GitRepository" | "HelmRepository"
	name: string
	namespace: string // TODO(gio): default global.id
}

#Global: {
	id: string
	pcloudEnvName: string
	domain: string
	namespacePrefix: string
	...
}

#Release: {
	namespace: string
}

global: #Global
release: #Release

images: {
	for key, value in images {
		"\(key)": #Image & value
	}
}

charts: {
	for key, value in charts {
		"\(key)": #Chart & value
	}
}

#Helm: {
	name: string
	dependsOn: [...#Helm] | *[]
	...
}

helm: {
	for key, value in helm {
		"\(key)": #Helm & value & {
			name: key
		}
	}
}

#HelmRelease: {
	_name: string
	_chart: #Chart
	_values: _
	_dependencies: [...#Helm] | *[]

	apiVersion: "helm.toolkit.fluxcd.io/v2beta1"
	kind: "HelmRelease"
	metadata: {
		name: _name
   		namespace: release.namespace
	}
	spec: {
		interval: "1m0s"
		dependsOn: [
			for d in _dependencies {
				name: d.name
				namespace: release.namespace
			}
    	]
		chart: {
			spec: _chart
		}
		values: _values
	}
}

output: {
	for name, r in helm {
		"\(name)": #HelmRelease & {
			_name: name
			_chart: r.chart
			_values: r.values
			_dependencies: r.dependsOn
		}
	}
}
