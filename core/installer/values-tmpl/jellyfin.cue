input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"
url: "https://\(_domain)"

name: "Jellyfin"
namespace: "app-jellyfin"
description: "Jellyfin is a free and open-source media server and suite of multimedia applications designed to organize, manage, and share digital media files to networked devices."
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 48 48'><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M24 20c-1.62 0-6.85 9.48-6.06 11.08s11.33 1.59 12.12 0S25.63 20 24 20Z'/><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M24 5.5c-4.89 0-20.66 28.58-18.25 33.4s34.13 4.77 36.51 0S28.9 5.5 24 5.5Zm12 29.21c-1.56 3.13-22.35 3.17-23.93 0S20.8 12.83 24 12.83s13.52 18.76 12 21.88Z'/></svg>"

readme: "jellyfin application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"

images: {
	jellyfin: {
		repository: "jellyfin"
		name: "jellyfin"
		tag: "10.8.10"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	jellyfin: {
		kind: "GitRepository"
		address: "https://code.v1.dodo.cloud/helm-charts"
		branch: "main"
		path: "charts/jellyfin"
	}
}

helm: {
	jellyfin: {
		chart: charts.jellyfin
		values: {
			pcloudInstanceId: global.id
			ingress: {
				className: input.network.ingressClass
				domain: _domain
			}
			image: {
				repository: images.jellyfin.fullName
				tag: images.jellyfin.tag
				pullPolicy: images.jellyfin.pullPolicy
			}
		}
	}
}
