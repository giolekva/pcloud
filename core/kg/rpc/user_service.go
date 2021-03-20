package rpc

import (
	"context"

	"github.com/giolekva/pcloud/core/kg/model/proto"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type userService struct {
	proto.UnimplementedUserServiceServer
	app appIface
}

// NewService returns new user service
func NewService(app appIface) proto.UserServiceServer {
	s := &userService{
		app: app,
	}
	return s
}

func (us *userService) GetUser(c context.Context, r *proto.GetUserRequest) (*proto.GetUserResponse, error) {
	user, err := us.app.GetUser(r.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "can't get user from application")
	}
	return &proto.GetUserResponse{
		User: &proto.User{
			Id:                 &user.ID,
			CreateAt:           &timestamppb.Timestamp{Seconds: user.CreateAt},
			UpdateAt:           &timestamppb.Timestamp{Seconds: user.UpdateAt},
			DeleteAt:           &timestamppb.Timestamp{Seconds: user.DeleteAt},
			Username:           user.Username,
			Password:           user.Password,
			LastPasswordUpdate: &timestamppb.Timestamp{Seconds: user.LastPasswordUpdate},
		},
	}, nil
}
func (us *userService) ListUsers(context.Context, *proto.ListUserRequest) (*proto.ListUserResponse, error) {
	return nil, nil
}
func (us *userService) CreateUser(context.Context, *proto.CreateUserRequest) (*proto.CreateUserResponse, error) {
	return nil, nil
}
