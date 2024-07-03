import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
	sshPort: int @name(SSH Port) @role(port)
	adminKey: string @name(Admin SSH Public Key)

	// TODO(gio): auto generate
	ssKeys: #SSHKey
	fluxKeys: #SSHKey
	dAppKeys: #SSHKey
}

name: "Dodo App"
namespace: "dodo-app"
readme: "Deploy app by pushing to Git repository"
description: "Deploy app by pushing to Git repository"
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 48 48'><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M2.837 27.257c3.363 2.45 11.566 3.523 12.546 1.4s.424-10.94.424-10.94s-1.763 1.192-2.302.147s.44-2.433 2.319-2.858c-1.96.05-2.221-.571-2.205-.93s.67-1.878 3.527-1.241c-1.6-.751-1.943-2.956 2.352-1.568c-1.421-.735-.36-2.825 1.649-.62c-.261-1.323 1.584-1.46 2.694.907M10.648 34.633a19 19 0 0 0-4.246.719'/><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M15.144 43.402c3.625-2.482 7.685-6.32 7.293-13.406s-1.6-6.368-.523-7.577s6.924-.99 10.712 3.353c.032-2.874-2.504-5.508-2.504-5.508a33 33 0 0 1 5.53.163c2.852.49 2.394 2.514 3.58 2.035s.971-3.472-.39-5.377c-1.666-2.33-3.223-2.83-6.358-2.188s-4.474.458-5.54-.587s-2.026-3.538-4.605-2.515c-2.935 1.164-4.398 2.438-3.767 5.04s2.34 4.558 2.972 6.844'/><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M22.001 16.552c-.925-.043-1.894.055-1.709 1.328'/><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M20.662 16.763c1.72 2.695 3.405 3.643 9.46 3.501'/><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M32.14 14.966c-1.223.879-2.18 3.781-2.496 5.307M23.1 14.908c.48 1.209 1.23.728 1.315.283a1.552 1.552 0 0 0-1.543-1.883m-.408 17.472c5.328 2.71 11.631.229 16.269-2.123c-1.176 4.572-5.911 5.585-8.916 6.107'/><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M29.099 37.115c4.376-.294 8.024-1.578 7.833-5.296'/><path fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' d='M20.27 38.702c6.771 3.834 12.505.798 13.786-2.615'/><circle cx='24' cy='24' r='21.5' fill='none' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round'/></svg>"
_domain: "\(input.subdomain).\(input.network.domain)"

images: {
	softserve: {
		repository: "charmcli"
		name: "soft-serve"
		tag: "v0.7.1"
		pullPolicy: "IfNotPresent"
	}
	dodoApp: {
		repository: "giolekva"
		name: "pcloud-installer"
		tag: "latest"
		pullPolicy: "Always"
	}
}

charts: {
	softserve: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/soft-serve"
	}
	dodoApp: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/dodo-app"
	}
}

portForward: [#PortForward & {
	allocator: input.network.allocatePortAddr
	reservator: input.network.reservePortAddr
	deallocator: input.network.deallocatePortAddr
	sourcePort: input.sshPort
	serviceName: "soft-serve"
	targetPort: 22
}]

helm: {
	softserve: {
		chart: charts.softserve
		info: "Installing Git server"
		values: {
			serviceType: "ClusterIP"
			addressPool: ""
			reservedIP: ""
			adminKey: strings.Join([input.fluxKeys.public, input.dAppKeys.public], "\n")
			privateKey: input.ssKeys.private
			publicKey: input.ssKeys.public
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
	"dodo-app": {
		chart: charts.dodoApp
		info: "Installing supervisor"
		values: {
			image: {
				repository: images.dodoApp.fullName
				tag: images.dodoApp.tag
				pullPolicy: images.dodoApp.pullPolicy
			}
			repoAddr: "soft-serve.\(release.namespace).svc.cluster.local:22"
			sshPrivateKey: base64.Encode(null, input.dAppKeys.private)
			self: "dodo-app.\(release.namespace).svc.cluster.local"
			namespace: release.namespace
			envConfig: base64.Encode(null, json.Marshal(global))
			appAdminKey: input.adminKey
			gitRepoPublicKey: input.ssKeys.public
		}
	}
}

resources: {
	"config-kustomization": {
		apiVersion: "kustomize.toolkit.fluxcd.io/v1"
		kind: "Kustomization"
		metadata: {
			name: "config"
			namespace: release.namespace
		}
		spec: {
			interval: "1m"
			path: "./"
			sourceRef: {
				kind: "GitRepository"
				name: "config"
				namespace: release.namespace
			}
			prune: true
		}
	}
	"config-secret": {
		apiVersion: "v1"
		kind: "Secret"
		type: "Opaque"
		metadata: {
			name: "config"
			namespace: release.namespace
		}
		data: {
			identity: base64.Encode(null, input.fluxKeys.private)
			"identity.pub": base64.Encode(null, input.fluxKeys.public)
			known_hosts: base64.Encode(null, "soft-serve.\(release.namespace).svc.cluster.local \(input.ssKeys.public)")
		}
	}
	"config-source": {
		apiVersion: "source.toolkit.fluxcd.io/v1"
		kind: "GitRepository"
		metadata: {
			name: "config"
			namespace: release.namespace
		}
		spec: {
			interval: "1m0s"
			ref: branch: "master"
			secretRef: name: "config"
			timeout: "60s"
			url: "ssh://soft-serve.\(release.namespace).svc.cluster.local:22/config"
		}
	}
}

help: [{
	title: "How to use"
	contents: """
	Clone: git clone ssh://\(_domain):\(input.sshPort)/app  <button onClick='copyToClipboard(this, "git clone ssh://\(_domain):\(input.sshPort)/app")'><svg width='24px' height='24px' viewBox='-2.4 -2.4 28.80 28.80' fill='none' xmlns='http://www.w3.org/2000/svg'><g id='SVGRepo_bgCarrier' stroke-width='0'></g><g id='SVGRepo_tracerCarrier' stroke-linecap='round' stroke-linejoin='round'></g><g id='SVGRepo_iconCarrier'> <path fill-rule='evenodd' clip-rule='evenodd' d='M19.5 16.5L19.5 4.5L18.75 3.75H9L8.25 4.5L8.25 7.5L5.25 7.5L4.5 8.25V20.25L5.25 21H15L15.75 20.25V17.25H18.75L19.5 16.5ZM15.75 15.75L15.75 8.25L15 7.5L9.75 7.5V5.25L18 5.25V15.75H15.75ZM6 9L14.25 9L14.25 19.5L6 19.5L6 9Z' fill='#080341'></path> </g></svg></button>  
	Server public key: \(input.ssKeys.public)
	"""
}]
