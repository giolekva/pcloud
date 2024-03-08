input: {}

name: "fluxcd-reconciler"
namespace: "fluxcd-reconciler"

images: {
	fluxcdReconciler: {
		repository: "giolekva"
		name: "fluxcd-reconciler"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	fluxcdReconciler: {
		chart: "charts/fluxcd-reconciler"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
}

helm: {
	"fluxcd-reconciler": {
		chart: charts.fluxcdReconciler
		values: {
			image: {
				repository: images.fluxcdReconciler.fullName
				tag: images.fluxcdReconciler.tag
				pullPolicy: images.fluxcdReconciler.pullPolicy
			}
		}
	}
}
