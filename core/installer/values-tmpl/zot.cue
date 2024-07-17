import (
	"encoding/yaml"
	"encoding/json"
)

input: {
    network: #Network @name(Network)
    subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"
url: "https://\(_domain)"

name: "Zot"
namespace: "app-zot"
readme: "OCI-native container image registry, simplified"
description: "OCI-native container image registry, simplified"
icon: """
<svg width='50px' height='50px' xmlns='http://www.w3.org/2000/svg' viewBox='0 0 39.68503937 36.27146462'>
  <defs>
    <style>
      .cls-1 {
        fill: currentColor;
      }

      .cls-2 {
        fill: none;
        stroke: #3a3a3a;
        stroke-miterlimit: 10;
        stroke-width: .98133445px;
      }
    </style>
  </defs>
  <rect class='cls-2' x='-9.97439025' y='-11.68117763' width='59.63381987' height='59.63381987'/>
  <g>
    <path class='cls-1' d='m29.74314495,24.98575641c-.75549716.74180664-1.41447384,1.43782557-2.10953123,2.09286451,1.88242421.2298085,3.61301638.54546895,5.1121059.94080001,3.75092895.97523237,4.57602343,2.025465,4.57602343,2.25055737,0,.22504658-.82509448,1.27527921-4.57602343,2.25055737-3.45092789.90018632-8.02713447,1.3878254-12.90334211,1.3878254-1.50330199,0-2.98132917-.04670346-4.40734152-.13717997,3.86411616-1.46447402,11.01249296-5.7430605,20.91142889-17.34213977C25.02701114,26.53182412,10.52274765,29.31182475,2.37253582,30.22290854c.09505528-.29812376,1.02930765-1.28379573,4.56668274-2.20348761.98370309-.2566401,2.06154572-.47848154,3.20898565-.66630271,2.60660258-.52536815,7.09562936-1.84945706,10.511026-3.07968117-.27252843-.00302199-.54505686-.00494507-.81685269-.00494507-9.56486882,0-19.84237751,1.87546447-19.84237751,6.00148632s10.27750869,6.00148632,19.84237751,6.00148632,19.87992343-1.87546447,19.84256066-6.00148632c0-2.67107167-4.30917267-4.3977261-9.94179322-5.28422189Z'/>
    <path class='cls-1' d='m19.84237751,12.00297264c4.12895226,0,8.39600024-.35036753,11.91139722-1.07953677-.06043977,4.29255173-6.00643139,9.89499819-8.79288808,11.37302537,3.02968099-1.43677245,16.76031538-5.95079933,16.72405152-16.29497492C39.68493817,1.87546447,29.40724633,0,19.84237751,0S0,1.87546447,0,6.00148632s10.27750869,6.00148632,19.84237751,6.00148632ZM6.93921856,3.75092895c3.45092789-.90027789,8.02695132-1.38787118,12.90315895-1.38787118s9.48996013.48759329,12.90334211,1.38787118c3.75092895.97518658,4.57602343,2.025465,4.57602343,2.25055737,0,.22500079-.82509448,1.27527921-4.57602343,2.25055737-3.45092789.90018632-8.02713447,1.3878254-12.90334211,1.3878254s-9.48977698-.48763908-12.90315895-1.3878254c-3.75092895-.97527816-4.57602343-2.02555658-4.57602343-2.25055737,0-.22509237.82509448-1.27537079,4.57602343-2.25055737Z'/>
    <path class='cls-1' d='m22.96088665,22.29646124c-.10128241.0480313-.19120946.09281168-.26776651.13406641.08553144-.03988109.17490904-.08484462.26776651-.13406641Z'/>
  </g>
</svg>"""

ingress: {
	zot: {
		auth: enabled: false
		network: input.network
		subdomain: input.subdomain
		service: {
			name: "zot"
			port: number: _httpPort // TODO(gio): make optional
		}
	}
}

// TODO(gio): configure busybox
images: {
	zot: {
		registry: "ghcr.io"
		repository: "project-zot"
		name: "zot-linux-amd64"
		tag: "v2.0.3"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	zot: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/zot"
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

volumes: zot: size: "100Gi"

_httpPort: 80
_oauth2ClientSecretName: "oauth2-client"

helm: {
	"oauth2-client": {
		chart: charts.oauth2Client
		info: "Creating OAuth2 client"
		// TODO(gio): remove once hydra maester is installed as part of dodo itself
		dependsOn: [{
			name: "auth"
			namespace: "\(global.namespacePrefix)core-auth"
		}]
		values: {
			name: "\(release.namespace)-zot"
			secretName: _oauth2ClientSecretName
			grantTypes: ["authorization_code"]
			responseTypes: ["code"]
			scope: "openid profile email groups"
			redirectUris: ["https://\(_domain)/zot/auth/callback/oidc"]
			hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
		}
	}
	"config-renderer": {
		chart: charts.resourceRenderer
		info: "Generating Zot configuration"
		values: {
			name: "config-renderer"
			secretName: _oauth2ClientSecretName
			resourceTemplate: yaml.Marshal({
				apiVersion: "v1"
				kind: "ConfigMap"
				metadata: {
					name: _zotConfigMapName
					namespace: "\(release.namespace)"
				}
				data: {
					"config.json": json.Marshal({
						storage: rootDirectory: "/var/lib/registry"
						http: {
							address: "0.0.0.0"
							port: "5000"
							externalUrl: url
							auth: openid: providers: oidc: {
								name: "dodo:"
								issuer: "https://hydra.\(networks.public.domain)"
								clientid: "{{ .client_id }}"
								clientsecret: "{{ .client_secret }}"
								keypath: ""
								scopes: ["openid", "profile", "email", "groups"]
							}
							accessControl: {
								repositories: {
									"**": {
										defaultPolicy: ["read", "create", "update", "delete"]
										anonymousPolicy: ["read"]
									}
								}
							}
						}
						log: level: "debug"
						extensions: {
							ui: enable: true
							search: enable: true
						}
					})
				}
			})
		}
	}
	zot: {
		chart: charts.zot
		info: "Installing Zot server"
		values: {
			image: {
				repository: images.zot.fullName
				tag: images.zot.tag
				pullPolicy: images.zot.pullPolicy
			}
			service: {
				type: "ClusterIP"
				additionalAnnotations: {
					"metallb.universe.tf/address-pool": global.id
				}
				port: _httpPort
			}
			ingress: enabled: false
			mountConfig: false
			persistence: true
			pvc: {
				create: false
				name: volumes.zot.name
			}
			extraVolumes: [{
				name: "config"
				configMap: name: _zotConfigMapName
			}]
			extraVolumeMounts: [{
				name: "config"
				mountPath: "/etc/zot"
			}]
			startupProbe: {}
		}
	}
}

_zotConfigMapName: "zot-config"

help: [{
	title: "Authenticate"
	contents: """
	First generate new API key.  
	docker login \\-\\-username=**\\<YOUR-USERNAME\\>**@\(networks.public.domain) \\-\\-password=**\\<YOUR-API-KEY\\>** \(_domain)  
	docker build \\-\\-tag=\(_domain)/**\\<IMAGE-NAME\\>**:**\\<IMAGE-TAG\\>** .  
	docker push \\-\\-tag=\(_domain)/**\\<IMAGE-NAME\\>**:**\\<IMAGE-TAG\\>**
	"""
}]
