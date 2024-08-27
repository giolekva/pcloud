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

out: {
	vm: {
		"\(input.name)": {
			username: input.username
			domain: global.domain
			cpuCores: input.cpuCores
			memory: input.memory
			vpn: {
				enabled: true
				loginServer: "https://headscale.\(global.domain)"
				authKey: input.authKey
			}
		}
	}
}
