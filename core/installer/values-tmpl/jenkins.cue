input: {
	network: #Network @name(Network)
	subdomain: string @name(Subdomain)
}

_domain: "\(input.subdomain).\(input.network.domain)"
url: "https://\(_domain)"

name: "Jenkins"
namespace: "app-jenkins"
readme: "Jenkins CI/CD"
description: "Build great things at any scale. The leading open source automation server, Jenkins provides hundreds of plugins to support building, deploying and automating any project."
icon: "<svg xmlns='http://www.w3.org/2000/svg' width='50' height='50' viewBox='0 0 24 24'><path fill='currentColor' d='M2.872 24h-.975a3.866 3.866 0 0 1-.07-.197c-.215-.666-.594-1.49-.692-2.154c-.146-.984.78-1.039 1.374-1.465c.915-.66 1.635-1.025 2.627-1.62c.295-.179 1.182-.624 1.281-.829c.201-.408-.345-.982-.49-1.3c-.225-.507-.345-.937-.376-1.435c-.824-.13-1.455-.627-1.844-1.185c-.63-.925-1.066-2.635-.525-3.936c.045-.103.254-.305.285-.463c.06-.308-.105-.72-.12-1.048c-.06-1.692.284-3.15 1.425-3.66c.463-1.84 2.113-2.453 3.673-3.367c.58-.342 1.224-.562 1.89-.807c2.372-.877 6.027-.712 7.994.783c.836.633 2.176 1.97 2.656 2.939c1.262 2.555 1.17 6.825.287 9.934c-.12.421-.29 1.032-.533 1.533c-.168.35-.689 1.05-.625 1.36c.064.314 1.19 1.17 1.432 1.395c.434.422 1.26.975 1.324 1.5c.07.557-.248 1.336-.41 1.875c-.217.721-.436 1.441-.654 2.131H2.87zm11.104-3.54a7.723 7.723 0 0 0-2.065-.757c-.87-.164-.78 1.188-.75 1.994c.03.643.36 1.316.51 1.744c.076.197.09.41.256.449c.3.068 1.29-.326 1.575-.479c.6-.328 1.064-.844 1.574-1.189c.016-.17.016-.34.03-.508a2.648 2.648 0 0 0-1.095-.277c.314-.15.75-.15 1.035-.332l.016-.193c-.496-.03-.69-.254-1.021-.436zm7.454 2.935a17.78 17.78 0 0 0 .465-1.752c.06-.287.215-.918.178-1.176c-.059-.459-.684-.799-1.004-1.086c-.584-.525-.95-.975-1.56-1.469c-.249.375-.78.615-.983.914c1.447-.689 1.71 2.625 1.141 3.69c.09.329.391.45.514.735l-.086.166h1.29c.013 0 .03 0 .044.014zm-6.634-.012c-.05-.074-.1-.135-.15-.209l-.301.195h.45zm2.77 0c.008-.209.018-.404.03-.598c-.53.029-.825-.48-1.196-.527c-.324-.045-.6.361-1.02.195c-.095.105-.183.227-.284.316c.154.18.295.375.424.584h.815a.298.298 0 0 1 .3-.285c.165 0 .284.121.284.27h.66zm2.116 0c-.314-.479-.947-.898-1.68-.555l-.03.541h1.71zm-8.51 0l-.104-.344c-.225-.72-.36-1.26-.405-1.68c-.914-.436-1.875-.87-2.654-1.426c-.15-.105-1.109-1.35-1.23-1.305c-1.739.676-3.359 1.86-4.814 2.984c.256.557.48 1.141.69 1.74h8.505zm8.265-2.113c-.029-.512-.164-1.56-.48-1.74c-.66-.39-1.846.78-2.34.943c.045.15.135.271.15.48c.285-.074.645-.029.898.092c-.299.03-.629.03-.824.164c-.074.195.016.48-.029.764c.69.197 1.5.303 2.385.332c.164-.227.225-.645.211-1.082zm-4.08-.36c-.044.375.046.51.12.943c1.26.391 1.034-1.74-.135-.959zM8.76 19.5c-.45.457 1.27 1.082 1.814 1.115c0-.29.165-.564.135-.77c-.65-.118-1.502-.042-1.945-.347zm5.565.215c0 .043-.061.03-.068.064c.58.451 1.014.545 1.802.51c.354-.262.67-.563 1.043-.807c-.855.074-1.931.607-2.774.23zm3.42-17.726c-1.606-.906-4.35-1.591-6.076-.731c-1.38.692-3.27 1.84-3.899 3.292c.6 1.402-.166 2.686-.226 4.109c-.018.757.36 1.42.391 2.242c-.2.338-.825.38-1.26.356c-.146-.729-.4-1.549-1.155-1.63c-1.064-.116-1.845.764-1.89 1.683c-.06 1.08.833 2.864 2.085 2.745c.488-.046.608-.54 1.139-.54c.285.57-.445.75-.523 1.154c-.016.105.06.511.104.705c.233.944.744 2.16 1.245 2.88c.635.9 1.884 1.051 3.229 1.141c.24-.525 1.125-.48 1.706-.346c-.691-.27-1.336-.945-1.875-1.529c-.615-.676-1.23-1.41-1.261-2.28c1.155 1.604 2.1 3 4.2 3.704c1.59.525 3.45-.254 4.664-1.109c.51-.359.811-.93 1.17-1.439c1.35-1.936 1.98-4.71 1.846-7.394c-.06-1.111-.06-2.221-.436-2.955c-.389-.781-1.695-1.471-2.475-.781c-.15-.764.63-1.23 1.545-.96c-.66-.854-1.336-1.858-2.266-2.384zM13.58 14.896c.615 1.544 2.724 1.363 4.505 1.323c-.084.194-.256.435-.465.515c-.57.232-2.145.408-2.937-.012c-.506-.27-.824-.873-1.102-1.227c-.137-.172-.795-.608-.012-.609zm.164-.87c.893.464 2.52.517 3.731.48c.066.267.066.593.068.913c-1.55.08-3.386-.304-3.794-1.395h-.005zm6.675-.586c-.473.9-1.145 1.897-2.539 1.928c-.023-.284-.045-.735 0-.904c1.064-.103 1.727-.646 2.543-1.017zm-.649-.667c-1.02.66-2.154 1.375-3.824 1.21c-.351-.31-.485-1-.14-1.458c.181.313.06.885.57.97c.944.165 2.038-.579 2.73-.84c.42-.713-.046-.976-.42-1.433c-.782-.93-1.83-2.1-1.802-3.51c.314-.224.346.346.391.45c.404.96 1.424 2.175 2.174 3c.18.21.48.39.51.524c.092.39-.254.854-.209 1.11zm-13.439-.675c-.314-.184-.393-.99-.768-1.01c-.535-.03-.438 1.05-.436 1.68c-.37-.33-.435-1.365-.164-1.89c-.308-.15-.445.164-.618.284c.22-1.59 2.34-.734 1.99.96zM4.713 5.995c-.685.756-.54 2.174-.459 3.188c1.244-.785 2.898.06 2.883 1.394c.595-.016.223-.744.115-1.215c-.353-1.528.592-3.187.041-4.59c-1.064.084-1.939.52-2.578 1.215zm9.12 1.113c.307.562.404 1.148.84 1.57c.195.19.574.424.387.95c-.045.121-.365.391-.551.45c-.674.195-2.254.03-1.721-.81c.563.015 1.314.36 1.732-.045c-.314-.524-.885-1.53-.674-2.13zm6.198-.013h.068c.33.668.6 1.375 1.004 1.965c-.27.628-2.053 1.19-2.023.057c.39-.17 1.05-.035 1.395-.25c-.193-.556-.48-1.006-.434-1.771zm-6.927-1.617c-1.422-.33-2.131.592-2.56 1.553c-.384-.094-.231-.615-.135-.883c.255-.701 1.28-1.633 2.119-1.506c.359.057.848.386.576.834zM9.642 1.593c-1.56.44-3.56 1.574-4.2 2.974c.495-.07.84-.321 1.33-.351c.186-.016.428.074.641.015c.424-.104.78-1.065 1.102-1.41c.31-.345.685-.496.94-.81c.167-.09.409-.074.42-.33c-.073-.075-.15-.135-.232-.105z'/></svg>"

_jenkinsServiceHTTPPortNumber: 80

ingress: {
	jenkins: {
		auth: enabled: false
		network: networks.private
		subdomain: input.subdomain
		service: {
			name: "jenkins"
			port: number: _jenkinsServiceHTTPPortNumber
		}
	}
}

images: {
    jenkins: {
        repository: "jenkins"
        name: "jenkins"
        tag: "2.452-jdk17"
        pullPolicy: "IfNotPresent"
    }
}

charts: {
    jenkins: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/jenkins"
    }
	oauth2Client: {
		kind: "GitRepository"
		address: "https://github.com/giolekva/pcloud.git"
		branch: "main"
		path: "charts/oauth2-client"
	}
}

volumes: jenkins: size: "10Gi"

_oauth2ClientCredentials:  "oauth2-credentials"
_oauth2ClientId: "client_id"
_oauth2ClientSecret: "client_secret"

helm: {
	"oauth2-client": {
		chart: charts.oauth2Client
		info: "Creating OAuth2 client"
		values: {
			name: "oauth2-client"
			secretName: _oauth2ClientCredentials
			grantTypes: ["authorization_code"]
			scope: "openid profile email offline offline_access"
			hydraAdmin: "http://hydra-admin.\(global.id)-core-auth.svc.cluster.local"
			redirectUris: ["https://\(_domain)/securityRealm/finishLogin"]
			tokenEndpointAuthMethod: "client_secret_post"
		}
	}
    jenkins: {
        chart: charts.jenkins
		info: "Installing Jenkins server"
        values: {
			fullnameOverride: "jenkins"
			controller: {
				image: {
					repository: images.jenkins.imageName
					tag: images.jenkins.tag
					pullPolicy: images.jenkins.pullPolicy
				}
				jenkinsUrlProtocol: "https://"
				jenkinsUrl: _domain
				sidecars: configAutoReload: enabled: false
				ingress: enabled: false
				servicePort: _jenkinsServiceHTTPPortNumber
				installPlugins: [
					"kubernetes:4203.v1dd44f5b_1cf9",
					"workflow-aggregator:596.v8c21c963d92d",
					"git:5.2.1",
					"configuration-as-code:1775.v810dc950b_514",
					"gerrit-code-review:0.4.9",
					"oic-auth:4.239.v325750a_96f3b_",
				]
				additionalExistingSecrets: [{
					name: _oauth2ClientCredentials
					keyName: _oauth2ClientId
				}, {
					name: _oauth2ClientCredentials
					keyName: _oauth2ClientSecret
				}]
				JCasC: {
					defaultConfig: true
					overwriteConfiguration: false
					securityRealm: """
oic:
  clientId: "${\(_oauth2ClientCredentials)-\(_oauth2ClientId)}"
  clientSecret: "${\(_oauth2ClientCredentials)-\(_oauth2ClientSecret)}"
  wellKnownOpenIDConfigurationUrl: "https://hydra.\(global.domain)/.well-known/openid-configuration"
  userNameField: "email"
"""
				}
			}
			agent: {
				runAsUser: 1000
				runAsGroup: 1000
				jenkinsUrl: "http://jenkins.\(release.namespace).svc.cluster.local"
			}
			persistence: {
				enabled: true
				existingClaim: volumes.jenkins.name
			}
        }
    }
}
