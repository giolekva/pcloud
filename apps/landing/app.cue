app: {
	type: "hugo:latest"
	ingress: {
		network: "Private"
		subdomain: "landing"
		auth: enabled: false
	}
}
