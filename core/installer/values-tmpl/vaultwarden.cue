input: {
    network: #Network @name(Network)
    subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"
url: "https://\(_domain)"

name: "Vaultwarden"
namespace: "app-vaultwarden"
readme: "Installs vaultwarden on private network accessible at \(_domain)"
description: "Alternative implementation of the Bitwarden server API written in Rust and compatible with upstream Bitwarden clients, perfect for self-hosted deployment where running the official resource-heavy service might not be ideal."
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 48 48'><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M35.38 25.63V9.37H24v28.87a34.93 34.93 0 0 0 5.41-3.48q6-4.66 6-9.14Zm4.87-19.5v19.5A11.58 11.58 0 0 1 39.4 30a16.22 16.22 0 0 1-2.11 3.81a23.52 23.52 0 0 1-3 3.24a34.87 34.87 0 0 1-3.22 2.62c-1 .69-2 1.35-3.07 2s-1.82 1-2.27 1.26l-1.08.51a1.53 1.53 0 0 1-1.32 0l-1.08-.51c-.45-.22-1.21-.64-2.27-1.26s-2.09-1.27-3.07-2A34.87 34.87 0 0 1 13.7 37a23.52 23.52 0 0 1-3-3.24A16.22 16.22 0 0 1 8.6 30a11.58 11.58 0 0 1-.85-4.32V6.13A1.64 1.64 0 0 1 9.38 4.5h29.24a1.64 1.64 0 0 1 1.63 1.63Z'/></svg>"

images: {
	vaultwarden: {
		repository: "vaultwarden"
		name: "server"
		tag: "1.28.1"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	vaultwarden: {
		chart: "charts/vaultwarden"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	vaultwarden: {
		chart: charts.vaultwarden
		values: {
			ingressClassName: input.network.ingressClass
			certificateIssuer: input.network.certificateIssuer
			domain: _domain
			image: {
				repository: images.vaultwarden.fullName
				tag: images.vaultwarden.tag
				pullPolicy: images.vaultwarden.pullPolicy
			}
		}
	}
}

help: [{
	title: "Access"
	contents: "Open [\(url)](\(url)) in a new tab."
}]
