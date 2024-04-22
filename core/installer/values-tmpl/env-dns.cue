import (
	"strings"
)

input: {}

name: "env-dns"
namespace: "dns"
readme: "env-dns"
description: "Environment local DNS manager"
icon: ""

images: {
	coredns: {
		repository: "coredns"
		name: "coredns"
		tag: "1.11.1"
		pullPolicy: "IfNotPresent"
	}
	api: {
		repository: "giolekva"
		name: "dns-api"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	coredns: {
		chart: "charts/coredns"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	api: {
		chart: "charts/dns-api"
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
	service: {
		chart: "charts/service"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	ipAddressPool: {
		chart: "charts/metallb-ipaddresspool"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

volumes: {
	data: {
		name: "data"
		accessMode: "ReadWriteMany"
		size: "5Gi"
	}
}

helm: {
	coredns: {
		chart: charts.coredns
		values: {
			image: {
				repository: images.coredns.fullName
				tag: images.coredns.tag
				pullPolicy: images.coredns.pullPolicy
			}
			replicaCount: 1
			resources: {
				limits: {
					cpu: "100m"
					memory: "128Mi"
				}
				requests: {
					cpu: "100m"
					memory: "128Mi"
				}
			}
			rollingUpdate: {
				maxUnavailable: 1
				maxSurge: "25%"
			}
			terminationGracePeriodSeconds: 30
			serviceType: "LoadBalancer"
			service: {
				name: "coredns"
				annotations: {
					"metallb.universe.tf/loadBalancerIPs": global.network.dns
				}
			}
			serviceAccount: create: false
			rbac: {
				create: false
				pspEnable: false
			}
			isClusterService: false
			servers: [{
				zones: [{
					zone: "."
				}]
				port: 53
				plugins: [
					{
						name: "log"
					},
					{
						name: "health"
						configBlock: "lameduck 5s"
					},
					{
						name: "ready"
					}
			    ]
			}]
			extraConfig: import: parameters: "\(_mountPath)/coredns.conf"
			extraVolumes: [{
				name: volumes.data.name
				persistentVolumeClaim: claimName: volumes.data.name
			}]
			extraVolumeMounts: [{
				name: volumes.data.name
				mountPath: _mountPath
			}]
			livenessProbe: {
				enabled: true
				initialDelaySeconds: 60
				periodSeconds: 10
				timeoutSeconds: 5
				failureThreshold: 5
				successThreshold: 1
			}
			readinessProbe: {
				enabled: true
				initialDelaySeconds: 30
				periodSeconds: 10
				timeoutSeconds: 5
				failureThreshold: 5
				successThreshold: 1
			}
			zoneFiles: []
			hpa: enabled: false
			autoscaler: enabled: false
			deployment: enabled: true
		}
	}
	api: {
		chart: charts.api
		values: {
			image: {
				repository: images.api.fullName
				tag: images.api.tag
				pullPolicy: images.api.pullPolicy
			}
			config: "coredns.conf"
			db: "records.db"
			zone: global.domain
			publicIP: strings.Join(global.publicIP, ",")
			privateIP: global.network.ingress
			nameserverIP: strings.Join(global.nameserverIP, ",")
			service: type: "ClusterIP"
			volume: {
				claimName: volumes.data.name
				mountPath: _mountPath
			}
		}
	}
	"data-volume": {
		chart: charts.volume
		values: volumes.data
	}
	"coredns-svc-cluster": {
		chart: charts.service
		values: {
			name: "dns"
			type: "LoadBalancer"
			protocol: "TCP"
			ports: [{
				name: "udp-53"
				port: 53
				protocol: "UDP"
				targetPort: 53
			}]
			targetPort: "http"
			selector:{
				"app.kubernetes.io/instance": "coredns"
				"app.kubernetes.io/name": "coredns"
			}
			annotations: {
				"metallb.universe.tf/loadBalancerIPs": global.network.dnsInClusterIP
			}
		}
	}
	"ipaddresspool-dns": {
		chart: charts.ipAddressPool
		values: {
			name: "\(global.id)-dns"
			autoAssign: false
			from: global.network.dns
			to: global.network.dns
			namespace: "metallb-system"
		}
	}
	"ipaddresspool-dns-in-cluster": {
		chart: charts.ipAddressPool
		values: {
			name: "\(global.id)-dns-in-cluster"
			autoAssign: false
			from: global.network.dnsInClusterIP
			to: global.network.dnsInClusterIP
			namespace: "metallb-system"
		}
	}
}

_mountPath: "/pcloud"
