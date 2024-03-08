import (
	"encoding/base64"
)

input: {
	repoAddr: string
	sshPrivateKey: string
}

name: "app-manager"
namespace: "appmanager"

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
				className: _ingressPrivate
				domain: "apps.\(global.privateDomain)"
				certificateIssuer: ""
			}
			clusterRoleName: "\(global.id)-appmanager"
			image: {
				repository: images.appmanager.fullName
				tag: images.appmanager.tag
				pullPolicy: images.appmanager.pullPolicy
			}
		}
	}
}
