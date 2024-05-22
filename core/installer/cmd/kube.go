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

func newHelmReleaseMonitor() (installer.HelmReleaseMonitor, error) {
	return installer.NewHelmReleaseMonitor(rootFlags.kubeConfig)
}

func newJobCreator() (installer.JobCreator, error) {
	clientset, err := installer.NewKubeConfig(rootFlags.kubeConfig)
	if err != nil {
		return nil, err
	}
	return installer.NewJobCreator(clientset.BatchV1()), nil
}
