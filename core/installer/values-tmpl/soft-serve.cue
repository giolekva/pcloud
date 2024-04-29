input: {
	subdomain: string @name(Subdomain)
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

helm: {
	softserve: {
		chart: charts.softserve
		values: {
			serviceType: "LoadBalancer"
			reservedIP: ""
			addressPool: global.id
			adminKey: input.adminKey
			privateKey: ""
			publicKey: ""
			ingress: {
				enabled: false
			}
			image: {
				repository: images.softserve.fullName
				tag: images.softserve.tag
				pullPolicy: images.softserve.pullPolicy
			}
		}
	}
}
