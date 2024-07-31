input: {
	name: string
	domain: string
}

images: {}

name: "Network"
namespace: "ingress-custom"
readme: "Configure custom public domain"
description: readme
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 48 48'><g fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' stroke-width='4'><path d='M4 34h8v8H4zM8 6h32v12H8zm16 28V18'/><path d='M8 34v-8h32v8m-4 0h8v8h-8zm-16 0h8v8h-8zm-6-22h2'/></g></svg>"

charts: {
	"certificate-issuer": {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/certificate-issuer-public"
	}
}

helm: {
	"certificate-issuer": {
		chart: charts["certificate-issuer"]
		dependsOn: [{
			name: "ingress-nginx"
			namespace: "\(global.namespacePrefix)ingress-private"
		}]
		values: {
			issuer: {
				name: input.name
				server: "https://acme-v02.api.letsencrypt.org/directory"
				domain: input.domain
				contactEmail: global.contactEmail
				ingressClass: networks.public.ingressClass
			}
		}
	}
}
