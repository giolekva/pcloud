package rpc

import (
	"context"

	"github.com/giolekva/pcloud/core/kg/app"
	"github.com/giolekva/pcloud/core/kg/model/proto"
)

type userService struct {
	proto.UnimplementedUserServiceServer
	app *app.App
}

// NewService returns new user service
func NewService(app *app.App) proto.UserServiceServer {
	s := &userService{
		app: app,
	}

	return s
}

func (us *userService) GetUser(context.Context, *proto.GetUserRequest) (*proto.User, error) {
	// us.app.getUser...
	return nil, nil
}
func (us *userService) ListUsers(context.Context, *proto.ListUserRequest) (*proto.ListUserResponse, error) {
	return nil, nil
}
func (us *userService) CreateUser(context.Context, *proto.CreateUserRequest) (*proto.User, error) {
	return nil, nil
}
