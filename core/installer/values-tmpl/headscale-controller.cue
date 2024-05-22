input: {}

name: "headscale-controller"
namespace: "core-headscale"

images: {
	headscaleController: {
		repository: "giolekva"
		name: "headscale-controller"
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
	headscaleController: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/headscale-controller"
	}
}

helm: {
	"headscale-controller": {
		chart: charts.headscaleController
		values: {
			installCRDs: true
			image: {
				repository: images.headscaleController.fullName
				tag: images.headscaleController.tag
				pullPolicy: images.headscaleController.pullPolicy
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
