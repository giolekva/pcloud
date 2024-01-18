input: {
	username: string
	preAuthKey: {
		enabled: bool | *false
	}
}

namespace: "app-headscale"

images: {}

charts: {
	headscaleUser: {
		chart: "charts/headscale-user"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	"headscale-user": {
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
