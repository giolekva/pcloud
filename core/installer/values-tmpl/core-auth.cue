input: {
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
		tag: "v0.13.0"
		pullPolicy: "IfNotPresent"
	}
	hydra: {
		repository: "oryd"
		name: "hydra"
		tag: "v2.1.2"
		pullPolicy: "IfNotPresent"
	}
	"hydra-maester": {
		repository: "giolekva"
		name: "ory-hydra-maester"
		tag: "latest"
		pullPolicy: "Always"
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
		chart: "charts/auth"
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
		dependsOn: [postgres]
		dependsOnExternal: [{
			name: "ingress-nginx"
			namespace: "\(global.namespacePrefix)ingress-private"
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
					admin: {
						enabled: true
						className: _ingressPrivate
						hosts: [{
							host: "kratos.\(global.privateDomain)"
							paths: [{
								path: "/"
								pathType: "Prefix"
							}]
						}]
						tls: [{
							hosts: [
								"kratos.\(global.privateDomain)"
						]
						}]
					}
					public: {
						enabled: true
						className: _ingressPublic
						annotations: {
							"acme.cert-manager.io/http01-edit-in-place": "true"
							"cert-manager.io/cluster-issuer": _issuerPublic
						}
						hosts: [{
							host: "accounts.\(global.domain)"
							paths: [{
								path: "/"
								pathType: "Prefix"
							}]
						}]
						tls: [{
							hosts: ["accounts.\(global.domain)"]
							secretName: "cert-accounts.\(global.domain)"
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
								base_url: "https://accounts.\(global.domain)"
								cors: {
									enabled: true
									debug: false
									allow_credentials: true
									allowed_origins: [
										"https://\(global.domain)",
										"https://*.\(global.domain)",
								]
								}
							}
							admin: {
								base_url: "https://kratos.\(global.privateDomain)/"
							}
						}
						selfservice: {
							default_browser_return_url: "https://accounts-ui.\(global.domain)"
							methods: {
								password: {
									enabled: true
								}
							}
							flows: {
								error: {
									ui_url: "https://accounts-ui.\(global.domain)/error"
								}
								settings: {
									ui_url: "https://accounts-ui.\(global.domain)/settings"
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
										default_browser_return_url: "https://accounts-ui.\(global.domain)/login"
									}
								}
								login: {
									ui_url: "https://accounts-ui.\(global.domain)/login"
									lifespan: "10m"
									after: {
										password: {
											default_browser_return_url: "https://accounts-ui.\(global.domain)/"
										}
									}
								}
								registration: {
									lifespan: "10m"
									ui_url: "https://accounts-ui.\(global.domain)/register"
									after: {
										password: {
											hooks: [{
												hook: "session"
											}]
											default_browser_return_url: "https://accounts-ui.\(global.domain)/"
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
							domain: global.domain
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
								connection_uri: "smtps://test-z1VmkYfYPjgdPRgPFgmeZ31esT9rUgS%40\(global.domain):iW%213Kk%5EPPLFrZa%24%21bbpTPN9Wv3b8mvwS6ZJvMLtce%23A2%2A4MotD@mx1.\(global.domain)"
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
					admin: {
						enabled: true
						className: _ingressPrivate
						hosts: [{
							host: "hydra.\(global.privateDomain)"
							paths: [{
								path: "/"
								pathType: "Prefix"
							}]
							   }]
						tls: [{
							hosts: ["hydra.\(global.privateDomain)"]
						}]
					}
					public: {
						enabled: true
						className: _ingressPublic
						annotations: {
							"acme.cert-manager.io/http01-edit-in-place": "true"
							"cert-manager.io/cluster-issuer": _issuerPublic
						}
						hosts: [{
							host: "hydra.\(global.domain)"
							paths: [{
								path: "/"
								pathType: "Prefix"
							}]
						}]
						tls: [{
							hosts: ["hydra.\(global.domain)"]
							secretName: "cert-hydra.\(global.domain)"
						}]
					}
				}
				secret: {
					enabled: true
				}
				maester: {
					enabled: true
				}
				"hydra-maester": {
					adminService: {
						name: "hydra-admin"
						port: 80
					}
					image: {
						repository: images["hydra-maester"].fullName
						tag: images["hydra-maester"].tag
						pullPolicy: images["hydra-maester"].pullPolicy
					}
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
										"https://\(global.domain)",
										"https://*.\(global.domain)"
								]
								}
							}
							admin: {
								cors: {
									allowed_origins: [
										"https://hydra.\(global.privateDomain)"
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
								public: "https://hydra.\(global.domain)"
								issuer: "https://hydra.\(global.domain)"
							}
							consent: "https://accounts-ui.\(global.domain)/consent"
							login: "https://accounts-ui.\(global.domain)/login"
							logout: "https://accounts-ui.\(global.domain)/logout"
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
				certificateIssuer: _issuerPublic
				ingressClassName: _ingressPublic
				domain: global.domain
				internalDomain: global.privateDomain
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
