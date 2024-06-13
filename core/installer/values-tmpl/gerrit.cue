input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
	key: #SSHKey
	sshPort: int @name(SSH Port) @role(port)
}

_domain: "\(input.subdomain).\(input.network.domain)"
url: "https://\(_domain)"

name: "Gerrit"
namespace: "app-gerrit"
readme: "gerrit"
description: "Gerrit Code Review is a web-based code review tool built on Git version control. Gerrit provides a framework you and your teams can use to review code before it becomes part of the code base. Gerrit works equally well in open source projects that limit the number of users who can approve changes (typical in open source software development) and in projects in which all contributors are trusted."
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 32 32'><path fill='currentColor' d='m16.865 3.573l-.328-.359c.005-.005.385-.354.552-.542c.161-.198.453-.646.458-.651l.406.26c-.021.021-.313.479-.5.698s-.573.573-.589.594zm2.104 14.125c-.016-.005-.323-.203-.49-.292a11.695 11.695 0 0 0-.563-.255l.286-.818l-1.198-.589l-.38 1.161c-.234.005-.953.068-2.016.516c-1.281.536-2.25 1.37-2.26 1.375l-.193.167l.859.031l.026-.021c.005-.01.958-.714 1.49-.943c.12-.047.276-.099.443-.135c-.281.135-.589.302-.802.422c-.266.161-.76.51-.781.526l-.25.172l.911.021l.026-.01c.016-.01 1.552-.833 2.385-1.016l.26-.063c.193-.047.328-.083.563-.083c.208 0 .49.026.917.094c.531.078.88.208.885.214l.318.125l-.427-.583l-.016-.01zM6.995 7.969h-.042L5.614 9.193v.036c-.021.354.104.693.344.958c.24.26.557.411.911.422h.057c.708 0 1.286-.547 1.323-1.25a1.338 1.338 0 0 0-1.255-1.391zm-.063 2.442H6.88a.999.999 0 0 1-.443-.115a.692.692 0 0 0 .797-.766a.689.689 0 0 0-.771-.589a.69.69 0 0 0-.594.708a1.113 1.113 0 0 1-.057-.37l1.214-1.115a1.141 1.141 0 0 1 1.026 1.177a1.12 1.12 0 0 1-1.125 1.068zM19.37 5.443l-.391-.26l-.547.354l-.526-.38l-.401.24l.542.391l-.557.359l.396.24l.536-.339l.516.375l.411-.229l-.542-.391zM32 26.031c-.286-.276-.557-.552-.839-.833a76.772 76.772 0 0 1-1.891-1.984a42.827 42.827 0 0 1-2.109-2.458a17.1 17.1 0 0 1-.859-1.172a13.625 13.625 0 0 1-.891-1.62a30.908 30.908 0 0 1-.771-1.807c.323.276.62.589.885.922c.026-.286.057-.573.078-.865l.031-.427c0-.047.016-.089-.01-.13a.46.46 0 0 0-.073-.099c-.156-.198-.349-.375-.536-.552a32.19 32.19 0 0 0-.781-.708l-.24-.208c-.036-.036-.078-.068-.115-.099c-.042-.042-.057-.13-.073-.182l-.208-.641c.813.38 1.479.99 2.104 1.62c.005-.292.005-.578 0-.87c0-.151 0-.302-.01-.458c0-.042.01-.135-.021-.172c-.016-.026-.042-.047-.057-.073c-.146-.156-.313-.286-.474-.417c-.229-.193-.469-.37-.703-.547c-.208-.156-.422-.307-.635-.458c-.026-.021-.104-.052-.089-.078l.052-.109c.031-.047.021-.057.073-.036l.229.078c.536.208 1.036.49 1.521.813a8.413 8.413 0 0 0-.698-1.729a13.423 13.423 0 0 0-1.953-2.807a16.75 16.75 0 0 0-1.625-1.594c-.297-.25-.609-.49-.932-.708c-.151-.099-.297-.198-.458-.292c-.068-.036-.141-.073-.203-.125c-.24-.188-.484-.375-.729-.568c.318.13.625.276.917.448c-.167-.26-.453-.443-.724-.578a7.706 7.706 0 0 0-1.292-.505c.151-.161.313-.307.464-.464c.151-.161.297-.328.438-.5c.172-.198.339-.396.5-.604l-2.182-1.37c-.161.323-.37.63-.609.906c-.24.271-.521.49-.802.719c-.25.208-.505.417-.75.625c-.068.057-.125.115-.198.161c-.031.031-.125.005-.167.005h-.318c-.396.01-.792.042-1.188.094c-.078.005-.151.016-.234.01l-.234-.016c-.182-.01-.365-.021-.547-.021c-.385-.005-.771 0-1.161.031a6.8 6.8 0 0 0-.969.151a2.52 2.52 0 0 0-.88.417c-.26.188-.516.427-.672.708c-.156.276-.229.604-.286.917c-.177.016-.354.016-.531.021a6.527 6.527 0 0 0-1.614.292a4.907 4.907 0 0 0-1.781 1a5.58 5.58 0 0 0-.719.797c-.026.031-.052.068-.083.089c-.016.01-.036.021-.047.036a.693.693 0 0 1-.068.099l-.177.286c-.224.38-.37.792-.51 1.208l-.063.161l.047-.026a3.772 3.772 0 0 0-.036.271l-.01.135v.073l-.089.016a5.981 5.981 0 0 0-.531.135a1.86 1.86 0 0 0-.448.203c-.141.083-.26.203-.38.318a3.42 3.42 0 0 0-.917 1.5c-.135.469-.182.984-.078 1.458c.026.12.063.25.141.349c.099.12.266.167.417.125c.177-.047.333-.161.495-.245l.422-.214c.604-.297 1.24-.594 1.917-.698c.047-.005.13.089.172.12c.068.052.135.099.198.141c.146.094.297.172.448.24c.349.151.719.234 1.089.307c.672.141 1.354.229 2.042.24c.276.005.552 0 .833-.021c.297-.026.599-.068.901-.068c.333-.005.661.031.99.073c.339.042.677.089 1.016.141c.693.104 1.375.224 2.057.37c-.151.24-.302.484-.448.729c-.01.016-.099 0-.12 0a.932.932 0 0 0-.167 0c-.099 0-.203.01-.302.026a3.886 3.886 0 0 0-.818.203c-.656.245-1.255.646-1.771 1.115c-.297.26-.573.542-.813.854a6.7 6.7 0 0 0-.182.255c.135-.031.281-.057.422-.094c.083-.021.156-.036.234-.052c.026-.01.036-.021.068-.036a9.04 9.04 0 0 1 .922-.75c.151-.104.302-.203.469-.286c.219-.109.469-.172.708-.229c-.438.24-.906.464-1.302.776c-.229.188-.438.391-.656.589l.875-.141c.01 0 .016-.005.031-.016l.224-.125c.151-.083.307-.167.464-.245c.318-.167.641-.323.974-.453c.318-.125.641-.24.979-.302c.292-.063.568-.068.865 0c.453.099.891.307 1.292.552c.026.021.047.047.073.021c.021-.021.13-.099.12-.125l-.24-.443c-.021-.042-.031-.068-.068-.089l-.177-.104a7.51 7.51 0 0 1-.677-.443c-.052-.031-.104-.052-.109-.12c-.01-.057.016-.12.036-.177c.042-.12.104-.229.172-.333c.047-.078.099-.146.146-.219c.021-.026.016-.031.042-.021l.161.047c.313.104.625.214.948.292c.359.094.724.167 1.094.234l.063.016c-.073-.042-.12-.12-.177-.182c-.031-.042-.047-.068-.099-.078l-.141-.031c-.099-.021-.193-.036-.297-.063a8.596 8.596 0 0 1-1.036-.281a15.4 15.4 0 0 0-1.526-.427a37.19 37.19 0 0 0-1.953-.365c-.333-.057-.667-.099-1-.146a15.528 15.528 0 0 0-.995-.12c-.719-.042-1.432.12-2.156.109c-.484-.005-.979-.073-1.458-.141l-.094-.01c.339-.125.667-.25 1-.38c.318-.125.63-.255.943-.385c.167-.068.333-.141.495-.208c.151-.068.302-.135.438-.229c.547-.37.901-.969 1.302-1.479c.365-.479.781-.932 1.318-1.208c.172-.089.349-.156.536-.208c-.38-.583-.734-1.24-.833-1.938l.125.047c.047.016.089.021.099.063l.036.177c.036.12.073.234.115.349c.099.255.214.5.349.734a10.87 10.87 0 0 0 1.021 1.495c.719.917 1.526 1.74 2.313 2.589c.193.208.37.432.547.656c.203.25.406.5.609.745c.161.188.313.38.474.568l.125.151c.021.026.052.036.083.047c.807.401 1.62.802 2.427 1.193c.583.281 1.161.563 1.75.833c.313.146.625.292.948.427c.036.016.083.036.13.052c.021.01.036.021.063.031l.021.063c.036.099.068.193.099.292c.068.188.13.37.198.552c.443 1.219.927 2.422 1.526 3.568a71.35 71.35 0 0 0 1.453 2.589c.536.906 1.083 1.802 1.635 2.698c.443.714.885 1.432 1.344 2.141c.193.302.385.615.583.917l.083.125l1.292-1.896c.01-.01.109-.135.099-.146l-.208-.323c-.385-.599-.776-1.198-1.161-1.797l-1.245-1.932l.875 1.063l1.49 1.802c.161.193.313.385.469.583c.292-.536.589-1.068.885-1.599c.115-.219.234-.443.354-.656zM16.172 2.552c.411-.328.75-.75 1.01-1.208l1.573.995l.24.146c-.328.401-.661.807-1.036 1.167a2.002 2.002 0 0 0-.141.135c-.026.036-.063.068-.094.099l-.042.052c-.031-.01-.063-.021-.094-.026c-.193-.052-.385-.104-.578-.146a11.358 11.358 0 0 0-1.182-.203c-.255-.031-.516-.052-.771-.078c.365-.313.74-.625 1.115-.932zM13.833 4.38c.313-.13.646-.198.974-.255a7.25 7.25 0 0 1 1.984-.052c.474.052.938.141 1.391.281l-.188.151l-.302-.083c-.188-.036-.375-.078-.563-.109a7.856 7.856 0 0 0-1-.083a6.98 6.98 0 0 0-1.828.208a5.861 5.861 0 0 0-1.172.443c-.38.208-.74.464-1.036.776a3.391 3.391 0 0 0-.479.609c-.078.12-.141.24-.203.365a1.56 1.56 0 0 0-.078.193l-.042.099a.406.406 0 0 1-.016.052l-.089-.016l-.109-.01c.313-.964 1.016-1.719 1.891-2.203c.276-.151.568-.286.865-.37zm-5.239.495c.354-.51.917-.865 1.521-.995c.667-.13 1.354-.156 2.031-.141c-.698.177-1.401.438-1.984.87a3.041 3.041 0 0 0-1.245.609a3.08 3.08 0 0 0-.328.318a1.25 1.25 0 0 0-.13.156c-.016.016-.036.036-.047.063h-.115c.031-.177.073-.359.13-.531c.042-.12.089-.24.161-.349zm1.172.078a4.718 4.718 0 0 0-.521.625c-.063.089-.13.203-.24.25c-.115.052-.26-.005-.375-.036a2.76 2.76 0 0 1 1.135-.839zM3.078 8.781c.104-.214.24-.422.365-.62c.021-.036.073-.063.099-.083c.068-.047.13-.094.193-.146c.411-.297.828-.594 1.25-.87c.224-.146.443-.286.672-.411c.24-.135.49-.24.75-.328A9.45 9.45 0 0 1 7.834 6c.229-.031.479-.078.708-.021c-.443.25-.88.5-1.323.745c-.453.255-.917.49-1.38.734c-.443.24-.88.5-1.307.771c-.448.276-.891.557-1.333.839c-.109.068-.214.141-.323.208c.063-.167.12-.339.203-.495zm1.344 4.073c-.036.073-.177.057-.25.057c-.125 0-.245 0-.37.005a4.13 4.13 0 0 0-1.01.188c-.635.193-1.229.5-1.823.802c-.13.073-.271.182-.422.219a.203.203 0 0 1-.219-.083a.704.704 0 0 1-.083-.266a2.255 2.255 0 0 1-.047-.49c0-.443.104-.88.286-1.281c.13-.281.297-.536.495-.766c.203-.234.438-.469.719-.599c.474-.219 1.036-.286 1.552-.313c.099-.01.193-.01.292-.01c.13 0 .286-.021.411.026c.099.036.161.141.203.229c.057.141.099.302.13.448c.083.349.167.698.193 1.057c.016.156.026.318.005.474c-.01.099-.016.208-.063.302zm3.771-2.635a4.38 4.38 0 0 1-.823.401a5.11 5.11 0 0 1-.88.24c-.13.016-.26.036-.391.026c-.135 0-.255-.047-.391-.089a4.004 4.004 0 0 1-.76-.307a.666.666 0 0 1-.229-.203a.34.34 0 0 1-.031-.229c.016-.307.125-.609.271-.88c.25-.458.641-.818 1.12-1.036c1.172-.526 2.484-.036 3.479.656l.109.078c-.208.224-.417.432-.63.641c-.266.25-.542.505-.849.703zm2.885-2.568a11.921 11.921 0 0 1-1.807-.984c.599.25 1.245.401 1.885.495c.344.047.698.094 1.042.104c.375.016.755-.026 1.12-.099c.729-.135 1.427-.406 2.089-.734s1.286-.714 1.885-1.146c.286-.203.573-.417.844-.646c.026-.021.229-.219.245-.208l.052.042l.719.557c.438.339.875.677 1.318 1.016a68.458 68.458 0 0 1-3.672 1.203c-.693.208-1.38.401-2.083.563c-.552.13-1.115.25-1.677.276c-.677.031-1.339-.177-1.958-.438zm11.797 5.255c.104.026.198.063.286.089l.13.047c.021.005.036.021.057.026l.026.078c.063.198.12.385.188.578c-.198-.172-.401-.339-.599-.505l-.12-.099c-.031-.021-.063-.031-.042-.063l.078-.151zm-.891 1.922l.047-.083l.036-.057c.016-.026.01-.031.042-.016c.172.068.344.146.51.224c.323.146.635.307.938.484c.146.089.292.182.432.276l.198.141l.099.073c.042.036.057.083.078.135c.135.375.292.755.448 1.125c.104.255.219.505.333.755a14.762 14.762 0 0 0-1.276-1.391a20.964 20.964 0 0 0-1.438-1.307l-.432-.354zm5.011 8.568l-.161.12l.01.021l.083.125l.365.557l1.203 1.87c.417.641.828 1.286 1.245 1.927l.411.641l.109.177a.256.256 0 0 1 .042.063c-.349.51-.698 1.026-1.047 1.536c-.036.047-.068.099-.099.146c-.318-.495-.635-.99-.953-1.49c-.531-.844-1.057-1.703-1.578-2.552a106.797 106.797 0 0 1-1.688-2.854c-.51-.901-1-1.818-1.411-2.771a39.425 39.425 0 0 1-1.078-2.828c.646.26 1.307.49 1.974.698c.193.057.385.12.578.167l.083.026c.01 0 .021-.052.026-.068c.026-.083.042-.172.063-.26c.036-.167.068-.339.094-.505c.276.568.583 1.125.938 1.646c.286.422.594.828.917 1.229a43.68 43.68 0 0 0 2.188 2.531c.62.656 1.245 1.313 1.88 1.953l.516.516c.01.01.052.042.052.057l-.042.068l-.198.37l-.786 1.422c-.24-.292-.479-.578-.719-.875l-1.5-1.818c-.422-.516-.849-1.031-1.271-1.547l-.25-.297z'/></svg>"

ingress: {
	gerrit: {
		auth: enabled: false
		network: input.network
		subdomain: input.subdomain
		service: {
			name: "gerrit-gerrit-service"
			port: number: _httpPort // TODO(gio): make optional
		}
	}
}

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
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/ingress"
	}
	volume: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/volumes"
	}
	gerrit: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/gerrit"
	}
	oauth2Client: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/oauth2-client"
	}
	resourceRenderer: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/resource-renderer"
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

portForward: [#PortForward & {
	allocator: input.network.allocatePortAddr
	reservator: input.network.reservePortAddr
	sourcePort: input.sshPort
	serviceName: "gerrit-gerrit-service"
	targetPort: _sshPort
}]

_oauth2ClientCredentials: "gerrit-oauth2-credentials"
_gerritConfigMapName: "gerrit-config"

helm: {
	"oauth2-client": {
		chart: charts.oauth2Client
		info: "Creating OAuth2 client"
		values: {
			name: "gerrit-oauth2-client"
			secretName: _oauth2ClientCredentials
			grantTypes: ["authorization_code"]
			scope: "openid profile email"
			hydraAdmin: "http://hydra-admin.\(global.id)-core-auth.svc.cluster.local"
			redirectUris: ["https://\(_domain)/oauth"]
		}
	}
	"config-renderer": {
		chart: charts.resourceRenderer
		info: "Generating Gerrit configuration"
		values: {
			name: "config-renderer"
			secretName: _oauth2ClientCredentials
			resourceTemplate: """
apiVersion: v1
kind: ConfigMap
metadata:
  name: \(_gerritConfigMapName)
  namespace: \(release.namespace)
data:
  replication.config: |
    [gerrit]
      autoReload = false
      replicateOnStartup = true
      defaultForceUpdate = true
  gerrit.config: |
    [gerrit]
      basePath = git # FIXED
      serverId = gerrit-1
      # The canonical web URL has to be set to the Ingress host, if an Ingress
      # is used. If a LoadBalancer-service is used, this should be set to the
      # LoadBalancer's external IP. This can only be done manually after installing
      # the chart, when you know the external IP the LoadBalancer got from the
      # cluster.
      canonicalWebUrl = https://\(_domain)
      disableReverseDnsLookup = true
    [index]
      type = LUCENE
    [auth]
      type = OAUTH
      gitBasicAuthPolicy = HTTP
      userNameToLowerCase = true
      userNameCaseInsensitive = true
    [plugin "gerrit-oauth-provider-pcloud-oauth"]
      root-url = https://hydra.\(global.domain)
      client-id = "{{ .client_id }}"
      client-secret = "{{ .client_secret }}"
      link-to-existing-openid-accounts = true
    [download]
      command = branch
      command = checkout
      command = cherry_pick
      command = pull
      command = format_patch
      command = reset
      scheme = http
      scheme = anon_http
    [httpd]
      # If using an ingress use proxy-http or proxy-https
      listenUrl = proxy-http://*:8080/
      requestLog = true
      gracefulStopTimeout = 1m
    [sshd]
      listenAddress = 0.0.0.0:29418
      advertisedAddress = \(_domain):\(input.sshPort)
    [transfer]
      timeout = 120 s
    [user]
      name = Gerrit Code Review
      email = gerrit@\(global.domain)
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
      javaOptions = -Xmx4g
"""
		}
	}
	gerrit: {
		chart: charts.gerrit
		info: "Installing Gerrit server"
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
					type: "LoadBalancer"
					externalTrafficPolicy: ""
					additionalAnnotations: {
						"metallb.universe.tf/address-pool": global.id
					}
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
						name: "download-commands"
					}, {
						name: "singleusergroup"
					}, {
						name: "codemirror-editor"
					}, {
						name: "reviewnotes"
					}, {
						name: "oauth"
						url: "https://drive.google.com/uc?export=download&id=1rSUpZCAVvHZTmRgUl4enrsAM73gndjeP"
						sha1: "cbdc5228a18b051a6e048a8e783e556394cc5db1"
					}, {
						name: "webhooks"
					}]
					libs: []
					cache: enabled: false
				}
				etc: {
					secret: {
						ssh_host_ecdsa_key: input.key.private
						"ssh_host_ecdsa_key.pub": input.key.public
					}
					existingConfigMapName: _gerritConfigMapName
				}
			}
		}
	}
	"git-volume": {
		chart: charts.volume
		info: "Creating disk for Git repositories"
		values: volumes.git
	}
	"log-volume": {
		chart: charts.volume
		info: "Creating disk for logging"
		values: volumes.logs
	}
}
