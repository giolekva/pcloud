import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

input: {
	repoAddr: string
	sshPrivateKey: string
}

#AppIngress: {
	network: string
	subdomain: string
	auth: #Auth
}

_goVer1220: "golang:1.22.0"
_goVer1200: "golang:1.20.0"

#GoAppTmpl: {
	type: _goVer1220 | _goVer1200
	run: string
	ingress: #AppIngress

	runConfiguration: [{
		bin: "/usr/local/go/bin/go",
		args: ["mod", "tidy"]
	}, {
		bin: "/usr/local/go/bin/go",
		args: ["build", "-o", ".app", run]
	}, {
		bin: ".app",
		args: []
	}]
}

#GoApp1200: #GoAppTmpl & {
	type: _goVer1200
}

#GoApp1220: #GoAppTmpl & {
	type: _goVer1220
}

#GoApp: #GoApp1200 | #GoApp1220

app: #GoApp

// output

_app: app
ingress: {
	app: {
		network: networks[strings.ToLower(_app.ingress.network)]
		subdomain: _app.ingress.subdomain
		auth: _app.ingress.auth
		service: {
			name: "app-app"
			port: name: "app"
		}
	}
}

images: {
	app: {
		repository: "giolekva"
		name: "app-runner"
		tag: strings.Replace(_app.type, ":", "-", -1)
		pullPolicy: "Always"
	}
}

charts: {
	app: {
		chart: "charts/app-runner"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

helm: {
	app: {
		chart: charts.app
		values: {
			image: {
				repository: images.app.fullName
				tag: images.app.tag
				pullPolicy: images.app.pullPolicy
			}
			appPort: 8080
			appDir: "/dodo-app"
			repoAddr: input.repoAddr
			sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
			runCfg: base64.Encode(null, json.Marshal(_app.runConfiguration))
			manager: "http://dodo-app.\(release.namespace).svc.cluster.local/register-worker"
		}
	}
}
