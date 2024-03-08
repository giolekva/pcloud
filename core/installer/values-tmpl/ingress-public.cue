input: {}

name: "ingress-public"
namespace: "ingress-public"

images: {
	ingressNginx: {
		registry: "registry.k8s.io"
		repository: "ingress-nginx"
		name: "controller"
		tag: "v1.8.0"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	ingressNginx: {
		chart: "charts/ingress-nginx"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
}

helm: {
	"ingress-public": {
		chart: charts.ingressNginx
		values: {
			fullnameOverride: _ingressPublic
			controller: {
				kind: "DaemonSet"
				hostNetwork: true
				hostPort: enabled: true
				service: enabled: false
				ingressClassByName: true
				ingressClassResource: {
					name: _ingressPublic
					enabled: true
					default: false
					controllerValue: "k8s.io/\(_ingressPublic)"
				}
				config: "proxy-body-size": "100M" // TODO(giolekva): configurable
				image: {
					registry: images.ingressNginx.registry
					image: images.ingressNginx.imageName
					tag: images.ingressNginx.tag
					pullPolicy: images.ingressNginx.pullPolicy
				}
			}
			tcp: {
				"53": "\(global.pcloudEnvName)-dns-zone-manager/coredns:53"
			}
			udp: {
				"53": "\(global.pcloudEnvName)-dns-zone-manager/coredns:53"
			}
		}
	}
}
