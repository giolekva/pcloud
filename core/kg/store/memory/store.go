package memory

import "github.com/giolekva/pcloud/core/kg/store"

type MemoryStore struct {
	stores memoryStoreStores
}

var _ store.Store = &MemoryStore{}

type memoryStoreStores struct {
	user    store.UserStore
	session store.SessionStore
}

func New() *MemoryStore {
	store := &MemoryStore{}
	store.stores.user = newMemoryUserStore(store)
	store.stores.session = newMemorySessionStore(store)

	return store
}

func (ms *MemoryStore) User() store.UserStore {
	return ms.stores.user
}

func (ms *MemoryStore) Session() store.SessionStore {
	return ms.stores.session
}
