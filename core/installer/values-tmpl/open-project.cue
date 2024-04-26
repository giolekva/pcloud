input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"
url: "https://\(_domain)"

name: "OpenProject"
namespace: "app-open-project"
readme: "Open source project management software. Powerful classic, agile or hybrid project management in a secure environment."
description: "Open source project management software. Powerful classic, agile or hybrid project management in a secure environment."
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 24 24'><path fill='currentColor' d='M19.35.37h-1.86a4.63 4.63 0 0 0-4.652 4.624v5.609H4.652A4.63 4.63 0 0 0 0 15.23v3.721c0 2.569 2.083 4.679 4.652 4.679h1.86c2.57 0 4.652-2.11 4.652-4.679v-3.72c0-.063 0-.158-.005-.158H8.373v3.88c0 1.026-.835 1.886-1.861 1.886h-1.86c-1.027 0-1.861-.864-1.861-1.886V15.23a1.84 1.84 0 0 1 1.86-1.833h14.697c2.57 0 4.652-2.11 4.652-4.679V4.997A4.63 4.63 0 0 0 19.35.37m1.861 8.345c0 1.026-.835 1.886-1.861 1.886h-3.721V4.997a1.84 1.84 0 0 1 1.86-1.833h1.86a1.84 1.84 0 0 1 1.862 1.833zm-8.373 9.706v.03c0 .746.629 1.344 1.396 1.344s1.395-.594 1.395-1.34v-3.384h-2.791z'/></svg>"

_httpPort: 8080
ingress: {
	gerrit: {
		auth: enabled: false
		network: input.network
		subdomain: input.subdomain
		service: {
			name: "open-project"
			port: number: _httpPort
		}
	}
}

images: {
	openProject: {
		repository: "openproject"
		name: "openproject"
		tag: "13.4.1"
		pullPolicy: "Always"
	}
	postgres: {
		repository: "library"
		name: "postgres"
		tag: "15.3"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	openProject: {
		chart: "charts/openproject"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	volume: {
		chart: "charts/volumes"
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

volumes: {
	openProject: {
		name: "open-project"
		accessMode: "ReadWriteMany"
		size: "50Gi"
	}
}

helm: {
	"open-project": {
		chart: charts.openProject
		values: {
			image: {
				registry: images.openProject.registry
				repository: images.openProject.imageName
				tag: images.openProject.tag
				imagePullPolicy: images.openProject.pullPolicy
			}
			nameOverride: "open-project"
			ingress: enabled: false
			memcached: bundled: true
			s3: enabled: false
			openproject: {
				host: _domain
				https: false
				hsts: false
				oidc: enabled: false // TODO(gio): enable
				admin_user: {
					password: "admin"
					password_reset: false
					name: "admin"
					mail: "op-admin@\(global.domain)"
				}
			}
			persistence: {
				enabled: true
				existingClaim: volumes.openProject.name
			}
			postgresql: {
				bundled: false
				connection: {
					host: "postgres.\(release.namespace).svc.cluster.local"
					port: 5432
				}
				auth: {
					username: "openproject"
					password: "openproject"
					database: "openproject"
				}
			}
			service: {
				enabled: true
				type: "ClusterIP"
			}
			initDb: {
				image: {
					registry: images.postgres.registry
					repository: images.postgres.imageName
					tag: images.postgres.tag
					imagePullPolicy: images.postgres.pullPolicy
				}
			}
		}
	}
	"open-project-volume": {
		chart: charts.volume
		values: volumes.openProject
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
						CREATE USER openproject WITH PASSWORD 'openproject';
						CREATE DATABASE openproject WITH OWNER = openproject ENCODING = UTF8 LOCALE = 'C' TEMPLATE = template0;
						"""
					}
				}
				persistence: {
					size: "50Gi"
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
