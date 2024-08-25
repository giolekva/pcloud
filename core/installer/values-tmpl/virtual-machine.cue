input: {
	name: string @name(Hostname)
	username: string @name(Username)
	authKey: string @name(Auth Key) @role(VPNAuthKey) @usernameField(username)
	cpuCores: int | *1 @name(CPU Cores)
	memory: string | *"2Gi" @name(Memory)
}

name: "Virutal Machine"
namespace: "app-vm"
readme: "Virtual Machine"
description: "Virtual Machine"
icon: """
<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 2048 2048"><path fill="currentColor" d="M1280 384H640V256h640zm0 1024H640v-128h640zm0 256H640v-128h640zM1408 0q27 0 50 10t40 27t28 41t10 50v1792H384V128q0-27 10-50t27-40t41-28t50-10zm0 128H512v1664h896z"/></svg>"""

charts: {
	virtualMachine: {
		kind: "GitRepository"
		address: "https://code.v1.dodo.cloud/helm-charts"
		branch: "main"
		path: "charts/virtual-machine"
	}
}

helm: {
	"virtual-machine": {
		chart: charts.virtualMachine
		values: {
			name: input.name
			cpuCores: input.cpuCores
			memory: input.memory
			disk: {
				source: "https://cloud.debian.org/images/cloud/bookworm-backports/latest/debian-12-backports-generic-amd64.qcow2"
				size: "64Gi"
			}
			ports: [22, 8080]
			cloudInit: userData: _cloudInitUserData
		}
	}
}

_cloudInitUserData: {
	system_info: {
		default_user: {
			name: input.username
			home: "/home/\(input.username)"
		}
	}
	password: "dodo" // TODO(gio): remove if possible
	chpasswd: {
		expire: false
	}
	hostname: input.name
	ssh_pwauth: true
	disable_root: false
	ssh_authorized_keys: [
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOa7FUrmXzdY3no8qNGUk7OPaRcIUi8G7MVbLlff9eB/ lekva@gl-mbp-m1-max.local"
    ]
	runcmd: [
		["sh", "-c", "curl -fsSL https://tailscale.com/install.sh | sh"],
		// TODO(gio): take auth key from input
		// TODO(gio): enable tailscale ssh
		["sh", "-c", "tailscale up --login-server=https://headscale.\(global.domain) --auth-key=\(input.authKey) --accept-routes"],
		["sh", "-c", "curl -fsSL https://code-server.dev/install.sh | HOME=/home/\(input.username) sh"],
		["sh", "-c", "systemctl enable --now code-server@\(input.username)"],
		["sh", "-c", "sleep 10"],
		// TODO(gio): listen only on tailscale interface
		["sh", "-c", "sed -i -e 's/127.0.0.1/0.0.0.0/g' /home/\(input.username)/.config/code-server/config.yaml"],
		["sh", "-c", "sed -i -e 's/auth: password/auth: none/g' /home/\(input.username)/.config/code-server/config.yaml"],
		["sh", "-c", "systemctl restart --now code-server@\(input.username)"],
    ]
}
