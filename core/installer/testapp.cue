app: {
	type: "golang:1.22.0"
	run: "main.go"
	ingress: {
		network: "private"
		subdomain: "testapp"
		auth: enabled: false
	}
}

// do create app --type=go[1.22.0] [--run-cmd=(*default main.go)]
// do create ingress --subdomain=testapp [--network=public (*default private)] [--auth] [--auth-groups="admin" (*default empty)] TODO(gio): port
