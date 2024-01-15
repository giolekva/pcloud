input: {
	privateKey: string
	publicKey: string
	adminKey: string
}

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
			namespace: global.pcloudEnvName
		}
	}
}

helm: {
	softserve: {
		chart: charts.softserve
		values: {
			serviceType: "ClusterIP"
			addressPool: ""
			reservedIP: ""
			adminKey: input.adminKey
			privateKey: input.privateKey
			publicKey: input.publicKey
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
