package server

import (
	"fmt"
	"net"
	"os"

	"github.com/giolekva/pcloud/core/kg/common"
	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/model/proto"
	"github.com/giolekva/pcloud/core/kg/rpc"
	"google.golang.org/grpc"
)

// GRPCServerImpl grpc server implementation
type GRPCServerImpl struct {
	Log    common.LoggerIface
	srv    *grpc.Server
	config *model.Config
	app    common.AppIface
}

var _ Server = &GRPCServerImpl{}

// NewGRPCServer creates new GRPC Server
func NewGRPCServer(logger common.LoggerIface, config *model.Config, app common.AppIface) Server {
	a := &GRPCServerImpl{
		Log:    logger,
		config: config,
		app:    app,
	}

	pwd, _ := os.Getwd()
	a.Log.Info("GRPC server current working", log.String("directory", pwd))
	return a
}

// Start method starts a grpc server
func (a *GRPCServerImpl) Start() error {
	a.Log.Info("Starting GRPC Server...")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.config.GRPC.Port))
	if err != nil {
		a.Log.Error("Failed to listen: %v", log.Err(err))
		return err
	}

	a.srv = grpc.NewServer()
	userService := rpc.NewService(a.app)
	proto.RegisterUserServiceServer(a.srv, userService)

	a.Log.Info("GRPC Server is listening on", log.Int("port", a.config.GRPC.Port))
	if err := a.srv.Serve(lis); err != nil {
		a.Log.Error("Failed to serve rpc: %v", log.Err(err))
		return err
	}
	return nil
}

// Shutdown method shuts grpc server down
func (a *GRPCServerImpl) Shutdown() error {
	a.Log.Info("Stopping GRPC Server...")
	a.srv.GracefulStop()
	a.Log.Info("GRPC Server stopped")
	return nil
}
