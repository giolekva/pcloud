input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
	sshPort: int @name(SSH Port) @role(port)
	adminKey: string @name(Admin SSH Public Key)
}

_domain: "\(input.subdomain).\(input.network.domain)"

name: "Soft-Serve"
namespace: "app-soft-serve"
// TODO(gio): make public network an option
readme: "softserve application will be installed on private network and be accessible to any user on https://\(_domain)"
description: "A tasty, self-hostable Git server for the command line. 🍦"
icon: """
<svg width='50px' height='50px' xmlns='http://www.w3.org/2000/svg' viewBox='0 0 28.17637795 39.68590646'>
  <defs>
    <style>
      .cls-1 {
        fill: currentColor;
      }

      .cls-2 {
        fill: none;
        stroke: #3a3a3a;
        stroke-miterlimit: 10;
        stroke-width: .98133445px;
      }
    </style>
  </defs>
  <rect class='cls-2' x='-15.72872096' y='-9.97395671' width='59.63381987' height='59.63381987'/>
  <g>
    <path class='cls-1' d='m14.08828766,39.68590646c-.24966985,0-.47802402-.14131246-.58973401-.36472824l-2.87761777-5.75512747-5.45346067-13.9623024c-.13243358-.33946568.03513141-.72156194.37420873-.85419039.33927469-.13262845.72157548.03473602.85381169.3742017l5.42918447,13.90783,2.26321282,4.52199865,2.28827849-4.57647105,5.40471091-13.8533576c.13243358-.33788677.51374753-.5060407.85381169-.3742017.33907732.13262845.50664231.51472471.37420873.85419039l-5.42898711,13.90783-2.90209134,5.80959987c-.11151262.22341579-.33986679.36472824-.58953664.36472824Z'/>
    <path class='cls-1' d='m18.88431728,29.13483942h-9.59205924c-.36414299,0-.65920736-.2952562-.65920736-.65919498s.29506437-.65919498.65920736-.65919498h9.59205924c.36414299,0,.65920736.2952562.65920736.65919498s-.29506437.65919498-.65920736.65919498Z'/>
    <path class='cls-1' d='m5.45484225,20.50214821c-.08269697,0-.16559131-.0157891-.24414357-.0473673-.21276214-.08447169-5.21069868-2.14337052-5.21069868-7.32614307,0-4.61041762,4.65846449-6.74115686,5.90878744-7.230619.52262907-1.25207574,2.82373645-5.89801884,8.17950022-5.89801884s7.65687115,4.6459431,8.17950022,5.89801884c1.25032295.48946214,5.90859007,2.62099084,5.90859007,7.230619,0,5.18277255-4.99773918,7.24167137-5.21050131,7.32614307-.18572279.07499823-.39670862.05999859-.57078673-.0386833-.03236827-.01894692-3.28912895-1.83232522-8.30680224-1.83232522s-8.27443398,1.8133783-8.30680224,1.83232522c-.10065741.05684077-.21335424.0860506-.32664317.0860506ZM14.08828766,1.31838997c-5.19806716,0-6.97555863,5.08882739-7.0485846,5.30592754-.06592074.1949954-.22045947.3497286-.41585327.41525337-.05309185.01736801-5.30543507,1.83311468-5.30543507,6.08906697,0,3.70728102,3.15531381,5.5175015,4.1153092,5.98170108.96295591-.47840977,4.11215132-1.8449565,8.65456373-1.8449565s7.69160783,1.36654673,8.65456373,1.8449565c.95979803-.46419958,4.11511183-2.27442006,4.11511183-5.98170108,0-4.25595229-5.25234322-6.07169896-5.3052377-6.08906697-.1953938-.06552477-.34993253-.22025797-.41585327-.41525337-.07302597-.21710014-1.85051744-5.30592754-7.0485846-5.30592754Zm-7.67364739,5.09593249h.00809207-.00809207Z'/>
  </g>
</svg>"""

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
		kind: "GitRepository"
		address: "https://code.v1.dodo.cloud/helm-charts"
		branch: "main"
		path: "charts/soft-serve"
	}
}

ingress: {
	gerrit: { // TODO(gio): rename to soft-serve
		auth: enabled: false
		network: input.network
		subdomain: input.subdomain
		service: {
			name: "soft-serve"
			port: number: 80
		}
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
		info: "Installing SoftServe server"
		values: {
			serviceType: "ClusterIP"
			adminKey: input.adminKey
			sshPublicPort: input.sshPort
			ingress: {
				enabled: false
				domain: _domain
			}
			image: {
				repository: images.softserve.fullName
				tag: images.softserve.tag
				pullPolicy: images.softserve.pullPolicy
			}
		}
	}
}

help: [{
	title: "Access"
	contents: """
	SSH CLI: ssh \(_domain) -p \(input.sshPort) help  
	SSH TUI: ssh \(_domain) -p \(input.sshPort)  
	Create repository: ssh \(_domain) -p \(input.sshPort) repos create \\<REPO-NAME\\>  
	HTTP: git clone https://\(_domain)/\\<REPO-NAME\\>  
	SSH: git clone ssh://\(_domain):\(input.sshPort)/\\<REPO-NAME\\>  

	See following resource on what you can do with Soft-Serve TUI: [https://github.com/charmbracelet/soft-serve](https://github.com/charmbracelet/soft-serve)
	"""
}]
