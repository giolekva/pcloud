#Global: {
	pcloudEnvName: string | *""
    publicIP: [...string] | *[]
	namespacePrefix: string | *""
    infraAdminPublicKey: string | *""
}

// TODO(gio): remove
ingressPublic: "\(global.pcloudEnvName)-ingress-public"

ingress: {}
_ingressValidate: {}

