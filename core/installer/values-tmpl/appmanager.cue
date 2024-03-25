import (
	"encoding/base64"
)

input: {
	repoAddr: string
	sshPrivateKey: string
	authGroups: string
}

name: "app-manager"
namespace: "appmanager"

_subdomain: "apps"
_httpPortName: "http"

_ingressWithAuthProxy: _IngressWithAuthProxy & {
	inp: {
		auth: {
			enabled: true
			groups: input.authGroups
		}
		network: networks.private
		subdomain: _subdomain
		serviceName: "appmanager"
		port: name: _httpPortName
	}
}

images: _ingressWithAuthProxy.out.images & {
	appmanager: {
		repository: "giolekva"
		name: "pcloud-installer"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: _ingressWithAuthProxy.out.charts & {
	appmanager: {
		chart: "charts/appmanager"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: _ingressWithAuthProxy.out.helm & {
	appmanager: {
		chart: charts.appmanager
		values: {
			repoAddr: input.repoAddr
			sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
			ingress: {
				className: networks.private.ingressClass
				domain: "\(_subdomain).\(networks.private.domain)"
				certificateIssuer: ""
			}
			clusterRoleName: "\(global.id)-appmanager"
			portName: _httpPortName
			image: {
				repository: images.appmanager.fullName
				tag: images.appmanager.tag
				pullPolicy: images.appmanager.pullPolicy
			}
		}
	}
}
