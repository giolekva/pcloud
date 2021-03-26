package app

import (
	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/pkg/errors"
)

// GetUser returns user
func (a *App) GetUser(userID string) (*model.User, error) {
	user, err := a.store.User().Get(userID)
	if err != nil {
		return nil, errors.Wrap(err, "can't get user from store")
	}
	return user, nil
}

func (a *App) CreateUser(user *model.User) (*model.User, error) {
	if !a.isFirstUserAccount() {
		return nil, errors.New("not a first user")
	}

	user.HashPassword()
	updatedUser, err := a.store.User().Save(user)
	if err != nil {
		return nil, errors.Wrap(err, "can't save user to the DB")
	}
	return updatedUser, nil
}

func (a *App) isFirstUserAccount() bool {
	count, err := a.store.User().Count()
	if err != nil {
		a.logger.Error("error fetching first user account", log.Err(err))
	}
	return count > 0
}
