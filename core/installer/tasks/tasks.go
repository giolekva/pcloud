package tasks

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
	OnDone(l TaskDoneListener)
}

type basicTask struct {
	title     string
	status    Status
	err       error
	listeners []TaskDoneListener
}

func newBasicTask(title string) basicTask {
	return basicTask{
		title:     title,
		status:    StatusPending,
		err:       nil,
		listeners: make([]TaskDoneListener, 0),
	}
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

type leafTask struct {
	basicTask
	start func() error
}

func newLeafTask(title string, start func() error) leafTask {
	return leafTask{
		basicTask: newBasicTask(title),
		start:     start,
	}
}

func (b *leafTask) Subtasks() []Task {
	return make([]Task, 0)
}

func (b *leafTask) Start() {
	b.callDoneListeners(b.start())
}

type parentTask struct {
	leafTask
	subtasks []Task
}

func newParentTask(title string, start func() error, subtasks ...Task) parentTask {
	return parentTask{
		leafTask: newLeafTask(title, start),
		subtasks: subtasks,
	}
}

func (t *parentTask) Subtasks() []Task {
	return t.subtasks
}

type sequentialParentTask struct {
	parentTask
}

func newSequentialParentTask(title string, subtasks ...Task) sequentialParentTask {
	start := func() error {
		errCh := make(chan error)
		for i := range subtasks[:len(subtasks)-1] {
			next := i + 1
			subtasks[i].OnDone(func(err error) {
				if err == nil {
					go subtasks[next].Start()
				} else {
					errCh <- err
				}
			})
		}
		subtasks[len(subtasks)-1].OnDone(func(err error) {
			errCh <- err
		})
		go subtasks[0].Start()
		return <-errCh
	}
	t := sequentialParentTask{
		parentTask: newParentTask(title, start, subtasks...),
	}
	return t
}
