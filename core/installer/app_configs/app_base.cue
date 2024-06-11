import (
  "net"
)

name: string | *""
description: string | *""
readme: string | *""
icon: string | *""
namespace: string | *""

help: [...#HelpDocument] | *[]

#HelpDocument: {
	title: string
	contents: string
	children: [...#HelpDocument] | *[]
}

url: string | *""

#AppType: "infra" | "env"
appType: #AppType | *"env"

#Auth: {
  enabled: bool | *false // TODO(gio): enabled by default?
  groups: string | *"" // TODO(gio): []string
}

#Network: {
	name: string
	ingressClass: string
	certificateIssuer: string | *""
	domain: string
	allocatePortAddr: string
}

#Image: {
	registry: string | *release.imageRegistry
	repository: string
	name: string
	tag: string
	pullPolicy: string | *"IfNotPresent"
	imageName: "\(repository)/\(name)"
	fullName: "\(registry)/\(imageName)"
	fullNameWithTag: "\(fullName):\(tag)"
}

#Chart: #GitRepositoryRef | #HelmRepositoryRef

#GitRepositoryRef: {
    name: string
	kind: "GitRepository"
    address: string
	branch: string
    path: string
}

#HelmRepositoryRef: {
    name: string
	kind: "HelmRepository"
    repository: string
	name: string
	tag: string
}

#EnvNetwork: {
	dns: net.IPv4
	dnsInClusterIP: net.IPv4
	ingress: net.IPv4
	headscale: net.IPv4
	servicesFrom: net.IPv4
	servicesTo: net.IPv4
}

#Release: {
	appInstanceId: string
	namespace: string
	repoAddr: string
	appDir: string
	imageRegistry: string | *"docker.io"
}

#PortForward: {
	allocator: string
	protocol: "TCP" | "UDP" | *"TCP"
	sourcePort: int
	targetService: string
	targetPort: int
}

portForward: [...#PortForward] | *[]

global: #Global
release: #Release

images: {}

images: {
	for key, value in images {
		"\(key)": #Image & value
	}
    for _, value in _ingressValidate {
        for name, image in value.out.images {
            "\(name)": #Image & image
        }
    }
}

charts: {}

charts: {
	for key, value in charts {
		"\(key)": #Chart & value & {
            name: key
        }
	}
    for _, value in _ingressValidate {
        for name, chart in value.out.charts {
            "\(name)": #Chart & chart & {
                name: name
            }
        }
    }
}

localCharts: {
	for key, _ in charts {
		"\(key)": {
        }
    }
}

#ResourceReference: {
    name: string
    namespace: string
}

#Helm: {
	name: string
	dependsOn: [...#ResourceReference] | *[]
	info: string | *""
	...
}

_helmValidate: {
	for key, value in helm {
		"\(key)": #Helm & value & {
			name: key
		}
	}
	for key, value in _ingressValidate {
		for ing, ingValue in value.out.helm {
            // TODO(gio): support multiple ingresses
			// "\(key)-\(ing)": #Helm & ingValue & {
			"\(ing)": #Helm & ingValue & {
				// name: "\(key)-\(ing)"
				name: ing
			}
		}
	}
}

#HelmRelease: {
	_name: string
	_chart: _
	_values: _
	_dependencies: [...#ResourceReference] | *[]
	_info: string | *""

	apiVersion: "helm.toolkit.fluxcd.io/v2beta1"
	kind: "HelmRelease"
	metadata: {
		name: _name
   		namespace: release.namespace
        annotations: {
          "dodo.cloud/installer-info": _info
        }
	}
	spec: {
		interval: "1m0s"
		dependsOn: _dependencies
		chart: spec: _chart
		values: _values
	}
}

output: {
	for name, r in _helmValidate {
		"\(name)": #HelmRelease & {
			_name: name
            _chart: localCharts[r.chart.name]
			_values: r.values
			_dependencies: r.dependsOn
			_info: r.info
		}
	}
}

#SSHKey: {
	public: string
	private: string
}

#HelpDocument: {
    title: string
    contents: string
    children: [...#HelpDocument]
}

help: [...#HelpDocument] | *[]

url: string | *""
