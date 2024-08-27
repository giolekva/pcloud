input: {
	network: #Network
}

name: "certificate-issuer-public"
namespace: "ingress-private"

out: {
	charts: {
		"certificate-issuer-public": {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/certificate-issuer-public"
		}
	}

	helm: {
		"certificate-issuer-public": {
			chart: charts["certificate-issuer-public"]
			values: {
				issuer: {
					name: input.network.certificateIssuer
					server: "https://acme-v02.api.letsencrypt.org/directory"
					domain: input.network.domain
					contactEmail: global.contactEmail
					ingressClass: input.network.ingressClass
				}
			}
		}
	}
}
