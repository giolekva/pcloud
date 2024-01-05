input: {
	name: string
	from: string
	to: string
	autoAssign: bool | *false
	namespace: string
}

images: {}

charts: {
	metallbIPAddressPool: {
		chart: "charts/metallb-ipaddresspool"
		sourceRef: {
			kind: "GitRepository"
			name: "pcloud"
			namespace: global.pcloudEnvName // TODO(gio): id ?
		}
	}
}

helm: {
	"metallb-ipaddresspool-\(input.name)": {
		chart: charts.metallbIPAddressPool
		values: {
			name: input.name
			from: input.from
			to: input.to
			autoAssign: input.autoAssign
			namespace: input.namespace
		}
	}
}
