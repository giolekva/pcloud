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

#Ingress: {
	auth: #Auth
	network: #Network
	subdomain: string
	appRoot: string | *""
	service: close({
		name: string
		port: close({ name: string }) | close({ number: int & > 0 })
	})

	_domain: "\(subdomain).\(network.domain)"
	_appRoot: appRoot
    _authProxyHTTPPortName: "http"

	out: {
		images: {
			authProxy: #Image & {
				repository: "giolekva"
				name: "auth-proxy"
				tag: "latest"
				pullPolicy: "Always"
			}
		}
		charts: {
			ingress: #Chart & {
				kind: "GitRepository"
				address: "https://github.com/giolekva/pcloud.git"
				branch: "main"
				path: "charts/ingress"
			}
			authProxy: #Chart & {
				kind: "GitRepository"
				address: "https://github.com/giolekva/pcloud.git"
				branch: "main"
				path: "charts/auth-proxy"
			}
		}
		charts: {
			for key, value in charts {
				"\(key)": #Chart & value & {
					name: key
				}
			}
		}
		helm: {
			if auth.enabled {
				"auth-proxy": {
					chart: charts.authProxy
					info: "Installing authentication proxy"
					values: {
						image: {
							repository: images.authProxy.fullName
							tag: images.authProxy.tag
							pullPolicy: images.authProxy.pullPolicy
						}
						upstream: "\(service.name).\(release.namespace).svc.cluster.local"
						whoAmIAddr: "https://accounts.\(global.domain)/sessions/whoami"
						loginAddr: "https://accounts-ui.\(global.domain)/login"
						membershipAddr: "http://memberships-api.\(global.id)-core-auth-memberships.svc.cluster.local/api/user"
						if global.privateDomain == "" {
							membershipPublicAddr: "https://memberships.\(global.domain)"
						}
						if global.privateDomain != "" {
							membershipPublicAddr: "https://memberships.\(global.privateDomain)"
						}
						groups: auth.groups
						portName: _authProxyHTTPPortName
					}
				}
			}
			"\(_domain)": {
				chart: charts.ingress
				_service: service
                info: "Generating TLS certificate for https://\(_domain)"
				values: {
					domain: _domain
					appRoot: _appRoot
					ingressClassName: network.ingressClass
					certificateIssuer: network.certificateIssuer
					service: {
						if auth.enabled {
							name: "auth-proxy"
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

ingress: {}

_ingressValidate: {
	for key, value in ingress {
		"\(key)": #Ingress & value
	}
}
