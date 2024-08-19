package soft

import (
	"sync"
	"testing"
)

type mockRepoIO struct {
	RepoFS
	addr string
	t    *testing.T
	l    sync.Locker
}

func NewMockRepoIO(fs RepoFS, addr string, t *testing.T) RepoIO {
	return &mockRepoIO{
		RepoFS: fs,
		addr:   addr,
		t:      t,
		l:      &sync.Mutex{},
	}
}

func (r mockRepoIO) FullAddress() string {
	return r.addr
}

func (r mockRepoIO) Pull() error {
	r.t.Logf("Pull: %s", r.addr)
	return nil
}

func (r mockRepoIO) CommitAndPush(message string, opts ...PushOption) (string, error) {
	r.t.Logf("Commit and push: %s", message)
	return "", nil
}

func (r mockRepoIO) Do(op DoFn, _ ...DoOption) (string, error) {
	r.l.Lock()
	defer r.l.Unlock()
	msg, err := op(r)
	if err != nil {
		return "", err
	}
	return r.CommitAndPush(msg)
}
