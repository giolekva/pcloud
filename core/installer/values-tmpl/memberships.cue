input: {
    network: #Network
    subdomain: string
}

_domain: "\(input.subdomain).\(input.network.domain)"

name: "memberships"
namespace: "app-memberships"
readme: "Memberships application will be installed on \(input.network.name) network and be accessible at https://\(_domain)"
description: "The application is a membership management system designed to facilitate the organization and administration of groups and memberships. Can be configured to be reachable only from private network or publicly."
icon: "<svg xmlns='http://www.w3.org/2000/svg" width="40.63" height="50" viewBox="0 0 13 16"><circle cx="2" cy="10" r="1" fill="currentColor"/><circle cx="2" cy="6" r="1" fill="currentColor"/><path fill="currentColor" d="M4.5 14c-.06 0-.11 0-.17-.03a.501.501 0 0 1-.3-.64l4-11a.501.501 0 0 1 .94.34l-4 11c-.07.2-.27.33-.47.33m3 0c-.06 0-.11 0-.17-.03a.501.501 0 0 1-.3-.64l4-11a.501.501 0 0 1 .94.34l-4 11c-.07.2-.27.33-.47.33'/></svg>"

images: {
	memberships: {
		repository: "giolekva"
		name: "memberships"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
    memberships: {
        chart: "charts/memberships"
        sourceRef: {
            kind: "GitRepository"
            name: "pcloud"
            namespace: global.id
        }
    }
}

helm: {
    "memberships": {
        chart: charts.memberships
        values: {
            ingressClassName: input.network.ingressClass
            certificateIssuer: input.network.certificateIssuer
            domain: _domain
            storage: {
                size: "1Gi"
            }
            image: {
				repository: images.memberships.name
				tag: images.memberships.tag
				pullPolicy: images.memberships.pullPolicy
			}
            port: 8080
        }
    }
}
