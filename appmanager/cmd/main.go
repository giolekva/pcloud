package main

import (
	app "github.com/giolekva/pcloud/appmanager"

	"github.com/golang/glog"
)

func main() {
	unpacker := app.NewHelmUnpacker("/usr/local/bin/helm")
	temps, err := unpacker.Unpack("/Users/lekva/dev/go/src/github.com/giolekva/pcloud/apps/rpuppy/chart",
		"app-rpuppy",
		map[string]string{
			"replicas":    "2",
			"servicePort": "8080",
		},
	)
	if err != nil {
		panic(err)
	}
	glog.Info(temps)
}
