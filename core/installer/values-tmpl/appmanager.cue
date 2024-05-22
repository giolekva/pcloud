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
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 24 24'><path fill='currentColor' d='M19 6h-2c0-2.8-2.2-5-5-5S7 3.2 7 6H5c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2m-7-3c1.7 0 3 1.3 3 3H9c0-1.7 1.3-3 3-3m7 17H5V8h14zm-7-8c-1.7 0-3-1.3-3-3H7c0 2.8 2.2 5 5 5s5-2.2 5-5h-2c0 1.7-1.3 3-3 3'/></svg>"

_subdomain: "apps"
_httpPortName: "http"

_domain: "\(_subdomain).\(networks.private.domain)"
url: "https://\(_domain)"

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
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/appmanager"
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
				domain: _domain
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
