#Input: {
	network: #Network
	subdomain: string
}

input: #Input

_domain: "\(input.subdomain).\(input.network.domain)"

readme: "rpuppy application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"

images: {
	rpuppy: {
		repository: "giolekva"
		name: "rpuppy"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	rpuppy: {
		source: {
			kind: "GitRepository"
			address: "pcloud"
		}
		chart: "./charts/rpuppy"
	}
}

helm: {
	rpuppy: {
		chart: charts.rpuppy
		values: {
			ingressClassName: input.network.ingressClass
			certificateIssuer: input.network.certificateIssuer
			domain: _domain
			image: {
				repository: images.rpuppy.fullName
				tag: images.rpuppy.tag
				pullPolicy: images.rpuppy.pullPolicy
			}
		}
	}
}

// TODO(gio): import

#Network: {
	name: string
	ingressClass: string
	certificateIssuer: string
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

#Global: {
	id: string
}

global: #Global

images: {
	for key, value in images {
		"\(key)": #Image & value
	}
}

#HelmRelease: {
	_name: string
	_chart: string
	_values: _

	apiVersion: "helm.toolkit.fluxcd.io/v2beta1"
	kind: "HelmRelease"
	metadata: {
		name: _name
   		namespace: "{{ .Release.Namespace }}"
	}
	spec: {
		interval: "1m0s"
		chart: {
			spec: {
				chart: _chart
				sourceRef: {
					kind: "HelmRepository"
					name: "pcloud"
					namespace: global.id
				}
			}
		}
		values: _values
	}
}

output: [
	for name, r in helm {
		#HelmRelease & {
			_name: name
			_chart: "rpuppy"
			_values: r.values
		}
	}
]
