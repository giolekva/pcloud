input: {
	network: #Network
	subdomain: string
}

_domain: "\(input.subdomain).\(input.network.domain)"

readme: "penpot application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"

images: {
	postgres: {
		repository: "library"
		name: "postgres"
		tag: "15.3"
		pullPolicy: "IfNotPresent"
	}
	backend: {
		repository: "penpotapp"
		name: "backend"
		tag: "1.16.0-beta"
		pullPolicy: "IfNotPresent"
	}
	frontend: {
		repository: "penpotapp"
		name: "frontend"
		tag: "1.16.0-beta"
		pullPolicy: "IfNotPresent"
	}
	exporter: {
		repository: "penpotapp"
		name: "exporter"
		tag: "1.16.0-beta"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	postgres: {
		chart: "charts/postgresql"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	oauth2Client: {
		chart: "charts/oauth2-client"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	penpot: {
		chart: "charts/penpot"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

_oauth2SecretName: "oauth2-credentials"

helm: {
	"oauth2-client": {
		chart: charts.oauth2Client
		values: {
			name: "penpot"
			secretName: _oauth2SecretName
			grantTypes: ["authorization_code"]
			responseTypes: ["code"]
			scope: "openid profile email"
			redirectUris: ["https://\(_domain)/api/auth/oauth/oidc/callback"]
			hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
			tokenEndpointAuthMethod: "client_secret_post"
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
			auth: {
				username: "penpot"
				password: "penpot"
				database: "penpot"
			}
		}
	}
	penpot: {
		chart: charts.penpot
		values: {
			"global": {
				postgresqlEnabled: false
				redisEnabled: true // TODO(gio): provide redis from outside
			}
			fullnameOverride: "penpot"
			backend: {
				image: {
					repository: images.backend.fullName
					tag: images.backend.tag
					imagePullPolicy: images.backend.pullPolicy
				}
			}
			frontend: {
				image: {
					repository: images.frontend.fullName
					tag: images.frontend.tag
					imagePullPolicy: images.frontend.pullPolicy
				}
				ingress: {
					enabled: true
					className: input.network.ingressClass
					if input.network.certificateIssuer != "" {
						annotations: {
							"acme.cert-manager.io/http01-edit-in-place": "true"
							"cert-manager.io/cluster-issuer": input.network.certificateIssuer
						}
					}
					hosts: [_domain]
					tls: [{
						hosts: [_domain]
						secretName: "cert-\(_domain)"
					}]
				}
			}
			persistence: enabled: true
			config: {
				publicURI: _domain
				flags: "enable-login-with-oidc enable-registration enable-insecure-register disable-demo-users disable-demo-warning" // TODO(gio): remove enable-insecure-register?
				postgresql: {
					host: "postgres.\(release.namespace).svc.cluster.local"
					database: "penpot"
					username: "penpot"
					password: "penpot"
				}
				redis: host: "penpot-redis-headless.\(release.namespace).svc.cluster.local"
				providers: {
					oidc: {
						enabled: true
						baseURI: "https://hydra.\(global.domain)"
						clientID: ""
						clientSecret: ""
						authURI: ""
						tokenURI: ""
						userURI: ""
						roles: ""
						rolesAttribute: ""
						scopes: ""
						nameAttribute: "name"
						emailAttribute: "email"
					}
					existingSecret: _oauth2SecretName
					secretKeys: {
						oidcClientIDKey: "client_id"
						oidcClientSecretKey: "client_secret"
					}
				}
			}
			exporter: {
				image: {
					repository: images.exporter.fullName
					tag: images.exporter.tag
					imagePullPolicy: images.exporter.pullPolicy
				}
			}
			redis: image: tag: "7.0.8-debian-11-r16"
		}
	}
}
