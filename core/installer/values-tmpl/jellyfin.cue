input: {
	network: #Network
	subdomain: string
}

_domain: "\(input.subdomain).\(input.network.domain)"

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
		chart: "charts/jellyfin"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
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
