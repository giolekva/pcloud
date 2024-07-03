import (
	"encoding/base64"
)

input: {
	appName: string
	repoAddr: string
	gitRepoPublicKey: string
	// TODO(gio): auto generate
	fluxKeys: #SSHKey
}

name: "Dodo App Instance"
namespace: "dodo-app-instance"
readme: "Deploy app by pushing to Git repository"
description: "Deploy app by pushing to Git repository"
icon: ""
_domain: "\(input.subdomain).\(input.network.domain)"

resources: {
	"config-kustomization": {
		apiVersion: "kustomize.toolkit.fluxcd.io/v1"
		kind: "Kustomization"
		metadata: {
			name: input.appName
			namespace: release.namespace
		}
		spec: {
			interval: "1m"
			path: "./"
			sourceRef: {
				kind: "GitRepository"
				name: "app"
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
			name: input.appName
			namespace: release.namespace
		}
		data: {
			identity: base64.Encode(null, input.fluxKeys.private)
			"identity.pub": base64.Encode(null, input.fluxKeys.public)
			known_hosts: base64.Encode(null, "soft-serve.\(release.namespace).svc.cluster.local \(input.gitRepoPublicKey)")
		}
	}
	"config-source": {
		apiVersion: "source.toolkit.fluxcd.io/v1"
		kind: "GitRepository"
		metadata: {
			name: input.appName
			namespace: release.namespace
		}
		spec: {
			interval: "1m0s"
			ref: branch: "dodo"
			secretRef: name: input.appName
			timeout: "60s"
			url: input.repoAddr
		}
	}
}
