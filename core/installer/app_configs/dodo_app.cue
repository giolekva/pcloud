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

	_network: networks[strings.ToLower(network)]
	baseURL: "https://\(subdomain).\(_network.domain)"
}

#Volumes: {
	...
}

app: {
	volumes: {
		for key, value in volumes {
			"\(key)": #volume & value & {
				name: key
			}
		}
	}
}

#Command: {
	bin: string
	args: [...string] | *[]
	env: [...string] | *[]
}

// Go app

_goVer1220: "golang:1.22.0"
_goVer1200: "golang:1.20.0"

#GoAppTmpl: {
	type: _goVer1220 | _goVer1200
	run: string
	ingress: #AppIngress
	volumes: #Volumes

	runConfiguration: [{
		bin: "/usr/local/go/bin/go",
		args: ["mod", "tidy"]
	}, {
		bin: "/usr/local/go/bin/go",
		args: ["build", "-o", ".app", run]
	}, {
		bin: ".app",
		args: [],
		env: [
			for k, v in volumes {
				"DODO_VOLUME_\(strings.ToUpper(k))=/dodo-volume/\(v.name)"
			}
	    ]
	}]
}

#GoApp1200: #GoAppTmpl & {
	type: _goVer1200
}

#GoApp1220: #GoAppTmpl & {
	type: _goVer1220
}

#GoApp: #GoApp1200 | #GoApp1220

// Hugo app

_hugoLatest: "hugo:latest"

#HugoAppTmpl: {
	type: _hugoLatest
	ingress: #AppIngress
	volumes: {}

	runConfiguration: [{
		bin: "/usr/bin/hugo",
		args: []
	}, {
		bin: "/usr/bin/hugo",
		args: [
			"server",
			"--watch=false",
			"--bind=0.0.0.0",
			"--port=\(_appPort)",
			"--baseURL=\(ingress.baseURL)",
			"--appendPort=false",
    	]
	}]
}

#HugoApp: #HugoAppTmpl

#App: #GoApp | #HugoApp

app: #App

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
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/app-runner"
	}
}

volumes: app.volumes

helm: {
	app: {
		chart: charts.app
		values: {
			image: {
				repository: images.app.fullName
				tag: images.app.tag
				pullPolicy: images.app.pullPolicy
			}
			appPort: _appPort
			appDir: _appDir
			repoAddr: input.repoAddr
			sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
			runCfg: base64.Encode(null, json.Marshal(_app.runConfiguration))
			manager: "http://dodo-app.\(release.namespace).svc.cluster.local/register-worker"
			volumes: [
				for key, value in _app.volumes {
					name: value.name
					mountPath: "/dodo-volume/\(key)"
				}
            ]
		}
	}
}

_appDir: "/dodo-app"
_appPort: 8080
