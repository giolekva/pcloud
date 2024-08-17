input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"

name: "Matrix"
namespace: "app-matrix"
readme: "matrix application will be installed on \(input.network.name) network and be accessible to any user on https://\(_domain)"
description: "An open network for secure, decentralised communication"
icon: """
<svg width='50px' height='50px' xmlns='http://www.w3.org/2000/svg' viewBox='0 0 39.68503937 39.68503937'>
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
  <rect class='cls-2' x='-9.97439025' y='-9.97439025' width='59.63381987' height='59.63381987'/>
  <path class='cls-1' d='m1.04503942.90944884v37.86613982h2.72503927v.90945071H0V0h3.77007869v.90944884H1.04503942Zm11.64590578,12.00472508v1.91314893h.05456692c.47654392-.69956134,1.10875881-1.27913948,1.84700726-1.69322862.71598361-.40511792,1.54771632-.60354293,2.48031496-.60354293.89291332,0,1.70811022.17692893,2.44889755.51921281.74078733.34393731,1.29803124.96236184,1.68661493,1.83212566.41999952-.61842453.99212662-1.16740212,1.70976444-1.64031434.71763782-.47456723,1.57086583-.71102334,2.55637717-.71102334.74905523,0,1.44188933.09259881,2.08346495.27614143.64157561.18188998,1.18393635.47291301,1.64196855.8763783.45637641.40511792.80858321.92433073,1.06818882,1.57252004.25133929.6481893.3803142,1.42700774.3803142,2.34307056v9.47149555h-3.88417161v-8.02133831c0-.4729138-.01653581-.92433073-.0529127-1.34267762-.02666609-.3797812-.12779852-.75060537-.2976383-1.09133833-.16496703-.31157689-.41647821-.56882971-.72425151-.74078733-.32078781-.1818892-.75566893-.27448879-1.29803124-.27448879-.54897601,0-.99212662.10582699-1.32779444.3125199-.33038665.20312114-.60355081.48709839-.79370003.82511744-.19910782.35594888-.32873086.74650374-.38196842,1.15086631-.06370056.42978918-.09685576.86355382-.09921329,1.29803124v7.88409548h-3.8858274v-7.93700819c0-.41999952-.00661369-.83173271-.0297632-1.24346433-.01353647-.38990201-.09350161-.7746348-.23645611-1.13763734-.13486952-.34292964-.3751576-.63417029-.68622041-.83173271-.32078781-.20669291-.78708634-.31417253-1.41212614-.31417253-.18354341,0-.42826743.03968532-.72590573.1223628-.2976383.08433012-.59527502.23645611-.87637751.46629853-.31383822.26829772-.56214032.60483444-.72590573.98385871-.19842501.42826743-.29763751.99212662-.29763751,1.68661335v8.21149541h-3.88417713v-14.16259852l3.66259868.00000079Zm25.94905485,25.86141789V.90944884h-2.72504056v-.90944884h3.77007988v39.68503937h-3.77007988v-.90944756h2.72504056Z'/>
</svg>"""

images: {
	matrix: {
		repository: "matrixdotorg"
		name: "synapse"
		tag: "v1.104.0"
		pullPolicy: "IfNotPresent"
	}
	postgres: {
		repository: "library"
		name: "postgres"
		tag: "15.3"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	oauth2Client: {
		kind: "GitRepository"
		address: "https://code.v1.dodo.cloud/helm-charts"
		branch: "main"
		path: "charts/oauth2-client"
	}
	matrix: {
		kind: "GitRepository"
		address: "https://code.v1.dodo.cloud/helm-charts"
		branch: "main"
		path: "charts/matrix"
	}
	postgres: {
		kind: "GitRepository"
		address: "https://code.v1.dodo.cloud/helm-charts"
		branch: "main"
		path: "charts/postgresql"
	}
}

_oauth2ClientSecretName: "oauth2-client"

helm: {
	"oauth2-client": {
		chart: charts.oauth2Client
		info: "Creating OAuth2 client"
		values: {
			name: "\(release.namespace)-matrix"
			secretName: _oauth2ClientSecretName
			grantTypes: ["authorization_code"]
			responseTypes: ["code"]
			scope: "openid profile"
			redirectUris: ["https://\(_domain)/_synapse/client/oidc/callback"]
			hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
		}
	}
	matrix: {
		dependsOn: [{
			name: "postgres"
			namespace: release.namespace
		}]
		chart: charts.matrix
		info: "Installing Synapse server"
		values: {
			domain: input.network.domain
			subdomain: input.subdomain
			oauth2: {
				secretName: "oauth2-client"
				issuer: "https://hydra.\(input.network.domain)"
			}
			postgresql: {
				host: "postgres"
				port: 5432
				database: "matrix"
				user: "matrix"
				password: "matrix"
			}
			certificateIssuer: input.network.certificateIssuer
			ingressClassName: input.network.ingressClass
			configMerge: {
				configName: "config-to-merge"
				fileName: "to-merge.yaml"
			}
			image: {
				repository: images.matrix.fullName
				tag: images.matrix.tag
				pullPolicy: images.matrix.pullPolicy
			}
		}
	}
	postgres: {
		chart: charts.postgres
		info: "Installing PostgreSQL"
		values: {
			fullnameOverride: "postgres"
			image: {
				registry: images.postgres.registry
				repository: images.postgres.imageName
				tag: images.postgres.tag
				pullPolicy: images.postgres.pullPolicy
			}
			service: {
				type: "ClusterIP"
				port: 5432
			}
			primary: {
				initdb: {
					scripts: {
						"init.sql": """
						CREATE USER matrix WITH PASSWORD 'matrix';
						CREATE DATABASE matrix WITH OWNER = matrix ENCODING = UTF8 LOCALE = 'C' TEMPLATE = template0;
						"""
					}
				}
				persistence: {
					size: "10Gi"
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
			volumePermissions: {
				securityContext: {
					runAsUser: 0
				}
			}
		}
	}
}

help: [{
	title: "Client Applications"
	contents: "You can connect to \(_domain) Matrix server with any of the official clients. We recommend using Element. You can use official Element Web application to chat within the browser. Platform native client applications can be downloaded from: [https://element.io/download](https://element.io/download). Follow **Custom Homeserver** section to login with your dodo: account."
}, {
	title: "Custom Homeserver"
	contents: "Click **Sign in** button, edit **Homeserver** address and enter **\(input.network.domain)**, click **Continue**. Choose **Continue with PCloud** option and login to your dodo: account."
}]
