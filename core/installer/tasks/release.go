package tasks

import (
	"fmt"
	"time"

	"github.com/giolekva/pcloud/core/installer"
)

func NewMonitorRelease(mon installer.HelmReleaseMonitor, rr installer.ReleaseResources) Task {
	var t []Task
	for _, h := range rr.Helm {
		t = append(t, newMonitorHelm(mon, h))
	}
	return newConcurrentParentTask("Monitor", true, t...)
}

func newMonitorHelm(mon installer.HelmReleaseMonitor, h installer.Resource) Task {
	t := newLeafTask(fmt.Sprintf("%s/%s", h.Namespace, h.Name), func() error {
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
