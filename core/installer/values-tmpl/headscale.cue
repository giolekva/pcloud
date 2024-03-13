input: {
	subdomain: string
	ipSubnet: string
}

name: "headscale"
namespace: "app-headscale"

images: {
	headscale: {
		repository: "headscale"
		name: "headscale"
		tag: "0.22.3"
		pullPolicy: "IfNotPresent"
	}
	api: {
		repository: "giolekva"
		name: "headscale-api"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	headscale: {
		chart: "charts/headscale"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	headscale: {
		chart: charts.headscale
		dependsOnExternal: [{
			name: "auth"
			namespace: "\(global.namespacePrefix)core-auth"
		}]
		values: {
			image: {
				repository: images.headscale.fullName
				tag: images.headscale.tag
				pullPolicy: images.headscale.pullPolicy
			}
			storage: size: "5Gi"
			ingressClassName: _ingressPublic
			certificateIssuer: _issuerPublic
			domain: "\(input.subdomain).\(global.domain)"
			publicBaseDomain: global.domain
			oauth2: {
				hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
				hydraPublic: "https://hydra.\(global.domain)"
				clientId: "headscale"
				secretName: "oauth2-client-headscale"
			}
			ipAddressPool: "\(global.id)-headscale"
			api: {
				port: 8585
				ipSubnet: input.ipSubnet
				image: {
					repository: images.api.fullName
					tag: images.api.tag
					pullPolicy: images.api.pullPolicy
				}
			}
			ui: enabled: false
		}
	}
}
