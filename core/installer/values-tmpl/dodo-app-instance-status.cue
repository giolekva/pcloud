input: {
	appName: string
	network: #Network
	appSubdomain: string
}

name: "Dodo App Instance Status"

_subdomain: "status.\(input.appSubdomain)"

out: {
	ingress: {
		"status-\(input.appName)": {
			auth: enabled: false
			network: input.network
			subdomain: _subdomain
			appRoot: "/\(input.appName)"
			service: {
				name: "web"
				port: name: "http"
			}
		}
	}
}
