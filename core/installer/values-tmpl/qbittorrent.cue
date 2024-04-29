input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"

name: "qBitorrent"
namespace: "app-qbittorrent"
readme: "qbittorrent application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"
description: "qBittorrent is a cross-platform free and open-source BitTorrent client written in native C++. It relies on Boost, Qt 6 toolkit and the libtorrent-rasterbar library, with an optional search engine written in Python."
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 48 48'><circle cx='24' cy='24' r='21.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M26.651 22.364a5.034 5.034 0 0 1 5.035-5.035h0a5.034 5.034 0 0 1 5.034 5.035v3.272a5.034 5.034 0 0 1-5.034 5.035h0a5.034 5.034 0 0 1-5.035-5.035m0 5.035V10.533m-5.302 15.103a5.034 5.034 0 0 1-5.035 5.035h0a5.034 5.034 0 0 1-5.034-5.035v-3.272a5.034 5.034 0 0 1 5.034-5.035h0a5.034 5.034 0 0 1 5.035 5.035m0-5.035v20.138'/></svg>"

images: {
	qbittorrent: {
		registry: "lscr.io"
		repository: "linuxserver"
		name: "qbittorrent"
		tag: "4.5.3"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	qbittorrent: {
		chart: "charts/qbittorrent"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	qbittorrent: {
		chart: charts.qbittorrent
		values: {
			pcloudInstanceId: global.id
			ingress: {
				className: input.network.ingressClass
				domain: _domain
			}
			webui: port: 8080
			bittorrent: port: 6881
			storage: size: "100Gi"
			image: {
				repository: images.qbittorrent.fullName
				tag: images.qbittorrent.tag
				pullPolicy: images.qbittorrent.pullPolicy
			}
		}
	}
}
