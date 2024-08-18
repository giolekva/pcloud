#Global: {
	pcloudEnvName: string | *""
    publicIP: [...string] | *[]
	namespacePrefix: string | *""
    infraAdminPublicKey: string | *""
}

#Network: {
	name: string
	ingressClass: string
	certificateIssuer: string | *""
	allocatePortAddr: string
	reservePortAddr: string
	deallocatePortAddr: string
}

#Networks: {
	public: #Network
}

networks: #Networks
