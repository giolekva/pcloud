#Input: {
	network: #Network
	subdomain: string
}

input: #Input

_domain: "\(input.subdomain).\(input.network.domain)"

readme: "matrix application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"

images: {
	matrix: {
		repository: "matrixdotorg"
		name: "synapse"
		tag: "latest"
		pullPolicy: "IfNotPresent"
	}
	postgres: {
		repository: "library"
		name: "postgres"
		tag: "15.3"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	matrix: {
		chart: "charts/matrix"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	postgres: {
		chart: "charts/postgresql"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	matrix: {
		dependsOn: [
			postgres
	    ]
		chart: charts.matrix
		values: {
			domain: global.domain
			subdomain: input.subdomain
			oauth2: {
				hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
				hydraPublic: "https://hydra.\(global.domain)"
				secretName: "oauth2-client"
			}
			postgresql: {
				host: "postgres"
				port: 5432
				database: "matrix"
				user: "matrix"
				password: "matrix"
			}
			certificateIssuer: "\(global.id)-public"
			ingressClassName: "\(global.pcloudEnvName)-ingress-public"
			configMerge: {
				configName: "config-to-merge"
				fileName: "to-merge.yaml"
			}
			image: {
				repository: images.matrix.fullName
				tag: images.matrix.tag
				pullPolicy: images.matrix.pullPolicy
			}
		}
	}
	postgres: {
		chart: charts.postgres
		values: {
			fullnameOverride: "postgres"
			image: {
				registry: images.postgres.registry
				repository: "\(images.postgres.repository)/\(images.postgres.name)"
				tag: images.postgres.tag
				pullPolicy: images.postgres.pullPolicy
			}
			service: {
				type: "ClusterIP"
				port: 5432
			}
			primary: {
				initdb: {
					scripts: {
						"init.sql": """
						CREATE USER matrix WITH PASSWORD 'matrix';
						CREATE DATABASE matrix WITH OWNER = matrix ENCODING = UTF8 LOCALE = 'C' TEMPLATE = template0;
						"""
					}
				}
				persistence: {
					size: "10Gi"
				}
				securityContext: {
					enabled: true
					fsGroup: 0
				}
				containerSecurityContext: {
					enabled: true
					runAsUser: 0
				}
			}
			volumePermissions: {
				securityContext: {
					runAsUser: 0
				}
			}
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
