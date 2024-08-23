package tasks

import (
	"fmt"
	"path/filepath"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/cluster"
	"github.com/giolekva/pcloud/core/installer/soft"
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

func NewClusterInitTask(m cluster.Manager, server cluster.Server, cnc installer.ClusterNetworkConfigurator, repo soft.RepoIO, setupFn cluster.ClusterSetupFunc) Task {
	d := &dynamicTaskSlice{t: []Task{}}
	done := make(chan error)
	setupTask := newLeafTask(fmt.Sprintf("Installing dodo on %s", server.IP.String()), func() error {
		_, err := m.Init(server, setupFn)
		return err
	})
	d.Append(&setupTask)
	setupTask.OnDone(func(err error) {
		if err != nil {
			done <- err
			return
		}
		if err := cnc.AddCluster(m.State().Name, m.State().IngressIP); err != nil {
			done <- err
			return
		}
		_, err = repo.Do(func(fs soft.RepoFS) (string, error) {
			if err := soft.WriteJson(fs, fmt.Sprintf("/clusters/%s/config.json", m.State().Name), m.State()); err != nil {
				return "", err
			}
			return fmt.Sprintf("add server to cluster: %s", m.State().Name), nil
		})
		done <- err
	})
	start := func() error {
		setupTask.Start()
		return <-done
	}
	t := newParentTask("Installing application", true, start, d)
	return &t
}

func NewRemoveClusterTask(m cluster.Manager, cnc installer.ClusterNetworkConfigurator,
	repo soft.RepoIO) Task {
	t := newLeafTask(fmt.Sprintf("Removing %s cluster", m.State().Name), func() error {
		if err := cnc.RemoveCluster(m.State().Name, m.State().IngressIP); err != nil {
			return err
		}
		_, err := repo.Do(func(fs soft.RepoFS) (string, error) {
			if err := fs.RemoveAll(fmt.Sprintf("/clusters/%s", m.State().Name)); err != nil {
				return "", err
			}
			kustPath := filepath.Join("/clusters", "kustomization.yaml")
			kust, err := soft.ReadKustomization(fs, kustPath)
			if err != nil {
				return "", err
			}
			kust.RemoveResources(m.State().Name)
			soft.WriteYaml(fs, kustPath, kust)
			return fmt.Sprintf("remove cluster: %s", m.State().Name), nil
		})
		return err
	})
	return &t
}

func NewClusterJoinControllerTask(m cluster.Manager, server cluster.Server, repo soft.RepoIO) Task {
	d := &dynamicTaskSlice{t: []Task{}}
	done := make(chan error)
	setupTask := newLeafTask(fmt.Sprintf("Joining %s to %s cluster", server.IP.String(), m.State().Name), func() error {
		return m.JoinController(server)
	})
	d.Append(&setupTask)
	setupTask.OnDone(func(err error) {
		if err != nil {
			done <- err
			return
		}
		_, err = repo.Do(func(fs soft.RepoFS) (string, error) {
			if err := soft.WriteJson(fs, fmt.Sprintf("/clusters/%s/config.json", m.State().Name), m.State()); err != nil {
				return "", err
			}
			return fmt.Sprintf("add controller server to cluster: %s", m.State().Name), nil
		})
		done <- err
	})
	start := func() error {
		setupTask.Start()
		return <-done
	}
	t := newParentTask("Installing application", true, start, d)
	return &t
}

func NewClusterJoinWorkerTask(m cluster.Manager, server cluster.Server, repo soft.RepoIO) Task {
	d := &dynamicTaskSlice{t: []Task{}}
	done := make(chan error)
	setupTask := newLeafTask(fmt.Sprintf("Joining %s to %s cluster", server.IP.String(), m.State().Name), func() error {
		return m.JoinWorker(server)
	})
	d.Append(&setupTask)
	setupTask.OnDone(func(err error) {
		if err != nil {
			done <- err
			return
		}
		_, err = repo.Do(func(fs soft.RepoFS) (string, error) {
			if err := soft.WriteJson(fs, fmt.Sprintf("/clusters/%s/config.json", m.State().Name), m.State()); err != nil {
				return "", err
			}
			return fmt.Sprintf("add worker server to cluster: %s", m.State().Name), nil
		})
		done <- err
	})
	start := func() error {
		setupTask.Start()
		return <-done
	}
	t := newParentTask("Installing application", true, start, d)
	return &t
}

func NewClusterRemoveServerTask(m cluster.Manager, server string, repo soft.RepoIO) Task {
	d := &dynamicTaskSlice{t: []Task{}}
	done := make(chan error)
	setupTask := newLeafTask(fmt.Sprintf("Removing %s from %s cluster", server, m.State().Name), func() error {
		return m.RemoveServer(server)
	})
	d.Append(&setupTask)
	setupTask.OnDone(func(err error) {
		if err != nil {
			done <- err
			return
		}
		_, err = repo.Do(func(fs soft.RepoFS) (string, error) {
			if err := soft.WriteJson(fs, fmt.Sprintf("/clusters/%s/config.json", m.State().Name), m.State()); err != nil {
				return "", err
			}
			return fmt.Sprintf("remove %s from cluster: %s", server, m.State().Name), nil
		})
		done <- err
	})
	start := func() error {
		setupTask.Start()
		return <-done
	}
	t := newParentTask("Installing application", true, start, d)
	return &t
}
