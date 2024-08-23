import (
	"strings"
)

input: {
	name: string
	domain: string
}

name: "Network"
namespace: "ingress-custom"
readme: "Configure custom public domain"
description: readme
icon: """
<svg width='50px' height='50px' xmlns='http://www.w3.org/2000/svg' viewBox='0 0 39.68503937 35.84897203'>
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
  <rect class='cls-2' x='-9.97439025' y='-11.89242392' width='59.63381987' height='59.63381987'/>
  <g>
    <path class='cls-1' d='m8.33392163,35.84897203H.66139092c-.36534918,0-.66139092-.29623977-.66139092-.66139092v-7.67213467c0-.36515115.29604174-.66139092.66139092-.66139092h7.67253071c.36534918,0,.66139092.29623977.66139092.66139092v7.67213467c0,.36515115-.29604174.66139092-.66139092.66139092Zm-7.01113979-1.32278184h6.34974887v-6.34935283H1.32278184v6.34935283Zm18.51973785-6.34935283c-.36534918,0-.66139092-.29623977-.66139092-.66139092v-14.68367051H4.49765628c-.36534918,0-.66139092-.29623977-.66139092-.66139092V.66139092c0-.36515115.29604174-.66139092.66139092-.66139092h30.68972682c.36534918,0,.66139092.29623977.66139092.66139092v11.50899409c0,.36515115-.29604174.66139092-.66139092.66139092h-14.68347249v14.68367051c0,.36515115-.29604174.66139092-.66139092.66139092Zm0-16.66784327h14.68347249V1.32278184H5.1590472v10.18621225h14.68347249Z'/>
    <path class='cls-1' d='m39.02364845,35.84897203h-7.67253071c-.36534918,0-.66139092-.29623977-.66139092-.66139092v-7.67213467c0-.36515115.29604174-.66139092.66139092-.66139092h3.17487444v-6.35014492H5.1590472v7.01153584c0,.36515115-.29604174.66139092-.66139092.66139092s-.66139092-.29623977-.66139092-.66139092v-7.67292676c0-.36515115.29604174-.66139092.66139092-.66139092h30.68972682c.36534918,0,.66139092.29623977.66139092.66139092v7.01153584h3.17487444c.36534918,0,.66139092.29623977.66139092.66139092v7.67213467c0,.36515115-.29604174.66139092-.66139092.66139092Zm-7.01113979-1.32278184h6.34974887v-6.34935283h-6.34974887v6.34935283Zm-8.33372361,1.32278184h-7.67253071c-.36534918,0-.66139092-.29623977-.66139092-.66139092v-7.67213467c0-.36515115.29604174-.66139092.66139092-.66139092h7.67253071c.36534918,0,.66139092.29623977.66139092.66139092v7.67213467c0,.36515115-.29604174.66139092-.66139092.66139092Zm-7.01113979-1.32278184h6.34974887v-6.34935283h-6.34974887v6.34935283ZM12.16998897,7.07727889h-1.91803367c-.36534918,0-.66139092-.29623977-.66139092-.66139092s.29604174-.66139092.66139092-.66139092h1.91803367c.36534918,0,.66139092.29623977.66139092.66139092s-.29604174.66139092-.66139092.66139092Z'/>
  </g>
</svg>"""

out: {
	charts: {
		"certificate-issuer": {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/certificate-issuer-public"
		}
	}

	helm: {
		"certificate-issuer": {
			chart: charts["certificate-issuer"]
			Info: "Configuring SSL certificate issuer for \(input.domain)"
			dependsOn: [{
				name: "ingress-nginx"
				namespace: "\(global.namespacePrefix)ingress-private"
			}]
			values: {
				issuer: {
					name: input.name
					server: "https://acme-v02.api.letsencrypt.org/directory"
					domain: input.domain
					contactEmail: global.contactEmail
					ingressClass: networks.public.ingressClass
				}
			}
		}
	}
}

help: [{
	title: "DNS"
	_records: [for ip in global.nameserverIP { "* 10800 IN A \(ip)" }]
	_allRecords: strings.Join(_records, "<br>")
	contents: """
	Publish following DNS records using \(input.domain) Domain Name Registrar<br><br>
	\(_allRecords)
	"""
}]
