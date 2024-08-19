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

#Volume: {
	size: string
	accessMode: "ReadWriteOnce" | "ReadOnlyMany" | "ReadWriteMany" | "ReadWriteOncePod" | *"ReadWriteOnce"
}

#volume: {
	#Volume
	name: string
}

volumes: {}
volumes: {
	for _, p in _postgresql {
		for k, v in p.out.volumes {
			"\(k)": v
		}
	}
}
volumes: {
	for key, value in volumes {
		"\(key)": #volume & value & {
			name: key
		}
	}
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
	reservator: string
	deallocator: string
	protocol: "TCP" | "UDP" | *"TCP"
	sourcePort: int
	serviceName: string
	targetService: "\(release.namespace)/\(serviceName)"
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
    for _, value in _postgresql {
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
    for _, value in _postgresql {
        for name, chart in value.out.charts {
            "\(name)": #Chart & chart & {
                name: name
            }
        }
    }
}
charts: {
	volume: {
		kind: "GitRepository"
		address: "https://code.v1.dodo.cloud/helm-charts"
		branch: "main"
		path: "charts/volumes"
	}
}

#PostgreSQL: {
	name: string
	version: "15.3"
	initSQL: string | *""
	size: string | *"1Gi"

	_size: size
	_volumeClaimName: "\(name)-postgresql"

	out: {
		images: {
			postgres: #Image & {
				repository: "library"
				name: "postgres"
				tag: version
				pullPolicy: "IfNotPresent"
			}
		}
		charts: {
			postgres: #Chart & {
				kind: "GitRepository"
				address: "https://code.v1.dodo.cloud/helm-charts"
				branch: "main"
				path: "charts/postgresql"
			}
		}
		volumes: {
			"\(_volumeClaimName)": size: _size
		}
		charts: {
			for key, value in charts {
				"\(key)": #Chart & value & {
					name: key
				}
			}
		}
		helm: {
			postgres: {
				chart: charts.postgres
				annotations: {
					"dodo.cloud/resource-type": "postgresql"
					"dodo.cloud/resource.postgresql.name": name
					"dodo.cloud/resource.postgresql.version": version
					"dodo.cloud/resource.postgresql.volume": _volumeClaimName
				}
				values: {
					fullnameOverride: "postgres-\(name)"
					image: {
						registry: images.postgres.registry
						repository: images.postgres.imageName
						tag: images.postgres.tag
						pullPolicy: images.postgres.pullPolicy
					}
					auth: {
						postgresPassword: "postgres"
						username: "postgres"
						password: "postgres"
						database: "postgres"
					}
					service: {
						type: "ClusterIP"
						port: 5432
					}
					global: {
						postgresql: {
							auth: {
								postgresPassword: "postgres"
								username: "postgres"
								password: "postgres"
								database: "postgres"
							}
						}
					}
					primary: {
						persistence: existingClaim: _volumeClaimName
						if initSQL != "" {
							initdb: scripts: "init.sql": initSQL
						}
						securityContext: {
							enabled: true
							fsGroup: 0
						}
						containerSecurityContext: {
							enabled: true
							runAsUser: 0
						}
					}
					volumePermissions: securityContext: runAsUser: 0
				}
			}
		}
	}
}

_ingressValidate: {}

postgresql: {}
_postgresql: {
	for key, value in postgresql {
		"\(key)": #PostgreSQL & value & {
			name: key
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
	annotations: {...} | *{}
	...
}

helm: {}
_helmValidate: {
	for key, value in helm {
		"\(key)": #Helm & value & {
			name: key
		}
	}
	for key, value in volumes {
		"\(key)-volume": #Helm & {
			chart: charts.volume
			info: "Creating disk for \(key)"
			annotations: {
				"dodo.cloud/resource-type": "volume"
				"dodo.cloud/resource.volume.name": value.name
				"dodo.cloud/resource.volume.size": value.size
			}
			values: value
		}
	}
	for key, value in _ingressValidate {
		for ing, ingValue in value.out.helm {
			"\(key)-\(ing)": #Helm & ingValue & {
				name: "\(key)-\(ing)"
			}
		}
	}
	for key, value in _postgresql {
		for post, postValue in value.out.helm {
			"\(key)-\(post)": #Helm & postValue & {
				name: "\(key)-\(post)"
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
	_annotations: {...} | *{}

	apiVersion: "helm.toolkit.fluxcd.io/v2beta1"
	kind: "HelmRelease"
	metadata: {
		name: _name
   		namespace: release.namespace
        annotations: _annotations & {
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
			_annotations: r.annotations
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
