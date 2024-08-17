input: {}

name: "csi-driver-smb"
namespace: "csi-driver-smb"

_baseImage: {
	registry: "registry.k8s.io"
	repository: "sig-storage"
	pullPolicy: "IfNotPresent"
}

images: {
	smb: _baseImage & {
		name: "smbplugin"
		tag: "v1.11.0"
	}
	csiProvisioner: _baseImage & {
		name: "csi-provisioner"
		tag: "v3.5.0"
	}
	livenessProbe: _baseImage & {
		name: "livenessprobe"
		tag: "v2.10.0"
	}
	nodeDriverRegistrar: _baseImage & {
		name: "csi-node-driver-registrar"
		tag: "v2.8.0"
	}
}

charts: {
	csiDriverSMB: {
		kind: "GitRepository"
		address: "https://code.v1.dodo.cloud/helm-charts"
		branch: "main"
		path: "charts/csi-driver-smb"
	}
}

helm: {
	"csi-driver-smb": {
		chart: charts.csiDriverSMB
		values: {
			image: {
				smb: {
					repository: images.smb.fullName
					tag: images.smb.tag
					pullPolicy: images.smb.pullPolicy
				}
				csiProvisioner: {
					repository: images.csiProvisioner.fullName
					tag: images.csiProvisioner.tag
					pullPolicy: images.csiProvisioner.pullPolicy
				}
				livenessProbe: {
					repository: images.livenessProbe.fullName
					tag: images.livenessProbe.tag
					pullPolicy: images.livenessProbe.pullPolicy
				}
				nodeDriverRegistrar: {
					repository: images.nodeDriverRegistrar.fullName
					tag: images.nodeDriverRegistrar.tag
					pullPolicy: images.nodeDriverRegistrar.pullPolicy
				}
			}
		}
	}
}
