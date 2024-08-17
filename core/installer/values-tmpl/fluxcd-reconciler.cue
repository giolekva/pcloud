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
		kind: "GitRepository"
		address: "https://code.v1.dodo.cloud/helm-charts"
		branch: "main"
		path: "charts/fluxcd-reconciler"
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
