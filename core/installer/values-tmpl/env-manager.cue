import (
	"encoding/base64"
)

name: "env-manager"
namespace: "env-manager"

input: {
	repoIP: string
	repoPort: number
	repoName: string
	sshPrivateKey: string
}

images: {
	envManager: {
		repository: "giolekva"
		name: "pcloud-installer"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	envManager: {
		chart: "charts/env-manager"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
}

helm: {
	"env-manager": {
		chart: charts.envManager
		values: {
			repoIP: input.repoIP
			repoPort: input.repoPort
			repoName: input.repoName
			sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
			clusterRoleName: "\(global.pcloudEnvName)-env-manager"
			image: {
				repository: images.envManager.fullName
				tag: images.envManager.tag
				pullPolicy: images.envManager.pullPolicy
			}
		}
	}
}
