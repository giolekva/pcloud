#Input: {
	network: #Network
	subdomain: string
}

input: #Input

_domain: "\(input.subdomain).\(input.network.domain)"

readme: "qbittorrent application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"

images: {
	qbittorrent: {
		registry: "lscr.io"
		repository: "linuxserver"
		name: "qbittorrent"
		tag: "4.5.3"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	qbittorrent: {
		chart: "charts/qbittorrent"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	qbittorrent: {
		chart: charts.qbittorrent
		values: {
			pcloudInstanceId: global.id
			ingress: {
				className: input.network.ingressClass
				domain: _domain
			}
			webui: port: 8080
			bittorrent: port: 6881
			storage: size: "1Ti"
			image: {
				repository: images.qbittorrent.fullName
				tag: images.qbittorrent.tag
				pullPolicy: images.qbittorrent.pullPolicy
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
