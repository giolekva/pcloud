input: {}

name: "certificate-issuer-private"
namespace: "ingress-private"

images: {}

charts: {
	"certificate-issuer-private": {
		path: "charts/certificate-issuer-private"
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
	}
}

helm: {
	"certificate-issuer-private": {
		chart: charts["certificate-issuer-private"]
		dependsOn: [{
			name: "ingress-nginx"
			namespace: "\(global.namespacePrefix)ingress-private"
		}]
		values: {
			issuer: {
				name: issuerPrivate
				server: "https://acme-v02.api.letsencrypt.org/directory"
				// server: "https://acme-staging-v02.api.letsencrypt.org/directory"
				domain: global.privateDomain
				contactEmail: global.contactEmail
			}
			config: {
				createTXTAddr: "http://dns-api.\(global.id)-dns.svc.cluster.local/create-txt-record"
				deleteTXTAddr: "http://dns-api.\(global.id)-dns.svc.cluster.local/delete-txt-record"
			}
		}
	}
}
