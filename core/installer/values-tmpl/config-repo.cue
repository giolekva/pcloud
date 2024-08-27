input: {
	privateKey: string
	publicKey: string
	adminKey: string
}

name: "config-repo"
namespace: "config-repo"

out: {
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
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/soft-serve"
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
				image: {
					repository: images.softserve.fullName
					tag: images.softserve.tag
					pullPolicy: images.softserve.pullPolicy
				}
			}
		}
	}
}
