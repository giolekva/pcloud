import (
	"encoding/base64"
)

input: {
	sshPrivateKey: string
}

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
	portAllocator: {
		repository: "giolekva"
		name: "port-allocator"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	ingressNginx: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/ingress-nginx"
	}
	portAllocator: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/port-allocator"
	}
}

helm: {
	"ingress-public": {
		chart: charts.ingressNginx
		values: {
			fullnameOverride: ingressPublic
			controller: {
				kind: "DaemonSet"
				hostNetwork: true
				hostPort: enabled: true
				service: enabled: false
				ingressClassByName: true
				ingressClassResource: {
					name: ingressPublic
					enabled: true
					default: false
					controllerValue: "k8s.io/\(ingressPublic)"
				}
				config: {
					"proxy-body-size": "200M" // TODO(giolekva): configurable
					"server-snippet": """
					more_clear_headers "X-Frame-Options";
					"""
				}
				image: {
					registry: images.ingressNginx.registry
					image: images.ingressNginx.imageName
					tag: images.ingressNginx.tag
					pullPolicy: images.ingressNginx.pullPolicy
				}
			}
			tcp: {
				"53": "\(global.pcloudEnvName)-dns-gateway/coredns:53"
			}
			udp: {
				"53": "\(global.pcloudEnvName)-dns-gateway/coredns:53"
			}
		}
	}
	"port-allocator": {
		chart: charts.portAllocator
		values: {
			repoAddr: release.repoAddr
			sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
			ingressNginxPath: "\(release.appDir)/ingress-public.yaml"
			image: {
				repository: images.portAllocator.fullName
				tag: images.portAllocator.tag
				pullPolicy: images.portAllocator.pullPolicy
			}
		}
	}
}
