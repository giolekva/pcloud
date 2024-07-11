import (
	"encoding/base64"
)

input: {
	repoAddr: string
	repoHost: string
	gitRepoPublicKey: string
	// TODO(gio): auto generate
	fluxKeys: #SSHKey
}

name: "Dodo App Instance"
namespace: "dodo-app-instance"
readme: "Deploy app by pushing to Git repository"
description: "Deploy app by pushing to Git repository"
icon: ""

resources: {
	"config-kustomization": {
		apiVersion: "kustomize.toolkit.fluxcd.io/v1"
		kind: "Kustomization"
		metadata: {
			name: "app"
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
			name: "app"
			namespace: release.namespace
		}
		data: {
			identity: base64.Encode(null, input.fluxKeys.private)
			"identity.pub": base64.Encode(null, input.fluxKeys.public)
			known_hosts: base64.Encode(null, "\(input.repoHost) \(input.gitRepoPublicKey)")
		}
	}
	"config-source": {
		apiVersion: "source.toolkit.fluxcd.io/v1"
		kind: "GitRepository"
		metadata: {
			name: "app"
			namespace: release.namespace
		}
		spec: {
			interval: "1m0s"
			ref: branch: "dodo"
			secretRef: name: "app"
			timeout: "60s"
			url: input.repoAddr
		}
	}
}
