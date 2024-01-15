input: {
	subdomain: string
	adminKey: string
}

_domain: "\(input.subdomain).\(global.privateDomain)"

readme: "softserve application will be installed on private network and be accessible to any user on https://\(_domain)" // TODO(gio): make public network an option

images: {
	softserve: {
		repository: "charmcli"
		name: "soft-serve"
		tag: "v0.7.1"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	softserve: {
		chart: "charts/soft-serve"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	softserve: {
		chart: charts.softserve
		values: {
			serviceType: "LoadBalancer"
			reservedIP: ""
			addressPool: global.id
			adminKey: input.adminKey
			privateKey: ""
			publicKey: ""
			ingress: {
				enabled: false
			}
			image: {
				repository: images.softserve.fullName
				tag: images.softserve.tag
				pullPolicy: images.softserve.pullPolicy
			}
		}
	}
}
