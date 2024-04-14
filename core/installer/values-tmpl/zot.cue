import (
	"encoding/json"
)

input: {
	network: #Network
	subdomain: string
}

_domain: "\(input.subdomain).\(input.network.domain)"

name: "zot"
namespace: "app-zot"
readme: "OCI-native container image registry, simplified"
description: "OCI-native container image registry, simplified"
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 24 24'><path fill='currentColor' d='M21.231 2.462L7.18 20.923h14.564V24H2.256v-2.462L16.308 3.076H2.975V0h18.256z'/></svg>"

_ingressWithAuthProxy: _IngressWithAuthProxy & {
	inp: {
		auth: {
			enabled: false
		}
		network: input.network
		subdomain: input.subdomain
		serviceName: "zot"
		port: number: _httpPort // TODO(gio): make optional
	}
}

// TODO(gio): configure busybox
images: _ingressWithAuthProxy.out.images & {
	zot: {
		registry: "ghcr.io"
		repository: "project-zot"
		name: "zot-linux-amd64"
		tag: "v2.0.3"
		pullPolicy: "IfNotPresent"
	}
}

charts: _ingressWithAuthProxy.out.charts & {
	zot: {
		chart: "charts/zot"
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
}

volumes: {
	zot: {
		name: "zot"
		accessMode: "ReadWriteOnce"
		size: "100Gi"
	}
}

_httpPort: 80

helm: _ingressWithAuthProxy.out.helm & {
	zot: {
		chart: charts.zot
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
				http: port: _httpPort
			}
			ingress: enabled: false
			mountConfig: true
			configFiles: {
				"config.json": json.Marshal({
					storage: rootDirectory: "/var/lib/registry"
					http: {
						address: "127.0.0.1"
						port: "5000"
					}
					log: level: "debug"
					extensions: {
						ui: enable: true
						search: enable: true
					}
				})
			}
			persistnce: true
			pvc: {
				create: false
				name: volumes.zot.name
			}
			startupProbe: {}
		}
	}
	volume: {
		chart: charts.volume
		values: volumes.zot
	}
}
