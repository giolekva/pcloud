input: {
    network: #Network @name(Network)
    subdomain: string @name(Subdomain)
	auth: #Auth @name(Authentication)
}

_domain: "\(input.subdomain).\(input.network.domain)"
url: "https://\(_domain)"

name: "URL Shortener"
namespace: "app-url-shortener"
readme: "URL shortener application will be installed on \(input.network.name) network and be accessible at https://\(_domain)"
description: "Provides URL shortening service. Can be configured to be reachable only from private network or publicly."
icon: """
<svg width='50px' height='50px' xmlns='http://www.w3.org/2000/svg' viewBox='0 0 36.43807123 39.68503937'>
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
  <rect class='cls-2' x='-11.59787432' y='-9.97439025' width='59.63381987' height='59.63381987'/>
  <g>
    <circle class='cls-1' cx='3.31578123' cy='26.42191445' r='3.31578123'/>
    <circle class='cls-1' cx='3.31578123' cy='13.15878954' r='3.31578123'/>
    <path class='cls-1' d='m11.6052343,39.68503937c-.19894668,0-.36473638,0-.56368306-.09947255-.85909349-.31306169-1.30368349-1.26152142-.994735-2.12210113L23.30994116.98987217c.37099907-.83910142,1.3519786-1.21857512,2.19107766-.84757764.75807463.33517241,1.1530841,1.1778629.92575531,1.97494297l-13.26312492,36.47359431c-.23210526.66315561-.89526087,1.09420755-1.55841648,1.09420755m9.94734369,0c-.19894668,0-.36473638,0-.56368306-.09947255-.85909349-.31306169-1.3036827-1.26152142-.994735-2.12210113L33.25728168.98987217c.37099907-.83910142,1.3519786-1.21857512,2.19107766-.84757764.75807463.33517241,1.1530841,1.1778629.92575531,1.97494297l-13.26312492,36.47359431c-.23210526.66315561-.89526087,1.09420755-1.55841648,1.09420755'/>
  </g>
</svg>"""

_httpPortName: "http"

ingress: {
	"url-shorteners": {
		auth: input.auth
		network: input.network
		subdomain: input.subdomain
		service: {
			name: "url-shortener"
			port: name: _httpPortName
		}
	}
}

images: {
	urlShortener: {
		repository: "giolekva"
		name: "url-shortener"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
    urlShortener: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/url-shortener"
    }
}

helm: {
    "url-shortener": {
        chart: charts.urlShortener
		info: "Installing server"
        values: {
            storage: {
                size: "1Gi"
            }
            image: {
				repository: images.urlShortener.fullName
				tag: images.urlShortener.tag
				pullPolicy: images.urlShortener.pullPolicy
			}
            portName: _httpPortName
			requireAuth: input.auth.enabled
        }
    }
}
