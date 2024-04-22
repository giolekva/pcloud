import (
	"encoding/base64"
)

input: {
	repoAddr: string
	sshPrivateKey: string
}

name: "welcome"
namespace: "app-welcome"

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
			loginAddr: "https://launcher.\(global.domain)"
			membershipsInitAddr: "http://memberships-api.\(global.namespacePrefix)core-auth-memberships.svc.cluster.local/api/init"
			ingress: {
				className: ingressPublic
				domain: "welcome.\(global.domain)"
				certificateIssuer: issuerPublic
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
