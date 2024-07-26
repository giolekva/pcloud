import (
	"encoding/base64"
)

input: {
	network: #Network
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
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/welcome"
	}
}

helm: {
	welcome: {
		chart: charts.welcome
		values: {
			repoAddr: input.repoAddr
			sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
			createAccountAddr: "http://api.\(global.namespacePrefix)core-auth.svc.cluster.local/identities"
			loginAddr: "https://launcher.\(networks.public.domain)"
			membershipsInitAddr: "http://memberships-api.\(global.namespacePrefix)core-auth-memberships.svc.cluster.local/api/init"
			ingress: {
				className: input.network.ingressClass
				domain: "welcome.\(input.network.domain)"
				certificateIssuer: input.network.certificateIssuer
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
