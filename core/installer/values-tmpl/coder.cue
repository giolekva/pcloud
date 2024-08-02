input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"
url: "https://\(_domain)"

name: "Coder"
namespace: "app-coder"
readme: "VSCode in the browser"
description: readme
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50px' height='50px' viewBox='0 0 16 16'><path fill='currentColor' fill-rule='evenodd' d='M11.573.275a1.203 1.203 0 0 0-.191.073c-.039.021-1.383 1.172-2.986 2.558C6.792 4.291 5.462 5.424 5.44 5.424c-.022 0-.664-.468-1.427-1.04c-.762-.571-1.428-1.057-1.48-1.078c-.15-.063-.468-.05-.613.024C1.754 3.416.189 4.975.094 5.15a.741.741 0 0 0 .04.766c.041.057.575.557 1.185 1.11c.611.553 1.107 1.015 1.102 1.026c-.004.012-.495.442-1.091.957c-.596.514-1.122.981-1.168 1.036a.746.746 0 0 0-.069.804c.096.175 1.659 1.734 1.827 1.821c.166.087.497.089.653.005c.059-.031.7-.502 1.424-1.046l1.318-.988l.109.1l2.73 2.473c1.846 1.671 2.666 2.396 2.772 2.453l.15.08h1.348l1.631-.814c1.5-.748 1.64-.823 1.748-.942c.213-.237.197.241.197-5.738c0-5.821.009-5.468-.151-5.699c-.058-.084-.41-.331-1.634-1.148c-.857-.572-1.613-1.065-1.68-1.095c-.1-.045-.187-.056-.482-.063c-.237-.005-.401.004-.48.027m1.699 2.305l1.233.82l.001 4.82l.001 4.82l-1.205.6l-1.204.6h-.569L8.66 11.644c-1.578-1.428-2.912-2.616-2.963-2.641c-.199-.094-.5-.101-.661-.014c-.034.018-.651.475-1.372 1.015c-.721.541-1.322.983-1.335.983c-.03 0-.477-.448-.461-.462c.673-.577 2.078-1.794 2.182-1.891c.086-.081.169-.192.21-.28c.057-.127.065-.174.054-.343c-.01-.158-.028-.223-.091-.324c-.053-.086-.454-.466-1.229-1.167l-1.15-1.04l.231-.233a1.83 1.83 0 0 1 .256-.234c.013 0 .644.465 1.4 1.033c1.496 1.123 1.537 1.148 1.81 1.116a.968.968 0 0 0 .253-.069c.062-.029.503-.39.979-.802L7.96 5.265a5929.2 5929.2 0 0 0 2.187-1.89a191.687 191.687 0 0 1 1.879-1.614c.008-.001.568.368 1.246.819M11.64 4.257a1.5 1.5 0 0 0-.16.051c-.059.021-1.079.738-2.267 1.593C6.867 7.59 6.92 7.547 6.851 7.854a.556.556 0 0 0 0 .292c.068.307.017.264 2.362 1.953c1.188.855 2.214 1.576 2.28 1.601c.347.133.743-.029.929-.38l.071-.133V4.813l-.071-.133a.76.76 0 0 0-.369-.356c-.127-.056-.324-.088-.413-.067m-.66 4.5l-.007.757l-1.046-.75A41.313 41.313 0 0 1 8.881 8c0-.007.471-.351 1.046-.764l1.046-.75l.007.757a95.51 95.51 0 0 1 0 1.514'/></svg>"

ingress: {
	coder: {
		auth: enabled: false
		network: input.network
		subdomain: input.subdomain
		service: {
			name: "coder"
			port: name: "http"
		}
	}
}

images: {
	postgres: {
		repository: "library"
		name: "postgres"
		tag: "15.3"
		pullPolicy: "IfNotPresent"
	}
	coder: {
		registry: "ghcr.io"
		repository: "coder"
		name: "coder"
		tag: "latest"
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
	coder: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/coder"
	}
	oauth2Client: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/oauth2-client"
	}
}

_oauth2ClientSecretName: "oauth2-credentials"

helm: {
	"oauth2-client": {
		chart: charts.oauth2Client
		values: {
			name: "\(release.namespace)-coder"
			secretName: _oauth2ClientSecretName
			grantTypes: ["authorization_code"]
			responseTypes: ["code"]
			scope: "openid profile email"
			redirectUris: ["\(url)/api/v2/users/oidc/callbackzot"]
			hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
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
				username: "coder"
				password: "coder"
				database: "coder"
			}
		}
	}
	coder: {
		chart: charts.coder
		values: coder: {
			image: {
				repo: images.coder.fullName
				tag: images.coder.tag
				pullPolicy: images.coder.pullPolicy
			}
			envUseClusterAccessURL: false
			env: [{
				name: "CODER_ACCESS_URL"
				value: url
			}, {
				name: "CODER_PG_CONNECTION_URL"
				value: "postgres://coder:coder@postgres.\(release.namespace).svc.cluster.local:5432/coder?sslmode=disable"
			}, {
				name: "CODER_OIDC_ISSUER_URL"
				value: "https://hydra.\(networks.public.domain)"
			}, {
				name: "CODER_OIDC_EMAIL_DOMAIN"
				value: networks.public.domain
			}, {
				name: "CODER_OIDC_CLIENT_ID"
				valueFrom: {
					secretKeyRef: {
						name: _oauth2ClientSecretName
						key: "client_id"
					}
				}
			}, {
				name: "CODER_OIDC_CLIENT_SECRET"
				valueFrom: {
					secretKeyRef: {
						name: _oauth2ClientSecretName
						key: "client_secret"
					}
				}
			}]
		}
	}
}
