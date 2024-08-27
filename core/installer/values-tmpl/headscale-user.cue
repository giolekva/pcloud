input: {
	username: string
	preAuthKey: {
		enabled: bool | *false
	}
}

name: "headscale-user"
namespace: "app-headscale"

out: {
	charts: {
		headscaleUser: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/headscale-user"
		}
	}

	helm: {
		"headscale-user-\(input.username)": {
			chart: charts.headscaleUser
			values: {
				username: input.username
				headscaleApiAddress: "http://headscale-api.\(global.namespacePrefix)app-headscale.svc.cluster.local"
				preAuthKey: {
					enabled: input.preAuthKey.enabled
					secretName: "\(input.username)-headscale-preauthkey"
				}
			}
		}
	}
}
