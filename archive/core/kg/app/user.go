package app

import (
	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// GetUser returns user
func (a *App) GetUser(userID string) (*model.User, error) {
	user, err := a.store.User().Get(userID)
	if err != nil {
		return nil, errors.Wrap(err, "can't get user from store")
	}
	return user, nil
}

// CreateUser creates a user. For now it is used only for creation of the very first user
func (a *App) CreateUser(user *model.User) (*model.User, error) {
	if !a.isFirstUserAccount() {
		return nil, errors.New("not a first user")
	}

	updatedUser, err := a.store.User().Save(user)
	if err != nil {
		return nil, errors.Wrap(err, "can't save user to the DB")
	}
	return updatedUser, nil
}

//GetUsers returns list of users
func (a *App) GetUsers(page, perPage int) ([]*model.User, error) {
	users, err := a.store.User().GetAllWithOptions(page, perPage)
	if err != nil {
		return nil, errors.Wrap(err, "can't get users with options from store")
	}
	return users, nil
}

func (a *App) isFirstUserAccount() bool {
	count, err := a.store.User().Count()
	if err != nil {
		a.logger.Error("error fetching first user account", log.Err(err))
	}
	return count > 0
}

// HashPassword hashes user's password
func HashPassword(password string) string {
	if password == "" {
		return ""
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		panic(err)
	}

	return string(hash)
}
