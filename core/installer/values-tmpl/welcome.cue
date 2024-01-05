import (
	"encoding/base64"
)

input: {
	repoAddr: string
	sshPrivateKey: string
	createAccountAddr: string
}

images: {
	welcome: {
		repository: "giolekva"
		name: "pcloud-installer"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	welcome: {
		chart: "charts/welcome"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	welcome: {
		chart: charts.welcome
		values: {
			repoAddr: input.repoAddr
			sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
			createAccountAddr: "http://api.\(global.namespacePrefix)core-auth.svc.cluster.local/identities"
			ingress: {
				className: _ingressPublic
				domain: "welcome.\(global.domain)"
				certificateIssuer: _issuerPublic
			}
			clusterRoleName: "\(global.id)-welcome"
			image: {
				repository: images.welcome.fullName
				tag: images.welcome.tag
				pullPolicy: images.welcome.pullPolicy
			}
		}
	}
}
