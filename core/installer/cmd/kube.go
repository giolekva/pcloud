package main

import (
	"github.com/giolekva/pcloud/core/installer"
)

func newNSCreator() (installer.NamespaceCreator, error) {
	return installer.NewNamespaceCreator(rootFlags.kubeConfig)
}

func newZoneFetcher() (installer.ZoneStatusFetcher, error) {
	return installer.NewZoneStatusFetcher(rootFlags.kubeConfig)
}
