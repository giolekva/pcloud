package store

import "github.com/giolekva/pcloud/core/kg/model"

// Store interface abstracts away the DB implementation
type Store interface {
	User() UserStore
}

// UserStore .
type UserStore interface {
	Save(user *model.User) (*model.User, error)
	Get(id string) (*model.User, error)
	GetAll() ([]*model.User, error)
	Count() (int64, error)
	GetAllWithOptions(page, perPage int) ([]*model.User, error)
	GetByUsername(username string) (*model.User, error)
}
