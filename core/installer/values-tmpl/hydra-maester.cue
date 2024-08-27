input: {}

name: "hydra-maester"
namespace: "auth"

out: {
	images: {
		hydraMaester: {
			repository: "giolekva"
			name: "ory-hydra-maester"
			tag: "latest"
			pullPolicy: "Always"
		}
	}

	charts: {
		hydraMaester: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/hydra-maester"
		}
	}

	helm: {
		"hydra-maester": {
			chart: charts.hydraMaester
			values: {
				adminService: {
					name: "foo.bar.svc.cluster.local"
					port: 80
					scheme: "http"
				}
				image: {
					repository: images.hydraMaester.fullName
					tag: images.hydraMaester.tag
					pullPolicy: images.hydraMaester.pullPolicy
				}
			}
		}
	}
}
