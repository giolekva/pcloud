import (
	"encoding/yaml"
	"encoding/json"
)

input: {
    network: #Network @name(Network)
    subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"
url: "https://\(_domain)"

name: "Zot"
namespace: "app-zot"
readme: "OCI-native container image registry, simplified"
description: "OCI-native container image registry, simplified"
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 24 24'><path fill='currentColor' d='M21.231 2.462L7.18 20.923h14.564V24H2.256v-2.462L16.308 3.076H2.975V0h18.256z'/></svg>"

ingress: {
	zot: {
		auth: enabled: false
		network: input.network
		subdomain: input.subdomain
		service: {
			name: "zot"
			port: number: _httpPort // TODO(gio): make optional
		}
	}
}

// TODO(gio): configure busybox
images: {
	zot: {
		registry: "ghcr.io"
		repository: "project-zot"
		name: "zot-linux-amd64"
		tag: "v2.0.3"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	zot: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/zot"
	}
	oauth2Client: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/oauth2-client"
	}
	resourceRenderer: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/resource-renderer"
	}
}

volumes: zot: size: "100Gi"

_httpPort: 80
_oauth2ClientSecretName: "oauth2-client"

helm: {
	"oauth2-client": {
		chart: charts.oauth2Client
		info: "Creating OAuth2 client"
		// TODO(gio): remove once hydra maester is installed as part of dodo itself
		dependsOn: [{
			name: "auth"
			namespace: "\(global.namespacePrefix)core-auth"
		}]
		values: {
			name: "\(release.namespace)-zot"
			secretName: _oauth2ClientSecretName
			grantTypes: ["authorization_code"]
			responseTypes: ["code"]
			scope: "openid profile email groups"
			redirectUris: ["https://\(_domain)/zot/auth/callback/oidc"]
			hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
		}
	}
	"config-renderer": {
		chart: charts.resourceRenderer
		info: "Generating Zot configuration"
		values: {
			name: "config-renderer"
			secretName: _oauth2ClientSecretName
			resourceTemplate: yaml.Marshal({
				apiVersion: "v1"
				kind: "ConfigMap"
				metadata: {
					name: _zotConfigMapName
					namespace: "\(release.namespace)"
				}
				data: {
					"config.json": json.Marshal({
						storage: rootDirectory: "/var/lib/registry"
						http: {
							address: "0.0.0.0"
							port: "5000"
							externalUrl: url
							auth: openid: providers: oidc: {
								name: "dodo:"
								issuer: "https://hydra.\(networks.public.domain)"
								clientid: "{{ .client_id }}"
								clientsecret: "{{ .client_secret }}"
								keypath: ""
								scopes: ["openid", "profile", "email", "groups"]
							}
							accessControl: {
								repositories: {
									"**": {
										defaultPolicy: ["read", "create", "update", "delete"]
										anonymousPolicy: ["read"]
									}
								}
							}
						}
						log: level: "debug"
						extensions: {
							ui: enable: true
							search: enable: true
						}
					})
				}
			})
		}
	}
	zot: {
		chart: charts.zot
		info: "Installing Zot server"
		values: {
			image: {
				repository: images.zot.fullName
				tag: images.zot.tag
				pullPolicy: images.zot.pullPolicy
			}
			service: {
				type: "ClusterIP"
				additionalAnnotations: {
					"metallb.universe.tf/address-pool": global.id
				}
				port: _httpPort
			}
			ingress: enabled: false
			mountConfig: false
			persistence: true
			pvc: {
				create: false
				name: volumes.zot.name
			}
			extraVolumes: [{
				name: "config"
				configMap: name: _zotConfigMapName
			}]
			extraVolumeMounts: [{
				name: "config"
				mountPath: "/etc/zot"
			}]
			startupProbe: {}
		}
	}
}

_zotConfigMapName: "zot-config"

help: [{
	title: "Authenticate"
	contents: """
	First generate new API key.  
	docker login \\-\\-username=**\\<YOUR-USERNAME\\>**@\(networks.public.domain) \\-\\-password=**\\<YOUR-API-KEY\\>** \(_domain)  
	docker build \\-\\-tag=\(_domain)/**\\<IMAGE-NAME\\>**:**\\<IMAGE-TAG\\>** .  
	docker push \\-\\-tag=\(_domain)/**\\<IMAGE-NAME\\>**:**\\<IMAGE-TAG\\>**
	"""
}]
