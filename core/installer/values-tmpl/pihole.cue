input: {
	network: #Network
	subdomain: string
	auth: #Auth
}

_domain: "\(input.subdomain).\(input.network.domain)"

name: "Pi-hole"
namespace: "app-pihole"
readme: "Installs pihole at https://\(_domain)"
description: "Pi-hole is a Linux network-level advertisement and Internet tracker blocking application which acts as a DNS sinkhole and optionally a DHCP server, intended for use on a private network."
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 24 24'><path fill='currentColor' d='M4.344 0c.238 4.792 3.256 7.056 6.252 7.376c.165-1.692-4.319-5.6-4.319-5.6c-.008-.011.009-.025.019-.014c0 0 4.648 4.01 5.423 5.645c2.762-.15 5.196-1.947 5-4.912c0 0-4.12-.613-5 4.618C11.48 2.753 8.993 0 4.344 0zM12 7.682v.002a3.68 3.68 0 0 0-2.591 1.077L4.94 13.227a3.683 3.683 0 0 0-.86 1.356a3.31 3.31 0 0 0-.237 1.255A3.681 3.681 0 0 0 4.92 18.45l4.464 4.466a3.69 3.69 0 0 0 2.251 1.06l.002.001c.093.01.187.015.28.017l-.1-.008c.06.003.117.009.177.009l-.077-.001L12 24l-.004-.005a3.68 3.68 0 0 0 2.61-1.077l4.469-4.465a3.683 3.683 0 0 0 1.006-1.888l.012-.063a3.682 3.682 0 0 0 .057-.541l.003-.061c0-.017.003-.05.004-.06h-.002a3.683 3.683 0 0 0-1.077-2.607l-4.466-4.468a3.694 3.694 0 0 0-1.564-.927l-.07-.02a3.43 3.43 0 0 0-.946-.133L12 7.682zm3.165 3.357c.023 1.748-1.33 3.078-1.33 4.806c.164 2.227 1.733 3.207 3.266 3.146c-.035.003-.068.007-.104.009c-1.847.135-3.209-1.326-5.002-1.326c-2.23.164-3.21 1.736-3.147 3.27l-.008-.104c-.133-1.847 1.328-3.21 1.328-5.002c-.173-2.32-1.867-3.284-3.46-3.132c.1-.011.203-.021.31-.027c1.847-.133 3.209 1.328 5.002 1.328c2.082-.155 3.074-1.536 3.145-2.968zM4.344 0c.238 4.792 3.256 7.056 6.252 7.376c.165-1.692-4.319-5.6-4.319-5.6c-.008-.011.009-.025.019-.014c0 0 4.648 4.01 5.423 5.645c2.762-.15 5.196-1.947 5-4.912c0 0-4.12-.613-5 4.618C11.48 2.753 8.993 0 4.344 0zM12 7.682v.002a3.68 3.68 0 0 0-2.591 1.077L4.94 13.227a3.683 3.683 0 0 0-.86 1.356a3.31 3.31 0 0 0-.237 1.255A3.681 3.681 0 0 0 4.92 18.45l4.464 4.466a3.69 3.69 0 0 0 2.251 1.06l.002.001c.093.01.187.015.28.017l-.1-.008c.06.003.117.009.177.009l-.077-.001L12 24l-.004-.005a3.68 3.68 0 0 0 2.61-1.077l4.469-4.465a3.683 3.683 0 0 0 1.006-1.888l.012-.063a3.682 3.682 0 0 0 .057-.541l.003-.061c0-.017.003-.05.004-.06h-.002a3.683 3.683 0 0 0-1.077-2.607l-4.466-4.468a3.694 3.694 0 0 0-1.564-.927l-.07-.02a3.43 3.43 0 0 0-.946-.133L12 7.682zm3.165 3.357c.023 1.748-1.33 3.078-1.33 4.806c.164 2.227 1.733 3.207 3.266 3.146c-.035.003-.068.007-.104.009c-1.847.135-3.209-1.326-5.002-1.326c-2.23.164-3.21 1.736-3.147 3.27l-.008-.104c-.133-1.847 1.328-3.21 1.328-5.002c-.173-2.32-1.867-3.284-3.46-3.132c.1-.011.203-.021.31-.027c1.847-.133 3.209 1.328 5.002 1.328c2.082-.155 3.074-1.536 3.145-2.968z'/></svg>"

images: {
	pihole: {
		repository: "pihole"
		name: "pihole"
		tag: "v5.8.1"
		pullPolicy: "IfNotPresent"
	}
	authProxy: {
		repository: "giolekva"
		name: "auth-proxy"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	pihole: {
		chart: "charts/pihole"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	ingress: {
		chart: "charts/ingress"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	authProxy: {
		chart: "charts/auth-proxy"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

_piholeServiceName: "pihole-web"
_authProxyServiceName: "auth-proxy"
_httpPortName: "http"
_serviceWebPort: 80

helm: {
	pihole: {
		chart: charts.pihole
		values: {
			fullnameOverride: "pihole"
			persistentVolumeClaim: { // TODO(gio): create volume separately as a dependency
				enabled: true
				size: "5Gi"
			}
			admin: {
				enabled: false
			}
			ingress: {
				enabled: false
			}
			serviceDhcp: {
				enabled: false
			}
			serviceDns: {
				type: "ClusterIP"
			}
			serviceWeb: {
				type: "ClusterIP"
				http: {
					enabled: true
					port: _serviceWebPort
				}
				https: {
					enabled: false
				}
			}
			virtualHost: _domain
			resources: {
				requests: {
					cpu: "250m"
					memory: "100M"
				}
				limits: {
					cpu: "500m"
					memory: "250M"
				}
			}
			image: {
				repository: images.pihole.fullName
				tag: images.pihole.tag
				pullPolicy: images.pihole.pullPolicy
			}
		}
	}
	if input.auth.enabled {
		"auth-proxy": {
			chart: charts.authProxy
			values: {
				image: {
					repository: images.authProxy.fullName
					tag: images.authProxy.tag
					pullPolicy: images.authProxy.pullPolicy
				}
				upstream: "\(_piholeServiceName).\(release.namespace).svc.cluster.local"
				whoAmIAddr: "https://accounts.\(global.domain)/sessions/whoami"
				loginAddr: "https://accounts-ui.\(global.domain)/login"
				membershipAddr: "http://memberships.\(global.id)-core-auth-memberships.svc.cluster.local/api/user"
				groups: input.auth.groups
				portName: _httpPortName
			}
		}
	}
	ingress: {
		chart: charts.ingress
		values: {
			domain: _domain
			ingressClassName: input.network.ingressClass
			certificateIssuer: input.network.certificateIssuer
			service: {
				if input.auth.enabled {
					name: _authProxyServiceName
					port: name: _httpPortName
				}
				if !input.auth.enabled {
					name: _piholeServiceName
					port: number: _serviceWebPort
				}
			}
		}
	}
}
