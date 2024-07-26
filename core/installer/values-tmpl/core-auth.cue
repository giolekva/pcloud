input: {
	network: #Network
	subdomain: string
}

name: "core-auth"
namespace: "core-auth"

_userSchema: ###"""
{
  "$id": "https://schemas.ory.sh/presets/kratos/quickstart/email-password/identity.schema.json",
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "User",
  "type": "object",
  "properties": {
	"traits": {
	  "type": "object",
	  "properties": {
		"username": {
		  "type": "string",
		  "format": "username",
		  "title": "Username",
		  "minLength": 3,
		  "ory.sh/kratos": {
			"credentials": {
			  "password": {
				"identifier": true
			  }
			}
		  }
		}
	  },
	  "additionalProperties": false
	}
  }
}
"""###

images: {
	kratos: {
		repository: "oryd"
		name: "kratos"
		tag: "v1.1.0-distroless"
		pullPolicy: "IfNotPresent"
	}
	hydra: {
		repository: "oryd"
		name: "hydra"
		tag: "v2.2.0-distroless"
		pullPolicy: "IfNotPresent"
	}
	ui: {
		repository: "giolekva"
		name: "auth-ui"
		tag: "latest"
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
	auth: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/auth"
	}
	postgres: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/postgresql"
	}
}

helm: {
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
						CREATE USER kratos WITH PASSWORD 'kratos';
						CREATE USER hydra WITH PASSWORD 'hydra';
						CREATE DATABASE kratos WITH OWNER = kratos;
						CREATE DATABASE hydra WITH OWNER = hydra;
						"""
					}
				}
				persistence: {
					size: "1Gi"
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
	auth: {
		chart: charts.auth
		dependsOn: [{
			name: "ingress-nginx"
			namespace: "\(global.namespacePrefix)ingress-private"
		}, {
			name: "postgres"
			namespace: release.namespace
		}]
		values: {
			kratos: {
				fullnameOverride: "kratos"
				image: {
					repository: images.kratos.fullName
					tag: images.kratos.tag
					pullPolicy: images.kratos.pullPolicy
				}
				service: {
					admin: {
						enabled: true
						type: "ClusterIP"
						port: 80
						name: "http"
					}
					public: {
						enabled: true
						type: "ClusterIP"
						port: 80
						name: "http"
					}
				}
				ingress: {
					admin: enabled: false
					public: {
						enabled: true
						className: input.network.ingressClass
						annotations: {
							"acme.cert-manager.io/http01-edit-in-place": "true"
							"cert-manager.io/cluster-issuer": input.network.certificateIssuer
						}
						hosts: [{
							host: "accounts.\(input.network.domain)"
							paths: [{
								path: "/"
								pathType: "Prefix"
							}]
						}]
						tls: [{
							hosts: ["accounts.\(input.network.domain)"]
							secretName: "cert-accounts.\(input.network.domain)"
						}]
					}
				}
				secret: {
					enabled: true
				}
				kratos: {
					automigration: {
						enabled: true
					}
					development: false
					courier: {
						enabled: false
					}
					config: {
						version: "v0.7.1-alpha.1"
						dsn: "postgres://kratos:kratos@postgres.\(global.namespacePrefix)core-auth.svc:5432/kratos?sslmode=disable&max_conns=20&max_idle_conns=4"
						serve: {
							public: {
								base_url: "https://accounts.\(input.network.domain)"
								cors: {
									enabled: true
									debug: false
									allow_credentials: true
									allowed_origins: [
										"https://\(input.network.domain)",
										"https://*.\(input.network.domain)",
								]
								}
							}
							admin: {
								base_url: "https://kratos-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
							}
						}
						selfservice: {
							default_browser_return_url: "https://accounts-ui.\(input.network.domain)"
							allowed_return_urls: [
								"https://*.\(input.network.domain)/",
								// TODO(gio): replace with input.network.privateSubdomain
								"https://*.\(global.privateDomain)",
						    ]
							methods: {
								password: {
									enabled: true
								}
							}
							flows: {
								error: {
									ui_url: "https://accounts-ui.\(input.network.domain)/error"
								}
								settings: {
									ui_url: "https://accounts-ui.\(input.network.domain)/settings"
									privileged_session_max_age: "15m"
								}
								recovery: {
									enabled: false
								}
								verification: {
									enabled: false
								}
								logout: {
									after: {
										default_browser_return_url: "https://accounts-ui.\(input.network.domain)/login"
									}
								}
								login: {
									ui_url: "https://accounts-ui.\(input.network.domain)/login"
									lifespan: "10m"
									after: {
										password: {
											default_browser_return_url: "https://accounts-ui.\(input.network.domain)/"
										}
									}
								}
								registration: {
									lifespan: "10m"
									ui_url: "https://accounts-ui.\(input.network.domain)/register"
									after: {
										password: {
											hooks: [{
												hook: "session"
											}]
											default_browser_return_url: "https://accounts-ui.\(input.network.domain)/"
										}
									}
								}
							}
						}
						log: {
							level: "debug"
							format: "text"
							leak_sensitive_values: true
						}
						cookies: {
							path: "/"
							same_site: "None"
							domain: input.network.domain
						}
						secrets: {
							cookie: ["PLEASE-CHANGE-ME-I-AM-VERY-INSECURE"]
						}
						hashers: {
							argon2: {
								parallelism: 1
								memory: "128MB"
								iterations: 2
								salt_length: 16
								key_length: 16
								}
						}
						identity: {
							schemas: [{
								id: "user"
								url: "file:///etc/config/identity.schema.json"
							}]
							default_schema_id: "user"
						}
						courier: {
							smtp: {
								connection_uri: "smtps://test-z1VmkYfYPjgdPRgPFgmeZ31esT9rUgS%40\(input.network.domain):iW%213Kk%5EPPLFrZa%24%21bbpTPN9Wv3b8mvwS6ZJvMLtce%23A2%2A4MotD@mx1.\(input.network.domain)"
							}
						}
					}
					identitySchemas: {
                        "identity.schema.json": _userSchema
					}
				}
			}
			hydra: {
				fullnameOverride: "hydra"
				image: {
					repository: images.hydra.fullName
					tag: images.hydra.tag
					pullPolicy: images.hydra.pullPolicy
				}
				service: {
					admin: {
						enabled: true
						type: "ClusterIP"
						port: 80
						name: "http"
					}
					public: {
						enabled: true
						type: "ClusterIP"
						port: 80
						name: "http"
					}
				}
				ingress: {
					admin: enabled: false
					public: {
						enabled: true
						className: input.network.ingressClass
						annotations: {
							"acme.cert-manager.io/http01-edit-in-place": "true"
							"cert-manager.io/cluster-issuer": input.network.certificateIssuer
						}
						hosts: [{
							host: "hydra.\(input.network.domain)"
							paths: [{
								path: "/"
								pathType: "Prefix"
							}]
						}]
						tls: [{
							hosts: ["hydra.\(input.network.domain)"]
							secretName: "cert-hydra.\(input.network.domain)"
						}]
					}
				}
				secret: {
					enabled: true
				}
				maester: {
					enabled: false
				}
				hydra: {
					automigration: {
						enabled: true
					}
					config: {
						version: "v1.10.6"
						dsn: "postgres://hydra:hydra@postgres.\(global.namespacePrefix)core-auth.svc:5432/hydra?sslmode=disable&max_conns=20&max_idle_conns=4"
						serve: {
							cookies: {
								same_site_mode: "None"
							}
							public: {
								cors: {
									enabled: true
									debug: false
									allow_credentials: true
									allowed_origins: [
										"https://\(input.network.domain)",
										"https://*.\(input.network.domain)"
								]
								}
							}
							admin: {
								cors: {
									allowed_origins: [
										"https://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
								]
								}
								tls: {
									allow_termination_from: [
										"0.0.0.0/0",
										"10.42.0.0/16",
										"10.43.0.0/16",
								]
								}
							}
							tls: {
								allow_termination_from: [
									"0.0.0.0/0",
									"10.42.0.0/16",
									"10.43.0.0/16",
							]
							}
						}
						urls: {
							self: {
								public: "https://hydra.\(input.network.domain)"
								issuer: "https://hydra.\(input.network.domain)"
							}
							consent: "https://accounts-ui.\(input.network.domain)/consent"
							login: "https://accounts-ui.\(input.network.domain)/login"
							logout: "https://accounts-ui.\(input.network.domain)/logout"
						}
						secrets: {
							system: ["youReallyNeedToChangeThis"]
						}
						oidc: {
							subject_identifiers: {
								supported_types: [
									"pairwise",
									"public",
							]
								pairwise: {
									salt: "youReallyNeedToChangeThis"
								}
							}
						}
						log: {
							level: "trace"
							leak_sensitive_values: false
						}
					}
				}
			}
			ui: {
				certificateIssuer: input.network.certificateIssuer
				ingressClassName: input.network.ingressClass
				domain: input.network.domain
				hydra: "hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
				enableRegistration: false
				image: {
					repository: images.ui.fullName
					tag: images.ui.tag
					pullPolicy: images.ui.pullPolicy
				}
			}
		}
	}
}
