#Input: {
	network: #Network
	subdomain: string
}

input: #Input

_domain: "\(input.subdomain).\(input.network.domain)"

readme: "Installs vaultwarden on private network accessible at \(_domain)"

images: {
	vaultwarden: {
		repository: "vaultwarden"
		name: "server"
		tag: "1.28.1"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	vaultwarden: {
		chart: "charts/vaultwarden"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	vaultwarden: {
		chart: charts.vaultwarden
		values: {
			ingressClassName: input.network.ingressClass
			certificateIssuer: input.network.certificateIssuer
			domain: _domain
			image: {
				repository: images.vaultwarden.fullName
				tag: images.vaultwarden.tag
				pullPolicy: images.vaultwarden.pullPolicy
			}
		}
	}
}

// TODO(gio): import

#Network: {
	name: string
	ingressClass: string
	certificateIssuer: string | *""
	domain: string
}

#Image: {
	registry: string | *"docker.io"
	repository: string
	name: string
	tag: string
	pullPolicy: string // TODO(gio): remove?
	fullName: "\(registry)/\(repository)/\(name)"
	fullNameWithTag: "\(fullName):\(tag)"
}

#Chart: {
	chart: string
	sourceRef: #SourceRef
}

#SourceRef: {
	kind: "GitRepository" | "HelmRepository"
	name: string
	namespace: string // TODO(gio): default global.id
}

#Global: {
	id: string
	pcloudEnvName: string
	domain: string
	namespacePrefix: string
	...
}

#Release: {
	namespace: string
}

global: #Global
release: #Release

images: {
	for key, value in images {
		"\(key)": #Image & value
	}
}

charts: {
	for key, value in charts {
		"\(key)": #Chart & value
	}
}

#Helm: {
	name: string
	dependsOn: [...#Helm] | *[]
	...
}

helm: {
	for key, value in helm {
		"\(key)": #Helm & value & {
			name: key
		}
	}
}

#HelmRelease: {
	_name: string
	_chart: #Chart
	_values: _
	_dependencies: [...#Helm] | *[]

	apiVersion: "helm.toolkit.fluxcd.io/v2beta1"
	kind: "HelmRelease"
	metadata: {
		name: _name
   		namespace: release.namespace
	}
	spec: {
		interval: "1m0s"
		dependsOn: [
			for d in _dependencies {
				name: d.name
				namespace: release.namespace
			}
    	]
		chart: {
			spec: _chart
		}
		values: _values
	}
}

output: {
	for name, r in helm {
		"\(name)": #HelmRelease & {
			_name: name
			_chart: r.chart
			_values: r.values
			_dependencies: r.dependsOn
		}
	}
}
