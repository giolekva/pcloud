input: {
	network: #Network
	subdomain: string
}

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
			certificateIssuer: _issuerPublic
			ingressClassName: _ingressPublic
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
				repository: images.postgres.imageName
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
