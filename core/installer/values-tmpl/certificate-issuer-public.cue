input: {}

images: {}

name: "certificate-issuer-public"
namespace: "ingress-private"

charts: {
	"certificate-issuer-public": {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/certificate-issuer-public"
	}
}

helm: {
	"certificate-issuer-public": {
		chart: charts["certificate-issuer-public"]
		dependsOn: [{
			name: "ingress-nginx"
			namespace: "\(global.namespacePrefix)ingress-private"
		}]
		values: {
			issuer: {
				name: issuerPublic
				server: "https://acme-v02.api.letsencrypt.org/directory"
				// server: "https://acme-staging-v02.api.letsencrypt.org/directory"
				domain: global.domain
				contactEmail: global.contactEmail
				ingressClass: ingressPublic
			}
		}
	}
}
