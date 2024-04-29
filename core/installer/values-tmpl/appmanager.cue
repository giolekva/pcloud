import (
	"encoding/base64"
)

input: {
	repoAddr: string
	sshPrivateKey: string
	authGroups: string
}

name: "App Manager"
namespace: "appmanager"

_subdomain: "apps"
_httpPortName: "http"

ingress: {
	appmanager: {
		auth: {
			enabled: true
			groups: input.authGroups
		}
		network: networks.private
		subdomain: _subdomain
		service: {
			name: "appmanager"
			port: name: _httpPortName
		}
	}
}

images: {
	appmanager: {
		repository: "giolekva"
		name: "pcloud-installer"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	appmanager: {
		chart: "charts/appmanager"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
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
