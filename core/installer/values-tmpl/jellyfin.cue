#Input: {
	network: #Network
	subdomain: string
}

input: #Input

_domain: "\(input.subdomain).\(input.network.domain)"

readme: "jellyfin application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"

images: {
	jellyfin: {
		repository: "jellyfin"
		name: "jellyfin"
		tag: "10.8.10"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	jellyfin: {
		chart: "charts/jellyfin"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	jellyfin: {
		chart: charts.jellyfin
		values: {
			pcloudInstanceId: global.id
			ingress: {
				className: input.network.ingressClass
				domain: _domain
			}
			image: {
				repository: images.jellyfin.fullName
				tag: images.jellyfin.tag
				pullPolicy: images.jellyfin.pullPolicy
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
