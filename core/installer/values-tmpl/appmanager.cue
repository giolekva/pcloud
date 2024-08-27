import (
	"encoding/base64"
)

input: {
	network: #Network @name(Network)
	repoAddr: string @name(Repository Address)
	sshPrivateKey: string @name(SSH Private Key)
	authGroups: string @name(Allowed Groups)
}

name: "App Manager"
namespace: "appmanager"
icon: """
<svg width='50px' height='50px' xmlns='http://www.w3.org/2000/svg' viewBox='0 0 33.66287237 39.68503937'>
  <defs>
    <style>
      .cls-1 {
        fill: currentColor;
      }
    </style>
  </defs>
  <path class='cls-1' d='m2.77885812,10.03744798c-1.53217449,0-2.77885812,1.24698542-2.77885812,2.77885812v24.08987515c0,1.5318727,1.24668363,2.77885812,2.77885812,2.77885812h28.10515613c1.53217449,0,2.77885812-1.24698542,2.77885812-2.77885812V12.8163061c0-1.5318727-1.24668363-2.77885812-2.77885812-2.77885812h-5.25110027v-1.23612107c0-4.9348271-3.86589623-8.80132691-8.80132691-8.80132691s-8.80162869,3.86649981-8.80162869,8.80132691v1.23612107H2.77885812Zm29.34127721,28.10485435H1.54273704V11.58018502h30.57739828v26.5621173ZM8.11234634,16.05991677c.58426035,4.33004521,4.20419987,7.56520583,8.71924074,7.56520583,4.51473908,0,8.1346786-3.23516062,8.71893895-7.56520583h-1.56084429c-.57067992,3.46210473-3.51008892,6.02246879-7.15809467,6.02246879s-6.58771654-2.56036406-7.15839646-6.02246879h-1.56084429Zm15.9778306-6.02246879h-14.51748151v-1.23612107c0-4.07050807,3.18838358-7.25858986,7.25889165-7.25858986,4.07020628,0,7.25858986,3.18808179,7.25858986,7.25858986v1.23612107Z'/>
</svg>"""

_subdomain: "apps"
_httpPortName: "http"

_domain: "\(_subdomain).\(input.network.domain)"
url: "https://\(_domain)"

out: {
	ingress: {
		appmanager: {
			auth: {
				enabled: true
				groups: input.authGroups
			}
			network: input.network
			subdomain: _subdomain
			service: {
				name: "appmanager"
				port: name: _httpPortName
			}
		}
	}

	images: {
		appmanager: {
			repository: "giolekva"
			name: "pcloud-installer"
			tag: "latest"
			pullPolicy: "Always"
		}
	}

	charts: {
		appmanager: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/appmanager"
		}
	}

	helm: {
		appmanager: {
			chart: charts.appmanager
			values: {
				repoAddr: input.repoAddr
				sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
				headscaleAPIAddr: "http://headscale-api.\(global.namespacePrefix)app-headscale.svc.cluster.local"
				ingress: {
					className: input.network.ingressClass
					domain: _domain
					certificateIssuer: ""
				}
				clusterRoleName: "\(global.id)-appmanager"
				portName: _httpPortName
				image: {
					repository: images.appmanager.fullName
					tag: images.appmanager.tag
					pullPolicy: images.appmanager.pullPolicy
				}
			}
		}
	}
}
