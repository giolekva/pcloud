import (
	"encoding/base64"
	"encoding/yaml"
	"list"
	"net"
)

input: {
	cluster?: #Cluster @name(Cluster)
}

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

#Namespace: {
	name: string
	kubeconfig?: string
}

namespaces: [...#Namespace] | *[]

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
	cluster?: #Cluster
	size: string
	accessMode: "ReadWriteOnce" | "ReadOnlyMany" | "ReadWriteMany" | "ReadWriteOncePod" | *"ReadWriteOnce"
}

#volume: {
	#Volume
	name: string
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

#Cluster: {
	name: string
	kubeconfig: string
	ingressClassName: string
}

global: #Global
release: #Release

#WriteFile: {
	path: string
	content: string
	owner: string
	permissions: string
	defer: bool | *true
}

#CloudInit: {
	runCmd: [...[...string]] | *[]
	writeFiles: [...#WriteFile] | *[]
}

#VPNDisabled: {
	enabled: false
}

#VPNEnabled: {
	enabled: true
	loginServer: string
	authKey: string
}

#VPN: #VPNEnabled | #VPNDisabled

#VirtualMachine: #WithOut & {
	name: string
	username: string
	domain: string
	vpn: #VPN | *{ enabled: false }
	codeServerEnabled: bool | *false
	cpuCores: int
	memory: string
	sshKnownHosts: [...string] | *[]
	sshAuthorizedKeys: [...string] | *[]
	cloudInit: #CloudInit

	_name: name
	_cpuCores: cpuCores
	_memory: memory

	_codeServerPort: 9090
	_codeServerCmd: [...[...string]] | *[]
	if codeServerEnabled {
		_codeServerCmd: [
			["sh", "-c", "curl -fsSL https://code-server.dev/install.sh | HOME=/home/\(username) sh"],
			["sh", "-c", "systemctl enable --now code-server@\(username)"],
			["sh", "-c", "sleep 10"],
			// TODO(gio): (maybe) listen only on tailscale interface
			["sh", "-c", "sed -i -e 's/127.0.0.1:8080/0.0.0.0:\(_codeServerPort)/g' /home/\(username)/.config/code-server/config.yaml"],
			["sh", "-c", "sed -i -e 's/auth: password/auth: none/g' /home/\(username)/.config/code-server/config.yaml"],
			["sh", "-c", "systemctl restart --now code-server@\(username)"],
	    ]
	}

	_vpnCmd: [...[...string]] | *[]
	if vpn.enabled {
		_vpnCmd: [
			["sh", "-c", "curl -fsSL https://tailscale.com/install.sh | sh"],
			// TODO(gio): (maybe) enable tailscale ssh
			["sh", "-c", "tailscale up --login-server=\(vpn.loginServer) --auth-key=\(vpn.authKey)"],
     	]
	}

	images: {}
	charts: {
		virtualMachine: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/virtual-machine"
		}
	}
	charts: {
		for key, value in charts {
			"\(key)": value & {
				name: key
			}
		}
	}
	helm: {
		"\(_name)-virtual-machine": {
			chart: charts.virtualMachine
			info: "Creating \(_name) virtual machine"
			annotations: {
				"dodo.cloud/resource-type": "virtual-machine"
				"dodo.cloud/resource.virtual-machine.name": _name
				"dodo.cloud/resource.virtual-machine.user": username
				"dodo.cloud/resource.virtual-machine.cpu-cores": "\(_cpuCores)"
				"dodo.cloud/resource.virtual-machine.memory": _memory
			}
			values: {
				name: _name
				cpuCores: _cpuCores
				memory: _memory
				disk: {
					source: "https://cloud.debian.org/images/cloud/bookworm-backports/latest/debian-12-backports-generic-amd64.qcow2"
					size: "64Gi"
				}
				ports: [22, 8080, _codeServerPort]
				servicePorts: [{
					name: "ssh"
					port: 22
					targetPort: 22
					protocol: "TCP"
				}, {
					name: "web"
					port: 80
					targetPort: 8080
					protocol: "TCP"
				}, {
					name: _codeServerPortName
					port: _codeServerPort
					targetPort: _codeServerPort
					protocol: "TCP"
				}]
				cloudInit: {
					userData: base64.Encode(null, "#cloud-config\n\(yaml.Marshal(_cloudInitUserData))")
					networkData: base64.Encode(null, yaml.Marshal({
						version: 2
						ethernets: {
							enp1s0: {
								dhcp4: true
							}
						}
					}))
				}
			}
			_cloudInitUserData: {
				system_info: {
					default_user: {
						name: username
						home: "/home/\(username)"
					}
				}
				password: "dodo" // TODO(gio): remove if possible
				chpasswd: {
					expire: false
				}
				hostname: _name
				ssh_pwauth: true
				disable_root: false
				ssh_authorized_keys: list.Concat([[
					"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOa7FUrmXzdY3no8qNGUk7OPaRcIUi8G7MVbLlff9eB/ lekva@gl-mbp-m1-max.local"
				], sshAuthorizedKeys])
				packages: [
					"curl",
					// "emacs",
					"git",
					"openssh-client",
				]
				write_files: list.Concat([[{
					path: "/home/\(username)/.gitconfig"
					content: """
					[user]
						name = \(username)
						email = \(username)@\(domain)

					"""
					owner: "\(username):\(username)"
					permissions: "0644"
				}], cloudInit.writeFiles])
				runcmd: list.Concat([
					[
						["sh", "-c", "chown -R \(username):\(username) /home/\(username)"],
						["sh", "-c", "ssh-keygen -t ed25519 -f /home/\(username)/.ssh/id_ed25519 -q -N ''"],
						["sh", "-c", "chown \(username):\(username) /home/\(username)/.ssh/id_ed25519*"],
						["sh", "-c", "chmod 0600 /home/\(username)/.ssh/id_ed25519*"],
						// TODO(gio): implement post app delete webhook to remove ssh key from memberships
						// TODO(gio): make memberships-api addr configurable
						["sh", "-c", "PUBKEY=$(cat /home/\(username)/.ssh/id_ed25519.pub) && curl --request POST --data \"{\\\"user\\\":\\\"\(username)\\\",\\\"publicKey\\\":\\\"${PUBKEY}\\\"}\" http://memberships-api.\(global.namespacePrefix)core-auth-memberships.svc.cluster.local/api/users/\(username)/keys"],
				    ],
					_vpnCmd,
					_codeServerCmd,
					[
						// TODO(gio): this waits for user keys are synced from memberships service back to the dodo-app.
						// We should inject this key into the dodo-app directly as well.
						// ["sh", "-c", "sleep 20"],
   				    ],
					cloudInit.runCmd])
			}
		}
	}
}

#PostgreSQL: #WithOut & {
	cluster?: #Cluster
	_cluster: cluster
	name: string
	version: "15.3"
	initSQL: string | *""
	size: string | *"1Gi"

	_size: size
	_volumeClaimName: "\(name)-postgresql"

	images: {
		postgres: {
			repository: "library"
			name: "postgres"
			tag: version
			pullPolicy: "IfNotPresent"
		}
	}
	charts: {
		postgres: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/postgresql"
		}
	}
	volumes: {
		"\(_volumeClaimName)": {
			size: _size
			if _cluster != _|_ {
				cluster: _cluster
			}
		}
	}
	helm: {
		postgres: {
			chart: charts.postgres
			if _cluster != _|_ {
				cluster: _cluster
			}
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

localCharts: {}
_localCharts: localCharts

#ResourceReference: {
    name: string
    namespace: string
}

#Helm: {
	name: string
	dependsOn: [...#ResourceReference] | *[]
	info: string | *""
	annotations: {...} | *{}
	cluster?: #Cluster | null
	targetNamespace?: string
	...
}

#HelmRelease: {
	_name: string
	_chart: _
	_values: _
	_dependencies: [...#ResourceReference] | *[]
	_info: string | *""
	_annotations: {...} | *{}
	_cluster?: #Cluster | null
	_namespace: string
	_targetNamespace?: string

	apiVersion: "helm.toolkit.fluxcd.io/v2beta1"
	kind: "HelmRelease"
	metadata: {
		name: _name
   		namespace: _namespace
        annotations: _annotations & {
          "dodo.cloud/installer-info": _info
        }
	}
	spec: {
		interval: "1m0s"
		dependsOn: _dependencies
		chart: spec: _chart
		if _targetNamespace != _|_ {
			targetNamespace: _targetNamespace
		}
		if _cluster != _|_ {
			if _cluster != null {
				kubeConfig: secretRef: {
					name: "cluster-kubeconfig"
					key: "kubeconfig"
				}
			}
		}
		install: remediation: retries: -1
		upgrade: remediation: retries: -1
		values: _values
	}
}

output: {
	for _, out in outs {
		images: out.images
		charts: out.charts
		clusterProxy: out.clusterProxy
		_lc: _localCharts & {
			for k, v in out.charts {
				"\(k)": {
					...
				}
			}
		}
		helm: {
			for name, r in out.helmR {
				"\(name)": #HelmRelease & {
					_name: name
					_chart: _lc[r.chart.name]
					_values: r.values
					_dependencies: r.dependsOn
					_info: r.info
					_annotations: r.annotations
					_namespace: release.namespace
					if r.cluster != _|_ {
						_cluster: r.cluster
					}
					if r.targetNamespace != _|_ {
						_targetNamespace: r.targetNamespace
					}
				}
			}
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

#WithOut: {
	cluster?: #Cluster
	if input.cluster != _|_ {
		cluster: #Cluster | *input.cluster
	}
	_cluster: cluster
	images: {...}
	charts: {...}
	helm: {...}
	clusterProxy: {...}
	images: {
		for k, v in images {
			"\(k)": #Image & v
		}
	}
	charts: {
		for k, v in charts {
			"\(k)": #Chart & v & {
				name: k
			}
		}
	}
	helm: {
		for k, v in helm {
			"\(k)": #Helm & v & {
				name: k
			}
		}
	}
	clusterProxy: {
		for k, v in clusterProxy {
			"\(k)": #ClusterProxy & v
		}
	}
	helmR: {
		for k, v in helm {
			"\(k)": v & {
				if v.cluster == _|_ && _cluster != _|_ {
					cluster: _cluster
				}
			}
		}
		...
	}
	...
}

outs: {
	"out": out
}
if out.cluster != _|_ {
	outs: kout: #WithOut & {
		cluster: out.cluster
		clusterName: cluster.name
		clusterKubeconfig: cluster.kubeconfig
		charts: {
			secret: {
				name: "secret"
				kind: "GitRepository"
				address: "https://code.v1.dodo.cloud/helm-charts"
				branch: "main"
				path: "charts/secret"
			}
 		}
		helm: {
			"cluster-kubeconfig": {
				chart: charts.secret
				cluster: null
				info: "Connecting to \(clusterName) cluster"
				values: {
					name: "cluster-kubeconfig"
					key: "kubeconfig"
					value: base64.Encode(null, clusterKubeconfig)
					keep: true
				}
			}
		}
	}
}

#WithOut: {
	cluster?: #Cluster
	_cluster: cluster
	charts: {
		volume: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/volumes"
		}
		secret: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/secret"
		}
		...
	}
	volumes: {...}
	volumes: {
		for k, v in volumes {
			"\(k)": #volume & v & {
				name: k
				if _cluster != _|_ {
					cluster: _cluster
				}
			}
		}
	}
	helm: {
		for k, v in volumes {
			"\(k)-volume": {
				chart: charts.volume
				info: "Creating disk for \(k)"
				annotations: {
					"dodo.cloud/resource-type": "volume"
					"dodo.cloud/resource.volume.name": v.name
					"dodo.cloud/resource.volume.size": v.size
				}
				values: v
				if v.cluster != _|_ {
					cluster: v.cluster
				}
			}
		}
	}
}

#WithOut: {
	cluster?: #Cluster
	_cluster: cluster
	postgresql: {...}
	postgresql: {
		for k, v in postgresql {
			"\(k)": #PostgreSQL & v & {
				if _cluster != _|_ {
					cluster: _cluster
				}
			}
		}
		...
	}
	images: {
		for k, v in postgresql {
			for x, y in v.images {
				"\(x)": y
			}
		}
	}
	charts: {
		for k, v in postgresql {
			for x, y in v.charts {
				"\(x)": y
			}
		}
	}
	helmR: {
		for k, v in postgresql {
			for x, y in v.helmR {
				"\(x)": y
			}
		}
		...
	}
	...
}

#WithOut: {
	vm: {...}
	_vm: {...}
	_vm: {
		for k, v in vm if len(v) > 0 {
			"\(k)": #VirtualMachine & v & {
				name: k
			}
		}
	}
	images: {
		for k, v in _vm {
			for x, y in v.images {
				"\(x)": y
			}
		}
	}
	charts: {
		for k, v in _vm {
			for x, y in v.charts {
				"\(x)": y
			}
		}
	}
	helmR: {
		for k, v in _vm {
			for x, y in v.helmR {
				"\(x)": y
			}
		}
		...
	}
	...
}

out: #WithOut
out: {}

_codeServerPortName: "code-server"

resources: { ... }

#ClusterProxy: {
	from: string
	to: string
}

// TODO(gio): Move this inside #WithOut definition
if out.cluster != _|_ {
	namespaces: [{
		name: release.namespace
		kubeconfig: out.cluster.kubeconfig
	}]
}
