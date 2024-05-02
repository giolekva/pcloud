import (
	"encoding/base64"
)

input: {
    repoAddr: string
	sshPrivateKey: string
}

_subdomain: "launcher"
_domain: "\(_subdomain).\(networks.public.domain)"

name: "Launcher"
namespace: "launcher"
readme: "App Launcher application will be installed on Private or Public network and be accessible at https://\(_domain)"
description: "The application is a App launcher, designed to run all accessible applications. Can be configured to be reachable only from private network or publicly."
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 48 48'><path fill='none' stroke='black' stroke-linecap='round' stroke-linejoin='round' d='M42.5 23.075L26.062 7.525a3 3 0 0 0-4.124 0L5.5 23.075m5.86 1.54v14.68a2 2 0 0 0 2 2h7.14v-9.5h7v9.5h7.14a2 2 0 0 0 2-2v-14.68'/></svg>"

_httpPortName: "http"

ingress: {
	launcher: {
		auth: enabled: true
		network: networks.public
		subdomain: _subdomain
		service: {
			name: "launcher"
			port: name: _httpPortName
		}
	}
}

images: {
    launcher: {
        repository: "giolekva"
        name: "pcloud-installer"
        tag: "latest"
        pullPolicy: "Always"
    }
}

charts: {
    launcher: {
        chart: "charts/launcher"
        sourceRef: {
            kind: "GitRepository"
            name: "pcloud"
            namespace: global.id
        }
    }
}

helm: {
    launcher: {
        chart: charts.launcher
        values: {
            image: {
                repository: images.launcher.fullName
                tag: images.launcher.tag
                pullPolicy: images.launcher.pullPolicy
            }
            portName: _httpPortName
            repoAddr: input.repoAddr
            sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
            logoutUrl: "https://accounts-ui.\(global.domain)/logout"
			repoAddr: input.repoAddr
			sshPrivateKey: base64.Encode(null, input.sshPrivateKey)
        }
    }
}
