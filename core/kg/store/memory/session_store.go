package memory

import (
	"strconv"
	"sync"
	"time"

	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/store"
	"github.com/pkg/errors"
)

type memorySessionStore struct {
	*MemoryStore

	sessions map[string]*model.Session
	maxID    int
	mutex    sync.RWMutex
}

var _ store.SessionStore = &memorySessionStore{}

func newMemorySessionStore(mStore *MemoryStore) store.SessionStore {
	ss := &memorySessionStore{
		MemoryStore: mStore,
		sessions:    map[string]*model.Session{},
		maxID:       1,
	}
	return ss
}

func (ss *memorySessionStore) Save(session *model.Session) (*model.Session, error) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if session.ID != "" {
		return nil, errors.Errorf("invalid session input id: %s", session.ID)
	}
	session.ID = strconv.Itoa(ss.maxID)
	ss.maxID++
	session.CreateAt = time.Now().Unix()
	session.LastActivityAt = session.CreateAt

	if session.Token == "" {
		session.Token = session.ID
	}

	ss.sessions[session.ID] = session

	return session, nil
}

func (ss *memorySessionStore) Remove(sessionID string) error {
	if _, ok := ss.sessions[sessionID]; !ok {
		return errors.New("session not found")
	}
	delete(ss.sessions, sessionID)
	return nil
}

func (ss *memorySessionStore) Get(sessionIDOrToken string) (*model.Session, error) {
	if session, ok := ss.sessions[sessionIDOrToken]; ok {
		return session, nil
	}
	for _, session := range ss.sessions {
		if session.Token == sessionIDOrToken {
			return session, nil
		}
	}
	return nil, errors.New("session not found")
}
