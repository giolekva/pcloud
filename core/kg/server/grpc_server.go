package server

import (
	"net"
	"os"

	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/store"
	"google.golang.org/grpc"
)

const listenerPort = ":9081"

// GRPCServerImpl grpc server implementation
type GRPCServerImpl struct {
	Log   *log.Logger
	srv   *grpc.Server
	store store.Store
}

var _ Server = &GRPCServerImpl{}

// NewGRPCServer creates new GRPC Server
func NewGRPCServer(logger *log.Logger) Server {
	a := &GRPCServerImpl{
		Log: logger,
	}

	pwd, _ := os.Getwd()
	a.Log.Info("GRPC server current working", log.String("directory", pwd))
	return a
}

// Start method starts an app
func (a *GRPCServerImpl) Start() error {
	a.Log.Info("Starting GRPC Server...")

	// settings := model.NewConfig().SqlSettings
	// a.store = sqlstore.New(settings)

	lis, err := net.Listen("tcp", listenerPort)
	if err != nil {
		a.Log.Error("failed to listen: %v", log.Err(err))
		return err
	}

	a.srv = grpc.NewServer()

	a.Log.Info("GRPC Server is listening on", log.String("port", listenerPort))
	if err := a.srv.Serve(lis); err != nil {
		a.Log.Error("failed to serve: %v", log.Err(err))
		return err
	}
	return nil
}

// Shutdown method shuts server down
func (a *GRPCServerImpl) Shutdown() error {
	a.Log.Info("Stopping GRPC Server...")
	a.srv.GracefulStop()
	return nil
}
