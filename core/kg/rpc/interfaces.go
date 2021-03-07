package rpc

import "github.com/giolekva/pcloud/core/kg/model"

type appIface interface {
	GetUser(userID string) (*model.User, error)
}
