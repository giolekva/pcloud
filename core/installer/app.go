package installer

import (
	"embed"
	"log"
	"text/template"
)

type App struct {
	Name      string
	Templates []*template.Template
}

//go:embed values-tmpl
var valuesTmpls embed.FS

func CreateAllApps() []App {
	tmpls, err := template.ParseFS(valuesTmpls, "values-tmpl/*.yaml")
	if err != nil {
		log.Fatal(err)
	}
	return []App{
		// CreateAppIngressPublic(tmpls),
		CreateAppIngressPrivate(tmpls),
		CreateAppCoreAuth(tmpls),
		CreateAppVaultwarden(tmpls),
		CreateAppMatrix(tmpls),
		CreateAppPihole(tmpls),
		CreateAppMaddy(tmpls),
		CreateAppQBittorrent(tmpls),
		CreateAppJellyfin(tmpls),
		CreateAppRpuppy(tmpls),
		CreateAppHeadscale(tmpls),
	}
}

func CreateAppIngressPublic(tmpls *template.Template) App {
	return App{
		"ingress-public",
		[]*template.Template{
			tmpls.Lookup("ingress-public.yaml"),
		},
	}
}

func CreateAppIngressPrivate(tmpls *template.Template) App {
	return App{
		"ingress-private",
		[]*template.Template{
			// tmpls.Lookup("vpn-mesh-config.yaml"),
			tmpls.Lookup("ingress-private.yaml"),
			tmpls.Lookup("certificate-issuer.yaml"),
		},
	}
}

func CreateAppCoreAuth(tmpls *template.Template) App {
	return App{
		"core-auth",
		[]*template.Template{
			tmpls.Lookup("core-auth-storage.yaml"),
			tmpls.Lookup("core-auth.yaml"),
		},
	}
}

func CreateAppVaultwarden(tmpls *template.Template) App {
	return App{
		"vaultwarden",
		[]*template.Template{
			tmpls.Lookup("vaultwarden.yaml"),
		},
	}
}

func CreateAppMatrix(tmpls *template.Template) App {
	return App{
		"matrix",
		[]*template.Template{
			tmpls.Lookup("matrix-storage.yaml"),
			tmpls.Lookup("matrix.yaml"),
		},
	}
}

func CreateAppPihole(tmpls *template.Template) App {
	return App{
		"pihole",
		[]*template.Template{
			tmpls.Lookup("pihole.yaml"),
		},
	}
}

func CreateAppMaddy(tmpls *template.Template) App {
	return App{
		"maddy",
		[]*template.Template{
			tmpls.Lookup("maddy.yaml"),
		},
	}
}

func CreateAppQBittorrent(tmpls *template.Template) App {
	return App{
		"qbittorrent",
		[]*template.Template{
			tmpls.Lookup("qbittorrent.yaml"),
		},
	}
}

func CreateAppJellyfin(tmpls *template.Template) App {
	return App{
		"jellyfin",
		[]*template.Template{
			tmpls.Lookup("jellyfin.yaml"),
		},
	}
}

func CreateAppRpuppy(tmpls *template.Template) App {
	return App{
		"rpuppy",
		[]*template.Template{
			tmpls.Lookup("rpuppy.yaml"),
		},
	}
}

func CreateAppHeadscale(tmpls *template.Template) App {
	return App{
		"headscale",
		[]*template.Template{
			tmpls.Lookup("headscale.yaml"),
		},
	}
}
