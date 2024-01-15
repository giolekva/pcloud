input: {}

images: {
	certManager: {
		registry: "quay.io"
		repository: "jetstack"
		name: "cert-manager-controller"
		tag: "v1.12.2"
		pullPolicy: "IfNotPresent"
	}
	cainjector: {
		registry: "quay.io"
		repository: "jetstack"
		name: "cert-manager-cainjector"
		tag: "v1.12.2"
		pullPolicy: "IfNotPresent"
	}
	webhook: {
		registry: "quay.io"
		repository: "jetstack"
		name: "cert-manager-webhook"
		tag: "v1.12.2"
		pullPolicy: "IfNotPresent"
	}
	dnsChallengeSolver: {
		repository: "giolekva"
		name: "dns-challenge-solver"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	certManager: {
		chart: "charts/cert-manager"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
	dnsChallengeSolver: {
		chart: "charts/cert-manager-webhook-pcloud"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName
		}
	}
}

helm: {
	"cert-manager": {
		chart: charts.certManager
		dependsOnExternal: [{
			name: "ingress-public"
			namespace: _ingressPublic
		}]
		values: {
			fullnameOverride: "\(global.pcloudEnvName)-cert-manager"
			installCRDs: true
			dns01RecursiveNameserversOnly: true
			dns01RecursiveNameservers: "1.1.1.1:53,8.8.8.8:53"
			image: {
				repository: images.certManager.fullName
				tag: images.certManager.tag
				pullPolicy: images.certManager.pullPolicy
			}
			cainjector: {
				image: {
					repository: images.cainjector.fullName
					tag: images.cainjector.tag
					pullPolicy: images.cainjector.pullPolicy
				}
			}
			webhook: {
				image: {
					repository: images.webhook.fullName
					tag: images.webhook.tag
					pullPolicy: images.webhook.pullPolicy
				}
			}
		}
	}
	"cert-manager-webhook-pcloud": {
		chart: charts.dnsChallengeSolver
		dependsOn: [helm["cert-manager"]]
		values: {
			fullnameOverride: "\(global.pcloudEnvName)-cert-manager-webhook-pcloud"
			certManager: {
				name: "\(global.pcloudEnvName)-cert-manager"
				namespace: "\(global.pcloudEnvName)-cert-manager"
			}
			image: {
				repository: images.dnsChallengeSolver.fullName
				tag: images.dnsChallengeSolver.tag
				pullPolicy: images.dnsChallengeSolver.pullPolicy
			}
			logLevel: 2
			apiGroupName: "dodo.cloud"
			resolverName: "dns-resolver-pcloud"
		}
	}
}
