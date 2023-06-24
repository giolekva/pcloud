package main

import (
	"github.com/giolekva/pcloud/core/installer"
)

func newNSCreator() (installer.NamespaceCreator, error) {
	if rootFlags.kubeConfig != "" {
		return installer.NewOutOfClusterNamespaceCreator(rootFlags.kubeConfig)
	} else {
		return installer.NewInClusterNamespaceCreator()
	}
}
