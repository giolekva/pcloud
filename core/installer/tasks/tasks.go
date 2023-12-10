package tasks

import (
	"fmt"
)

type Status int

const (
	StatusPending Status = 0
	StatusRunning        = 1
	StatusFailed         = 2
	StatusDone           = 3
)

type TaskDoneListener func(err error)

type Task interface {
	Title() string
	Start()
	Status() Status
	Err() error
	Subtasks() []Task
	AddSubtask(t Task) error
	FinalizeSubtasks()
	OnDone(l TaskDoneListener)
}

type basicTask struct {
	title     string
	status    Status
	err       error
	subtasks  []Task
	done      []bool
	finalized bool
	listeners []TaskDoneListener
}

func (b *basicTask) Title() string {
	return b.title
}

func (b *basicTask) Status() Status {
	return b.status
}

func (b *basicTask) Err() error {
	return b.err
}

func (b *basicTask) Subtasks() []Task {
	return b.subtasks
}

func (b *basicTask) AddSubtask(t Task) error {
	if b.finalized {
		return fmt.Errorf("already finalized")
	}
	i := len(b.subtasks)
	b.subtasks = append(b.subtasks, t)
	b.done = append(b.done, false)
	t.OnDone(func(err error) {
		if b.done[i] {
			panic(fmt.Sprintf("already done: %s", b.subtasks[i].Title()))
		}
		b.done[i] = true
		if err != nil {
			b.callDoneListeners(err)
		}
		if !b.finalized {
			return
		}
		done := 0
		for _, d := range b.done {
			if d {
				done++
			} else {
				break
			}
		}
		if done == len(b.subtasks) {
			b.callDoneListeners(nil)
		}
	})
	return nil
}

func (b *basicTask) FinalizeSubtasks() {
	b.finalized = true
}

func (b *basicTask) OnDone(l TaskDoneListener) {
	b.listeners = append(b.listeners, l)
}

func (b *basicTask) callDoneListeners(err error) {
	for _, l := range b.listeners {
		go l(err)
	}
	if err == nil {
		b.status = StatusDone
	} else {
		b.status = StatusFailed
		b.err = err
	}
}
