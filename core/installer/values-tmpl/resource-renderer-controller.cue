input: {}

name: "resource-renderer-controller"
namespace: "rr-controller"

images: {
	resourceRenderer: {
		repository: "giolekva"
		name: "resource-renderer-controller"
		tag: "latest"
		pullPolicy: "Always"
	}
	kubeRBACProxy: {
		registry: "gcr.io"
		repository: "kubebuilder"
		name: "kube-rbac-proxy"
		tag: "v0.13.0"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	resourceRenderer: {
		chart: "charts/resource-renderer-controller"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
}

helm: {
	"resource-renderer": {
		chart: charts.resourceRenderer
		values: {
			installCRDs: true
			image: {
				repository: images.resourceRenderer.fullName
				tag: images.resourceRenderer.tag
				pullPolicy: images.resourceRenderer.pullPolicy
			}
			kubeRBACProxy: {
				image: {
					repository: images.kubeRBACProxy.fullName
					tag: images.kubeRBACProxy.tag
					pullPolicy: images.kubeRBACProxy.pullPolicy
				}
			}
		}
	}
}
