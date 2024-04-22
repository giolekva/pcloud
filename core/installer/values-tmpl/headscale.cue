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
	oauth2Client: {
		chart: "charts/oauth2-client"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	headscale: {
		chart: "charts/headscale"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

_domain: "\(input.subdomain).\(global.domain)"
_oauth2ClientSecretName: "oauth2-client"

helm: {
	"oauth2-client": {
		chart: charts.oauth2Client
		// TODO(gio): remove once hydra maester is installed as part of dodo itself
		dependsOn: [{
			name: "auth"
			namespace: "\(global.namespacePrefix)core-auth"
		}]
		values: {
			name: "oauth2-client"
			secretName: _oauth2ClientSecretName
			grantTypes: ["authorization_code"]
			responseTypes: ["code"]
			scope: "openid profile email"
			redirectUris: ["https://\(_domain)/oidc/callback"]
			hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
		}
	}
	headscale: {
		chart: charts.headscale
		dependsOn: [{
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
			ingressClassName: ingressPublic
			certificateIssuer: issuerPublic
			domain: _domain
			publicBaseDomain: global.domain
			ipAddressPool: "\(global.id)-headscale"
			oauth2: {
				secretName: _oauth2ClientSecretName
				issuer: "https://hydra.\(global.domain)"
			}
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
