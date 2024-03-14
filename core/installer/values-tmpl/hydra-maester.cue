input: {}

name: "hydra-maester"
namespace: "auth"

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
		chart: "charts/hydra-maester"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
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
