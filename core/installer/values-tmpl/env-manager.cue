import (
	"encoding/base64"
)

input: {
	repoIP: string
	repoPort: number
	repoName: string
	sshPrivateKey: string
}

name: "env-manager"
namespace: "env-manager"

out: {
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
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/env-manager"
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
}
