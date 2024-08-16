package tasks

import (
	"github.com/giolekva/pcloud/core/installer"
)

type InstallFunc func() (installer.ReleaseResources, error)

type dynamicTaskSlice struct {
	t []Task
}

func (d *dynamicTaskSlice) Tasks() []Task {
	return d.t
}

func (d *dynamicTaskSlice) Append(t Task) {
	d.t = append(d.t, t)
}

func NewInstallTask(mon installer.HelmReleaseMonitor, fn InstallFunc) Task {
	d := &dynamicTaskSlice{t: []Task{}}
	var rr installer.ReleaseResources
	done := make(chan error)
	installTask := newLeafTask("Downloading configuration files", func() error {
		var err error
		rr, err = fn()
		return err
	})
	d.Append(&installTask)
	installTask.OnDone(func(err error) {
		if err != nil {
			done <- err
			return
		}
		monTasks := NewMonitorReleaseTasks(mon, rr)
		for _, mt := range monTasks {
			d.Append(mt)
		}
		monitor := newConcurrentParentTask("Monitor", true, monTasks...)
		monitor.OnDone(func(err error) {
			done <- err
		})
		monitor.Start()
	})
	start := func() error {
		installTask.Start()
		return <-done
	}
	t := newParentTask("Installing application", true, start, d)
	return &t
}
