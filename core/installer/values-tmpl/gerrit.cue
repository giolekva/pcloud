input: {
	network: #Network
	subdomain: string
}

_domain: "\(input.subdomain).\(input.network.domain)"

name: "gerrit"
namespace: "app-gerrit"
readme: "gerrit"
description: "gerrit"
icon: ""

// TODO(gio): configure busybox
_images: {
	gerrit: #Image & {
		repository: "k8sgerrit"
		name: "gerrit"
		tag: _latest
		pullPolicy: "Always"
	}
	gerritInit: #Image & {
		repository: "k8sgerrit"
		name: "gerrit-init"
		tag: _latest
		pullPolicy: "Always"
	}
	gitGC: #Image & {
		repository: "k8sgerrit"
		name: "git-gc"
		tag: _latest
		pullPolicy: "Always"
	}
}
images: _images

charts: {
	ingress: {
		chart: "charts/ingress"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	volume: {
		chart: "charts/volumes"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
	gerrit: {
		chart: "charts/gerrit"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

volumes: {
	git: {
		name: "git"
		accessMode: "ReadWriteMany"
		size: "50Gi"
	}
	logs: {
		name: "logs"
		accessMode: "ReadWriteMany"
		size: "5Gi"
	}
}

_dockerIO: "docker.io"
_latest: "latest"

_longhorn: "longhorn"

_httpPort: 80
_sshPort: 22

helm: {
	gerrit: {
		chart: charts.gerrit
		values: {
			images: {
				busybox: {
					registry: _dockerIO
					tag: _latest
				}
				registry: {
					name: _dockerIO
					ImagePullSecret: create: false
					version: _latest
					imagePullPolicy: "Always"
				}
			}
			storageClasses: {
				default: {
					name: _longhorn
					create: false
				}
				shared: {
					name: _longhorn
					create: false
				}
			}
			persistence: {
				enabled: true
				size: "10Gi"
			}
			nfsWorkaround: {
				enabled: false
				chownOnStartup: false
				idDomain: _domain
			}
			networkPolicies: enabled: false
			gitRepositoryStorage: {
				externalPVC: {
					use: true
					name: volumes.git.name
				}
			}
			logStorage: {
				enabled: true
				externalPVC: {
					use: true
					name: volumes.logs.name
				}
			}
			ingress: enabled: false
			gitGC: {
				image: _images.gitGC.imageName
				logging: persistence: enabled: false
			}
			gerrit: {
				images: {
					gerritInit: _images.gerritInit.imageName
					gerrit: _images.gerrit.imageName
				}
				service: {
					type: "ClusterIP"
					externalTrafficPolicy: ""
					http: port: _httpPort
					ssh: {
						enabled: true
						port: _sshPort
					}
				}
				pluginManagement: {
					plugins: [{
						name: "gitiles"
					}, {
						name: "healthcheck"
					}]
					libs: []
					cache: enabled: false
				}
				etc: {
					config: {
						"replication.config": ###"""
[gerrit]
  autoReload = false
  replicateOnStartup = true
  defaultForceUpdate = true"""###
						"gerrit.config": ###"""
[gerrit]
  basePath = git # FIXED
  serverId = gerrit-1
  # The canonical web URL has to be set to the Ingress host, if an Ingress
  # is used. If a LoadBalancer-service is used, this should be set to the
  # LoadBalancer's external IP. This can only be done manually after installing
  # the chart, when you know the external IP the LoadBalancer got from the
  # cluster.
  canonicalWebUrl = https://gerrit.p.v0.dodo.cloud
  disableReverseDnsLookup = true
[index]
  type = LUCENE
[auth]
  type = DEVELOPMENT_BECOME_ANY_ACCOUNT
  gitBasicAuthPolicy = HTTP
[httpd]
  # If using an ingress use proxy-http or proxy-https
  listenUrl = proxy-http://*:8080/
  requestLog = true
  gracefulStopTimeout = 1m
[sshd]
  listenAddress = off
[transfer]
  timeout = 120 s
[user]
  name = Gerrit Code Review
  email = gerrit@p.v0.dodo.cloud
  anonymousCoward = Unnamed User
[cache]
  directory = cache
[container]
  user = gerrit # FIXED
  javaHome = /usr/lib/jvm/java-11-openjdk # FIXED
  javaOptions = -Djavax.net.ssl.trustStore=/var/gerrit/etc/keystore # FIXED
  javaOptions = -Xms200m
  # Has to be lower than 'gerrit.resources.limits.memory'. Also
  # consider memories used by other applications in the container.
  javaOptions = -Xmx4g"""###
					}
				}
			}
		}
	}
	ingress: {
		chart: charts.ingress
		values: {
			domain: _domain
			ingressClassName: input.network.ingressClass
			certificateIssuer: input.network.certificateIssuer
			service: {
				name: "gerrit-gerrit-service"
				port: number: _httpPort
			}
		}
	}
	"git-volume": {
		chart: charts.volume
		values: volumes.git
	}
	"log-volume": {
		chart: charts.volume
		values: volumes.logs
	}
}
