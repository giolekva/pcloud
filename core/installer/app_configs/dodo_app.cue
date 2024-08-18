import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

input: {
	repoAddr: string
	managerAddr: string
	appId: string
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

#PostgreSQLs: {
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
	postgresql: {
		for key, value in postgresql {
			"\(key)": #PostgreSQL & value & {
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
	run: string | *"main.go"
	ingress: #AppIngress
	volumes: #Volumes
	postgresql: #PostgreSQLs
	port: int | *8080
	rootDir: _appDir

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
			for k, v in postgresql {
				"DODO_POSTGRESQL_\(strings.ToUpper(k))_ADDRESS=\(v.name).\(release.namespace).svc.cluster.local"
			}
			for k, v in postgresql {
				"DODO_POSTGRESQL_\(strings.ToUpper(k))_USERNAME=postgres"
			}
			for k, v in postgresql {
				"DODO_POSTGRESQL_\(strings.ToUpper(k))_PASSWORD=postgres"
			}
			for k, v in postgresql {
				"DODO_POSTGRESQL_\(strings.ToUpper(k))_DATABASE=postgres"
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
	postgresql: {}
	port: int | *8080
	rootDir: _appDir

	runConfiguration: [{
		bin: "/usr/bin/hugo",
		args: []
	}, {
		bin: "/usr/bin/hugo",
		args: [
			"server",
			"--watch=false",
			"--bind=0.0.0.0",
			"--port=\(port)",
			"--baseURL=\(ingress.baseURL)",
			"--appendPort=false",
    	]
	}]
}

#HugoApp: #HugoAppTmpl

// PHP app

#PHPAppTmpl: {
	type: "php:8.2-apache"
	ingress: #AppIngress
	volumes: {}
	postgresql: {}
	port: int | *80
	rootDir: "/var/www/html"

	runConfiguration: [{
		bin: "/usr/local/bin/apache2-foreground",
		env: [
			for k, v in volumes {
				"DODO_VOLUME_\(strings.ToUpper(k))=/dodo-volume/\(v.name)"
			}
	    ]
	}]
}

#PHPApp: #PHPAppTmpl

#App: #GoApp | #HugoApp | #PHPApp

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
		address: "https://code.v1.dodo.cloud/helm-charts"
		branch: "main"
		path: "charts/app-runner"
	}
}

volumes: app.volumes
postgresql: app.postgresql

helm: {
	app: {
		chart: charts.app
		values: {
			image: {
				repository: images.app.fullName
				tag: images.app.tag
				pullPolicy: images.app.pullPolicy
			}
			runtimeClassName: "untrusted-external" // TODO(gio): make this part of the infra config
			appPort: _app.port
			appDir: _app.rootDir
			appId: input.appId
			repoAddr: input.repoAddr
			sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
			runCfg: base64.Encode(null, json.Marshal(_app.runConfiguration))
			managerAddr: input.managerAddr
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
