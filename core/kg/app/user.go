package app

import (
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
