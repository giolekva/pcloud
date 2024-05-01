input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
	sshPort: int @name(SSH Port)
	adminKey: string @name(Admin SSH Public Key)
}

_domain: "\(input.subdomain).\(global.privateDomain)"

name: "Soft-Serve"
namespace: "app-soft-serve"
// TODO(gio): make public network an option
readme: "softserve application will be installed on private network and be accessible to any user on https://\(_domain)"
description: "A tasty, self-hostable Git server for the command line. 🍦"
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 48 48'><g fill='none' stroke='currentColor' stroke-linecap='round' stroke-width='4'><path stroke-linejoin='round' d='M15.34 22.5L21 37l3 6l3-6l5.66-14.5'/><path d='M19 32h10'/><path stroke-linejoin='round' d='M24 3c-6 0-8 6-8 6s-6 2-6 7s5 7 5 7s3.5-2 9-2s9 2 9 2s5-2 5-7s-6-7-6-7s-2-6-8-6Z'/></g></svg>"

images: {
	softserve: {
		repository: "charmcli"
		name: "soft-serve"
		tag: "v0.7.1"
		pullPolicy: "IfNotPresent"
	}
}

charts: {
	softserve: {
		chart: "charts/soft-serve"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.id
		}
	}
}

ingress: {
	gerrit: {
		auth: enabled: false
		network: input.network
		subdomain: input.subdomain
		service: {
			name: "soft-serve"
			port: number: 80
		}
	}
}

portForward: [#PortForward & {
	allocator: input.network.allocatePortAddr
	sourcePort: input.sshPort
	// TODO(gio): namespace part must be populated by app manager. Otherwise
	// third-party app developer might point to a service from different namespace.
	targetService: "\(release.namespace)/soft-serve"
	targetPort: 22
}]

helm: {
	softserve: {
		chart: charts.softserve
		values: {
			serviceType: "ClusterIP"
			adminKey: input.adminKey
			sshPublicPort: input.sshPort
			ingress: {
				enabled: false
				domain: _domain
			}
			image: {
				repository: images.softserve.fullName
				tag: images.softserve.tag
				pullPolicy: images.softserve.pullPolicy
			}
		}
	}
}

help: [{
	title: "Access"
	contents: """
	SSH CLI: ssh \(_domain) -p \(input.sshPort) help  
	SSH TUI: ssh \(_domain) -p \(input.sshPort)  
	HTTP: git clone https://\(_domain)/<REPO-NAME>  
	SSH: git clone ssh://\(_domain):\(input.sshPort)/<REPO-NAME>  

	See following resource on what you can do with Soft-Serve TUI: [https://github.com/charmbracelet/soft-serve](https://github.com/charmbracelet/soft-serve)
	"""
}]
