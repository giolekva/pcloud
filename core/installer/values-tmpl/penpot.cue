input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"
url: "https://\(_domain)"

name: "Penpot"
namespace: "app-penpot"
readme: "penpot application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"
description: "Penpot is the first Open Source design and prototyping platform meant for cross-domain teams. Non dependent on operating systems, Penpot is web based and works with open standards (SVG). Penpot invites designers all over the world to fall in love with open source while getting developers excited about the design process in return."
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 24 24'><path fill='currentColor' d='M7.654 0L5.13 3.554v2.01L2.934 6.608l-.02-.009v13.109l8.563 4.045L12 24l.523-.247l8.563-4.045V6.6l-.017.008l-2.196-1.045V3.555l-.077-.108L16.349.001l-2.524 3.554v.004L11.989.973l-1.823 2.566l-.065-.091zm.447 2.065l.976 1.374H6.232l.964-1.358zm8.694 0l.976 1.374h-2.845l.965-1.358zm-4.36.971l.976 1.375h-2.845l.965-1.359zM5.962 4.132h1.35v4.544l-1.35-.638Zm2.042 0h1.343v5.506l-1.343-.635zm6.652 0h1.35V9l-1.35.637zm2.042 0h1.343v3.905l-1.343.634zm-6.402.972h1.35v5.62l-1.35-.638zm2.042 0h1.343v4.993l-1.343.634zm6.534 1.493l1.188.486l-1.188.561zM5.13 6.6v1.047l-1.187-.561ZM3.96 8.251l7.517 3.55v10.795l-7.516-3.55zm16.08 0v10.794l-7.517 3.55V11.802z'/></svg>"

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
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/postgresql"
	}
	oauth2Client: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/oauth2-client"
	}
	penpot: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/penpot"
	}
}

_oauth2SecretName: "oauth2-credentials"

helm: {
	"oauth2-client": {
		chart: charts.oauth2Client
		values: {
			name: "\(release.namespace)-penpot"
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
