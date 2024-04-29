input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"

name: "Matrix"
namespace: "app-matrix"
readme: "matrix application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"
description: "An open network for secure, decentralised communication"
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 24 24'><path fill='currentColor' d='M.632.55v22.9H2.28V24H0V0h2.28v.55zm7.043 7.26v1.157h.033a3.312 3.312 0 0 1 1.117-1.024c.433-.245.936-.365 1.5-.365c.54 0 1.033.107 1.481.314c.448.208.785.582 1.02 1.108c.254-.374.6-.706 1.034-.992c.434-.287.95-.43 1.546-.43c.453 0 .872.056 1.26.167c.388.11.716.286.993.53c.276.245.489.559.646.951c.152.392.23.863.23 1.417v5.728h-2.349V11.52c0-.286-.01-.559-.032-.812a1.755 1.755 0 0 0-.18-.66a1.106 1.106 0 0 0-.438-.448c-.194-.11-.457-.166-.785-.166c-.332 0-.6.064-.803.189a1.38 1.38 0 0 0-.48.499a1.946 1.946 0 0 0-.231.696a5.56 5.56 0 0 0-.06.785v4.768h-2.35v-4.8c0-.254-.004-.503-.018-.752a2.074 2.074 0 0 0-.143-.688a1.052 1.052 0 0 0-.415-.503c-.194-.125-.476-.19-.854-.19c-.111 0-.259.024-.439.074c-.18.051-.36.143-.53.282a1.637 1.637 0 0 0-.439.595c-.12.259-.18.6-.18 1.02v4.966H5.46V7.81zm15.693 15.64V.55H21.72V0H24v24h-2.28v-.55z'/></svg>"

images: {
	matrix: {
		repository: "matrixdotorg"
		name: "synapse"
		tag: "v1.104.0"
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
	oauth2Client: {
		chart: "charts/oauth2-client"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
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

_oauth2ClientSecretName: "oauth2-client"

helm: {
	"oauth2-client": {
		chart: charts.oauth2Client
		values: {
			name: "oauth2-client"
			secretName: _oauth2ClientSecretName
			grantTypes: ["authorization_code"]
			responseTypes: ["code"]
			scope: "openid profile"
			redirectUris: ["https://\(_domain)/_synapse/client/oidc/callback"]
			hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
		}
	}
	matrix: {
		dependsOn: [{
			name: "postgres"
			namespace: release.namespace
		}]
		chart: charts.matrix
		values: {
			domain: global.domain
			subdomain: input.subdomain
			oauth2: {
				secretName: "oauth2-client"
				issuer: "https://hydra.\(global.domain)"
			}
			postgresql: {
				host: "postgres"
				port: 5432
				database: "matrix"
				user: "matrix"
				password: "matrix"
			}
			certificateIssuer: issuerPublic
			ingressClassName: ingressPublic
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
