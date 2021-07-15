package common

import (
	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/model"
)

type LoggerIface interface {
	Debug(message string, fields ...log.Field)
	Info(message string, fields ...log.Field)
	Warn(message string, fields ...log.Field)
	Error(message string, fields ...log.Field)
}

type AppIface interface {
	GetUser(userID string) (*model.User, error)
	CreateUser(user *model.User) (*model.User, error)
	GetUsers(page, perPage int) ([]*model.User, error)

	AuthenticateUserForLogin(userID, loginID, password string) (*model.User, error)

	CreateSession(userID string) (*model.Session, error)
	Session() *model.Session
	RevokeSession(sessionID string) error
	GetSession(token string) (*model.Session, error)
	SetSession(s *model.Session)
}
