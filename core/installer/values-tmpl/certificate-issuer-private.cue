input: {
	apiConfigMap: {
		name: string
		namespace: string
	}
}

namespace: "ingress-private"

images: {}

charts: {
	"certificate-issuer-private": {
		chart: "charts/certificate-issuer-private"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	"certificate-issuer-private": {
		chart: charts["certificate-issuer-private"]
		dependsOnExternal: [{
			name: "ingress-nginx"
			namespace: "\(global.namespacePrefix)ingress-private"
		}]
		values: {
			issuer: {
				name: _issuerPrivate
				server: "https://acme-v02.api.letsencrypt.org/directory"
				// server: "https://acme-staging-v02.api.letsencrypt.org/directory"
				domain: global.privateDomain
				contactEmail: global.contactEmail
			}
			apiConfigMap: {
				name: input.apiConfigMap.name
				namespace: input.apiConfigMap.namespace
			}
		}
	}
}
