input: {
	apiConfigMapName: string
	volume: {
		size: string
		claimName: string
		mountPath: string
	}
}

images: {
	dnsZoneController: {
		repository: "giolekva"
		name: "dns-ns-controller"
		tag: "latest"
		pullPolicy: "Always"
	}
	kubeRBACProxy: {
		registry: "gcr.io"
		repository: "kubebuilder"
		name: "kube-rbac-proxy"
		tag: "v0.13.0"
		pullPolicy: "IfNotPresent"
	}
	coredns: {
		repository: "coredns"
		name: "coredns"
		tag: "1.11.1"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	volume: {
		chart: "charts/volumes"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
	dnsZoneController: {
		chart: "charts/dns-ns-controller"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
	coredns: {
		chart: "charts/coredns"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
}

_volumeName: "zone-configs"

helm: {
	volume: {
		chart: charts.volume
		values: {
			name: input.volume.claimName
			size: input.volume.size
			accessMode: "ReadWriteMany"
		}
	}
	"dns-zone-controller": {
		chart: charts.dnsZoneController
		values: {
			installCRDs: true
			apiConfigMapName: input.apiConfigMapName
			volume: {
				claimName: input.volume.claimName
				mountPath: input.volume.mountPath
			}
			image: {
				repository: images.dnsZoneController.fullName
				tag: images.dnsZoneController.tag
				pullPolicy: images.dnsZoneController.pullPolicy
			}
			kubeRBACProxy: {
				image: {
					repository: images.kubeRBACProxy.fullName
					tag: images.kubeRBACProxy.tag
					pullPolicy: images.kubeRBACProxy.pullPolicy
				}
			}
		}
	}
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
			serviceType: "ClusterIP"
			service: name: "coredns"
			serviceAccount: create: false
			rbac: {
				create: true
				pspEnable: false
			}
			isClusterService: true
			securityContext: capabilities: add: ["NET_BIND_SERVICE"]
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
			extraConfig: import: parameters: "\(input.volume.mountPath)/coredns.conf"
			extraVolumes: [{
				name: _volumeName
				persistentVolumeClaim: claimName: input.volume.claimName
			}]
			extraVolumeMounts: [{
				name: _volumeName
				mountPath: input.volume.mountPath
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
}
