input: {
	name: string
	from: string
	to: string
	autoAssign: bool | *false
	namespace: string
}

name: "metallb-ipaddresspool"
namespace: "metallb-ipaddresspool"

out: {
	charts: {
		metallbIPAddressPool: {
			kind: "GitRepository"
			address: "https://code.v1.dodo.cloud/helm-charts"
			branch: "main"
			path: "charts/metallb-ipaddresspool"
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
}
