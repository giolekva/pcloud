package tasks

import (
	"time"

	"github.com/giolekva/pcloud/core/installer"
)

func NewMonitorReleaseTasks(mon installer.HelmReleaseMonitor, rr installer.ReleaseResources) []Task {
	var t []Task
	for _, h := range rr.Helm {
		t = append(t, newMonitorHelm(mon, h))
	}
	return t
}

func NewMonitorRelease(mon installer.HelmReleaseMonitor, rr installer.ReleaseResources) Task {
	return newConcurrentParentTask("Monitor", true, NewMonitorReleaseTasks(mon, rr)...)
}

func newMonitorHelm(mon installer.HelmReleaseMonitor, h installer.Resource) Task {
	t := newLeafTask(h.Info, func() error {
		for {
			if ok, err := mon.IsReleased(h.Namespace, h.Name); err == nil && ok {
				break
			}
			time.Sleep(5 * time.Second)
		}
		return nil
	})
	return &t
}
