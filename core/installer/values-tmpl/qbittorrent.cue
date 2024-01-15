input: {
	network: #Network
	subdomain: string
}

_domain: "\(input.subdomain).\(input.network.domain)"

readme: "qbittorrent application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"

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
