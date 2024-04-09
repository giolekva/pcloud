input: {
	authGroups: string
}

_subdomain: "launcher"
_domain: "\(_subdomain).\(global.privateDomain)"

name: "launcher"
namespace: "core-installer-welcome-launcher"
readme: "App Launcher application will be installed on Private or Public network and be accessible at https://\(_domain)"
description: "The application is a App launcher, designed to run all accessible applications. Can be configured to be reachable only from private network or publicly."
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 24 24'><path fill='currentColor' d='M15.43 15.48c-1.1-.49-2.26-.73-3.43-.73c-1.18 0-2.33.25-3.43.73c-.23.1-.4.29-.49.52h7.85a.978.978 0 0 0-.5-.52m-2.49-6.69C12.86 8.33 12.47 8 12 8s-.86.33-.94.79l-.2 1.21h2.28z' opacity='0.3'/><path fill='currentColor' d='M10.27 12h3.46a1.5 1.5 0 0 0 1.48-1.75l-.3-1.79a2.951 2.951 0 0 0-5.82.01l-.3 1.79c-.15.91.55 1.74 1.48 1.74m.79-3.21c.08-.46.47-.79.94-.79s.86.33.94.79l.2 1.21h-2.28zm-9.4 2.32c-.13.26-.18.57-.1.88c.16.69.76 1.03 1.53 1h1.95c.83 0 1.51-.58 1.51-1.29c0-.14-.03-.27-.07-.4c-.01-.03-.01-.05.01-.08c.09-.16.14-.34.14-.53c0-.31-.14-.6-.36-.82c-.03-.03-.03-.06-.02-.1c.07-.2.07-.43.01-.65a1.12 1.12 0 0 0-.99-.74a.09.09 0 0 1-.07-.03C5.03 8.14 4.72 8 4.37 8c-.3 0-.57.1-.75.26c-.03.03-.06.03-.09.02a1.24 1.24 0 0 0-1.7 1.03c0 .02-.01.04-.03.06c-.29.26-.46.65-.41 1.05c.03.22.12.43.25.6c.03.02.03.06.02.09m14.58 2.54c-1.17-.52-2.61-.9-4.24-.9c-1.63 0-3.07.39-4.24.9A2.988 2.988 0 0 0 6 16.39V18h12v-1.61c0-1.18-.68-2.26-1.76-2.74M8.07 16a.96.96 0 0 1 .49-.52c1.1-.49 2.26-.73 3.43-.73c1.18 0 2.33.25 3.43.73c.23.1.4.29.49.52zm-6.85-1.42A2.01 2.01 0 0 0 0 16.43V18h4.5v-1.61c0-.83.23-1.61.63-2.29c-.37-.06-.74-.1-1.13-.1c-.99 0-1.93.21-2.78.58m21.56 0A6.95 6.95 0 0 0 20 14c-.39 0-.76.04-1.13.1c.4.68.63 1.46.63 2.29V18H24v-1.57c0-.81-.48-1.53-1.22-1.85M22 11v-.5c0-1.1-.9-2-2-2h-2c-.42 0-.65.48-.39.81l.7.63c-.19.31-.31.67-.31 1.06c0 1.1.9 2 2 2s2-.9 2-2'/></svg>"

_httpPortName: "http"

ingress: {
	launcher: {
		auth: {
			enabled: true
			groups: input.authGroups
		}
		network: networks.private
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
        name: "launcher"
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
            logoutUrl: "https://accounts-ui.\(global.domain)/logout"
        }
    }
}
