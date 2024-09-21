input: {
}

name: "longhorn"
namespace: "longhorn"
_pullPolicy: "IfNotPresent"

out: {
	images: {
		longhornEngine: {
			repository: "longhornio"
			name: "longhorn-engine"
			tag: "v1.5.2"
			pullPolicy: _pullPolicy
		}
		longhornManager: {
			repository: "longhornio"
			name: "longhorn-manager"
			tag: "v1.5.2"
			pullPolicy: _pullPolicy
		}
		longhornUI: {
			repository: "longhornio"
			name: "longhorn-ui"
			tag: "v1.5.2"
			pullPolicy: _pullPolicy
		}
		longhornInstanceManager: {
			repository: "longhornio"
			name: "longhorn-instance-manager"
			tag: "v1.5.2"
			pullPolicy: _pullPolicy
		}
		longhornShareManager: {
			repository: "longhornio"
			name: "longhorn-share-manager"
			tag: "v1.5.2"
			pullPolicy: _pullPolicy
		}
		longhornBackingImageManager: {
			repository: "longhornio"
			name: "backing-image-manager"
			tag: "v1.5.2"
			pullPolicy: _pullPolicy
		}
		longhornSupportBundleKit: {
			repository: "longhornio"
			name: "support-bundle-kit"
			tag: "v0.0.27"
			pullPolicy: _pullPolicy
		}
		csiAttacher: {
			repository: "longhornio"
			name: "csi-attacher"
			tag: "v4.2.0"
			pullPolicy: _pullPolicy
		}
		csiProvisioner: {
			repository: "longhornio"
			name: "csi-provisioner"
			tag: "v3.4.1"
			pullPolicy: _pullPolicy
		}
		csiNodeDriverRegistrar: {
			repository: "longhornio"
			name: "csi-node-driver-registrar"
			tag: "v2.7.0"
			pullPolicy: _pullPolicy
		}
		csiResizer: {
			repository: "longhornio"
			name: "csi-resizer"
			tag: "v1.7.0"
			pullPolicy: _pullPolicy
		}
		csiSnapshotter: {
			repository: "longhornio"
			name: "csi-snapshotter"
			tag: "v6.2.1"
			pullPolicy: _pullPolicy
		}
		csiLivenessProbe: {
			repository: "longhornio"
			name: "livenessprobe"
			tag: "v2.9.0"
			pullPolicy: _pullPolicy
		}
	}
	charts: {
		longhorn: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/longhorn"
		}
	}
	helm: {
		longhorn: {
			chart: charts.longhorn
			info: "Installing distributed storage servers"
			values: {
				image: {
					longhorn: {
						engine: {
							repository: images.longhornEngine.imageName
							tag: images.longhornEngine.tag
						}
						manager: {
							repository: images.longhornManager.imageName
							tag: images.longhornManager.tag
						}
						ui: {
							repository: images.longhornUI.imageName
							tag: images.longhornUI.tag
						}
						instanceManager: {
							repository: images.longhornInstanceManager.imageName
							tag: images.longhornInstanceManager.tag
						}
						shareManager: {
							repository: images.longhornShareManager.imageName
							tag: images.longhornShareManager.tag
						}
						backingImageManager: {
							repository: images.longhornBackingImageManager.imageName
							tag: images.longhornBackingImageManager.tag
						}
						supportBundleKit: {
							repository: images.longhornSupportBundleKit.imageName
							tag: images.longhornSupportBundleKit.tag
						}
					}
					csi: {
						attacher: {
							repository: images.csiAttacher.imageName
							tag: images.csiAttacher.tag
						}
						provisioner: {
							repository: images.csiProvisioner.imageName
							tag: images.csiProvisioner.tag
						}
						nodeDriverRegistrar: {
							repository: images.csiNodeDriverRegistrar.imageName
							tag: images.csiNodeDriverRegistrar.tag
						}
						resizer: {
							repository: images.csiResizer.imageName
							tag: images.csiResizer.tag
						}
						snapshotter: {
							repository: images.csiSnapshotter.imageName
							tag: images.csiSnapshotter.tag
						}
						livenessProbe: {
							repository: images.csiLivenessProbe.imageName
							tag: images.csiLivenessProbe.tag
						}
					}
					pullPolicy: _pullPolicy
				}
				// if input.storageDir != _|_ {
				// 	defaultSettings: defaultDataPath: input.storageDir
				// }
				// if input.volumeDefaultReplicaCount != _|_ {
					persistence: defaultClassReplicaCount: 1 // input.volumeDefaultReplicaCount
				// }
				service: ui: type: "ClusterIP"
				ingress: enabled: false
			}
		}
	}
}
