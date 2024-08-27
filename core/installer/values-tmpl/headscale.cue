input: {
	network: #Network
	subdomain: string
	ipSubnet: string
}

name: "headscale"
namespace: "app-headscale"
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 48 48'><circle cx='24' cy='24' r='4.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='38' cy='24' r='4.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='38' cy='10' r='4.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='24' cy='10' r='4.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='10' cy='10' r='4.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='10' cy='24' r='4.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='10' cy='38' r='4.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='24' cy='38' r='4.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='38' cy='38' r='4.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='24' cy='38' r='2' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='24' cy='24' r='2' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='10' cy='24' r='2' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/><circle cx='38' cy='24' r='2' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/></svg>"

_domain: "\(input.subdomain).\(input.network.domain)"
_oauth2ClientSecretName: "oauth2-client"

out: {
	images: {
		headscale: {
			repository: "headscale"
			name: "headscale"
			tag: "0.22.3"
			pullPolicy: "IfNotPresent"
		}
		api: {
			repository: "giolekva"
			name: "headscale-api"
			tag: "latest"
			pullPolicy: "Always"
		}
	}

	charts: {
		oauth2Client: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/oauth2-client"
		}
		headscale: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/headscale"
		}
	}

	helm: {
		"oauth2-client": {
			chart: charts.oauth2Client
			// TODO(gio): remove once hydra maester is installed as part of dodo itself
			dependsOn: [{
				name: "auth"
				namespace: "\(global.namespacePrefix)core-auth"
			}]
			values: {
				name: "\(release.namespace)-headscale"
				secretName: _oauth2ClientSecretName
				grantTypes: ["authorization_code"]
				responseTypes: ["code"]
				scope: "openid profile email"
				redirectUris: ["https://\(_domain)/oidc/callback"]
				hydraAdmin: "http://hydra-admin.\(global.namespacePrefix)core-auth.svc.cluster.local"
			}
		}
		headscale: {
			chart: charts.headscale
			dependsOn: [{
				name: "auth"
				namespace: "\(global.namespacePrefix)core-auth"
			}]
			values: {
				image: {
					repository: images.headscale.fullName
					tag: images.headscale.tag
					pullPolicy: images.headscale.pullPolicy
				}
				storage: size: "5Gi"
				ingressClassName: input.network.ingressClass
				certificateIssuer: input.network.certificateIssuer
				domain: _domain
				publicBaseDomain: input.network.domain
				ipAddressPool: "\(global.id)-headscale"
				oauth2: {
					secretName: _oauth2ClientSecretName
					issuer: "https://hydra.\(input.network.domain)"
				}
				api: {
					port: 8585
					ipSubnet: input.ipSubnet
					self: "http://headscale-api.\(release.namespace).svc.cluster/sync-users"
					image: {
						repository: images.api.fullName
						tag: images.api.tag
						pullPolicy: images.api.pullPolicy
					}
				}
				ui: enabled: false
			}
		}
	}
}

help: [{
	title: "Install"
	contents: """
	You can install Tailscale client on any of your personal devices running: macOS, iOS, Windows, Lonux or Android. Installer packages can be found at: [https://tailscale.com/download](https://tailscale.com/download). After installing the client application you need to configure it to use https://\(_domain) as a login URL, so you can login to the VPN network with your dodo: account. See "Configure Login URL" section below for more details.
	"""
	children: [{
		title: "Widnows with MSI"
		contents: "[https://tailscale.com/kb/1189/install-windows-msi](https://tailscale.com/kb/1189/install-windows-msi)"
	}]
}, {
	title: "Configure Login URL"
	contents: "After installing the client application you need to configure it to use https://\(_domain) as a login URL, so you can login to the VPN network with your dodo: account"
	children: [{
		title: "macOS"
		contents: "[https://headscale.\(input.network.domain)/apple](https://headscale.\(input.network.domain)/apple)"
	}, {
		title: "iOS"
		contents: "[https://headscale.\(input.network.domain)/apple](https://headscale.\(input.network.domain)/apple)"
	}, {
		title: "Windows"
		contents: "[https://tailscale.com/kb/1318/windows-mdm](https://tailscale.com/kb/1318/windows-mdm)"
	}, {
		title: "Linux"
		contents: "tailscale up --login-server https://\(_domain)"
	}, {
		title: "Android"
		contents: """
		After opening the app, the kebab menu icon (three dots) on the top bar on the right must be repeatedly opened and closed until the Change server option appears in the menu. This is where you can enter your headscale URL: https://\(_domain)

		A screen recording of this process can be seen in the tailscale-android PR which implemented this functionality: [https://github.com/tailscale/tailscale-android/pull/55](https://github.com/tailscale/tailscale-android/pull/55)

		After saving and restarting the app, selecting the regular Sign in option should open up the dodo: authentication page.
		"""
	}, {
		title: "Command Line"
		contents: "tailscale up --login-server https://\(_domain)"
	}]
}]
