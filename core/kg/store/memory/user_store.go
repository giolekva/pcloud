package memory

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/store"
)

type memoryUserStore struct {
	*MemoryStore

	users map[string]*model.User
	maxID int
	mutex sync.RWMutex
}

var _ store.UserStore = &memoryUserStore{}

func newMemoryUserStore(mStore *MemoryStore) store.UserStore {
	us := &memoryUserStore{
		MemoryStore: mStore,
		users:       map[string]*model.User{},
		maxID:       1,
	}
	return us
}

func (us *memoryUserStore) Save(user *model.User) (*model.User, error) {
	us.mutex.Lock()
	defer us.mutex.Unlock()
	if user.ID == "" {
		user.ID = strconv.Itoa(us.maxID)
		us.maxID++
		user.CreateAt = time.Now().Unix()
		user.DeleteAt = 0
	} else {
		user.UpdateAt = time.Now().Unix()
	}
	us.users[user.ID] = user
	return user, nil
}

func (us *memoryUserStore) Get(id string) (*model.User, error) {
	us.mutex.RLock()
	defer us.mutex.RUnlock()
	user, ok := us.users[id]
	if !ok {
		return nil, errors.New("User not found")
	}
	return user.Clone(), nil
}

func (us *memoryUserStore) GetAll() ([]*model.User, error) {
	us.mutex.RLock()
	defer us.mutex.RUnlock()
	users := make([]*model.User, 0, len(us.users))
	for _, user := range us.users {
		users = append(users, user.Clone())
	}
	return users, nil
}

func (us *memoryUserStore) GetAllWithOptions(page, perPage int) ([]*model.User, error) {
	us.mutex.RLock()
	defer us.mutex.RUnlock()

	// Not an ideal way to implement a pagination over a map but memory store is for testing anyway.
	v := make([]*model.User, 0, len(us.users))
	for _, value := range us.users {
		v = append(v, value)
	}
	users := make([]*model.User, 0, perPage)
	startIndex := page * perPage
	for i := startIndex; i < startIndex+perPage && i < len(us.users); i++ {
		users = append(users, v[i].Clone())
	}
	return users, nil
}

func (us *memoryUserStore) Count() (int64, error) {
	return int64(us.maxID) - 1, nil
}
