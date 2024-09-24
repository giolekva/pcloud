import (
	"strings"
)

#Global: {
	id: string | *""
	pcloudEnvName: string | *""
	domain: string | *""
    privateDomain: string | *""
    contactEmail: string | *""
    adminPublicKey: string | *""
    publicIP: [...string] | *[]
    nameserverIP: [...string] | *[]
	namespacePrefix: string | *""
	network: #EnvNetwork
}

#Network: {
	name: string
	ingressClass: string
	certificateIssuer: string | *""
	domain: string
	allocatePortAddr: string
	reservePortAddr: string
	deallocatePortAddr: string
}

#Networks: {
	...
}

networks: #Networks

clusters: [...#Cluster] | *[]
clusterMap: {
	for c in clusters {
		"\(strings.ToLower(c.name))": c
	}
}

#Ingress: #WithOut & {
	name: string
	auth: #Auth
	network: #Network
	subdomain: string
	appRoot: string | *""
	service: close({
		name: string
		port: close({ name: string }) | close({ number: int & > 0 })
	})
	cluster?: #Cluster
	_cluster: cluster
	g?: #Global

	_domain: "\(subdomain).\(network.domain)"
	_appRoot: appRoot
	_authProxyName: "\(name)-auth-proxy"
    _authProxyHTTPPortName: "http"

	if _cluster != _|_ {
		clusterProxy: {
			"\(name)": {
				from: _domain
				_sanitizedDomain: strings.Replace(_domain, ".", "-", -1)
				to: "\(_sanitizedDomain).\(_cluster.name).cluster.\(global.privateDomain)"
			}
		}
	}
	images: {
		authProxy: {
			repository: "giolekva"
			name: "auth-proxy"
			tag: "latest"
			pullPolicy: "Always"
		}
	}
	charts: {
		ingress: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/ingress"
		}
		authProxy: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/auth-proxy"
		}
	}
	helm: {
		if auth.enabled {
			"\(name)-auth-proxy": {
				chart: charts.authProxy
				info: "Installing authentication proxy"
				// NOTE(gio): Force to install in default cluster.
				cluster: null
				_name: name
				values: {
					name: _authProxyName
					image: {
						repository: images.authProxy.fullName
						tag: images.authProxy.tag
						pullPolicy: images.authProxy.pullPolicy
					}
					upstream: "\(service.name).\(release.namespace).svc.cluster.local"
					whoAmIAddr: "https://accounts.\(g.domain)/sessions/whoami"
					loginAddr: "https://accounts-ui.\(g.domain)/login"
					membershipAddr: "http://memberships-api.\(g.namespacePrefix)core-auth-memberships.svc.cluster.local/api/user"
					if g.privateDomain == "" {
						membershipPublicAddr: "https://memberships.\(g.domain)"
					}
					if g.privateDomain != "" {
						membershipPublicAddr: "https://memberships.\(g.privateDomain)"
					}
					groups: auth.groups
					noAuthPathPrefixes: strings.Join(auth.noAuthPathPrefixes, ",")
					portName: _authProxyHTTPPortName
				}
			}
		}
		if _cluster != _|_ {
			"\(name)-ingress-\(_cluster.name)": {
				chart: charts.ingress
				cluster: _cluster
				_service: service
				_sanitizedDomain: strings.Replace(_domain, ".", "-", -1)
				_clusterDomain: "\(_sanitizedDomain).\(cluster.name).cluster.\(global.privateDomain)"
				info: "Configuring secure route to \(cluster.name) cluster"
				annotations: {
					// TODO(gio): Change type to cluster-gateway or sth similar.
					"dodo.cloud/resource-type": "ingress"
					"dodo.cloud/resource.ingress.host": "https://\(_clusterDomain)"
				}
				values: {
					domain: _clusterDomain
					ingressClassName: cluster.ingressClassName
					certificateIssuer: ""
					annotations: {
						"nginx.ingress.kubernetes.io/force-ssl-redirect": "false"
						"nginx.ingress.kubernetes.io/ssl-redirect": "false"
					}
					service: {
						name: _service.name
						if _service.port.name != _|_ {
							port: name: _service.port.name
						}
						if _service.port.number != _|_ {
							port: number: _service.port.number
						}
					}
				}
			}
			"\(release.appInstanceId)-\(name)-ingress": {
				chart: charts.ingress
				// NOTE(gio): Force to install in default cluster.
				cluster: null
				// TODO(gio): take it from input.network.namespace
				targetNamespace: "\(global.namespacePrefix)ingress-private"
				_service: service
				info: "Generating TLS certificate for https://\(_domain)"
				annotations: {
					"dodo.cloud/resource-type": "ingress"
					"dodo.cloud/resource.ingress.host": "https://\(_domain)"
				}
				values: {
					domain: _domain
					appRoot: _appRoot
					ingressClassName: network.ingressClass
					certificateIssuer: network.certificateIssuer
					service: {
						if auth.enabled {
							name: _authProxyName
							port: name: _authProxyHTTPPortName
						}
						if !auth.enabled {
							// TODO(gio): make this variables part of the env configuration
							name: "proxy-backend-service"
							port: name: "http"
						}
					}
				}
			}
		}
		if _cluster == _|_ {
			"\(name)-ingress": {
				chart: charts.ingress
				// NOTE(gio): Force to install in default cluster.
				cluster: null
				_service: service
				info: "Generating TLS certificate for https://\(_domain)"
				annotations: {
					"dodo.cloud/resource-type": "ingress"
					"dodo.cloud/resource.ingress.host": "https://\(_domain)"
				}
				values: {
					domain: _domain
					appRoot: _appRoot
					ingressClassName: network.ingressClass
					certificateIssuer: network.certificateIssuer
					service: {
						if auth.enabled {
							name: _authProxyName
							port: name: _authProxyHTTPPortName
						}
						if !auth.enabled {
							name: _service.name
							if _service.port.name != _|_ {
								port: name: _service.port.name
							}
							if _service.port.number != _|_ {
								port: number: _service.port.number
							}
						}
					}
				}
			}
		}
	}
}

#WithOut: {
	cluster?: #Cluster
	_cluster: cluster
	ingress: {...}
	ingress: {
		for k, v in ingress {
			"\(k)": #Ingress & v & {
				name: k
				g: global
				if _cluster != _|_ {
					cluster: _cluster
				}
			}
		}
		...
	}
	clusterProxy: {
		for k, v in ingress {
			for i, j in v.clusterProxy {
				"\(k)-\(i)": j
			}
		}
		...
	}
	images: {
		for k, v in ingress {
			for x, y in v.images {
				"\(x)": y
			}
		}
	}
	charts: {
		for k, v in ingress {
			for x, y in v.charts {
				"\(x)": y
			}
		}
	}
	helmR: {
		for k, v in ingress {
			for x, y in v.helmR {
				"\(x)": y
			}
		}
		...
	}
	...
}
