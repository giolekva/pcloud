input: {
	network: #Network
	subdomain: string
}

_domain: "\(input.subdomain).\(input.network.domain)"

readme: "rpuppy application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"

images: {
	rpuppy: {
		repository: "giolekva"
		name: "rpuppy"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	rpuppy: {
		chart: "charts/rpuppy"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	rpuppy: {
		chart: charts.rpuppy
		values: {
			ingressClassName: input.network.ingressClass
			certificateIssuer: input.network.certificateIssuer
			domain: _domain
			image: {
				repository: images.rpuppy.fullName
				tag: images.rpuppy.tag
				pullPolicy: images.rpuppy.pullPolicy
			}
		}
	}
}
