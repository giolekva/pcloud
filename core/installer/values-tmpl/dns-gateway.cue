input: {
	servers: [...#Server]
}

#Server: {
	zone: string
	address: string
}

name: "dns-gateway"
namespace: "dns-gateway"

out: {
	images: {
		coredns: {
			repository: "coredns"
			name: "coredns"
			tag: "1.11.1"
			pullPolicy: "IfNotPresent"
		}
	}

	charts: {
		coredns: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/coredns"
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
				serviceType: "ClusterIP"
				service: name: "coredns"
				serviceAccount: create: false
				rbac: {
					create: false
					pspEnable: false
				}
				isClusterService: false
				if len(input.servers) > 0 {
					servers: [
						for s in input.servers {
							zones: [{
								zone: s.zone
							}]
							port: 53
							plugins: [{
								name: "log"
							}, {
								name: "forward"
								parameters: ". \(s.address)"
							}, {
								name: "health"
								configBlock: "lameduck 5s"
							}, {
								name: "ready"
							}]
						}
					]
				}
				if len(input.servers) == 0 {
					servers: [{
						zones: [{
							zone: "."
						}]
						port: 53
						plugins: [{
							name: "ready"
						}]
					}]
				}
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
}
