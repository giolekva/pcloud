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
	g?: #Global

	_domain: "\(subdomain).\(network.domain)"
	_appRoot: appRoot
	_authProxyName: "\(name)-auth-proxy"
    _authProxyHTTPPortName: "http"

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
					portName: _authProxyHTTPPortName
				}
			}
		}
		"\(name)-ingress": {
			chart: charts.ingress
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

#WithOut: {
	ingress: {...}
	ingress: {
		for k, v in ingress {
			"\(k)": #Ingress & v & {
				name: k
				g: global
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
