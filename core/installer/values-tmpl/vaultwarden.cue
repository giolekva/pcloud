input: {
	network: #Network
	subdomain: string
}

_domain: "\(input.subdomain).\(input.network.domain)"

readme: "Installs vaultwarden on private network accessible at \(_domain)"

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
