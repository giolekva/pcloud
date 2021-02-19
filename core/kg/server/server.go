package server

import (
	"net"
	"os"

	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/store"
	"google.golang.org/grpc"
)

const listenerPort = ":9081"

// Server type defines application global state
type Server struct {
	Log   *log.Logger
	srv   *grpc.Server
	store store.Store
}

// NewServer creates new Server
func NewServer(logger *log.Logger) (*Server, error) {
	a := &Server{
		Log: logger,
	}

	pwd, _ := os.Getwd()
	a.Log.Info("Printing current working", log.String("directory", pwd))
	return a, nil
}

// Start method starts an app
func (a *Server) Start() error {
	// settings := model.NewConfig().SqlSettings
	// a.store = sqlstore.New(settings)

	lis, err := net.Listen("tcp", listenerPort)
	if err != nil {
		a.Log.Error("failed to listen: %v", log.Err(err))
		return err
	}

	a.srv = grpc.NewServer()

	a.Log.Info("Server is listening on", log.String("port", listenerPort))
	if err := a.srv.Serve(lis); err != nil {
		a.Log.Error("failed to serve: %v", log.Err(err))
		return err
	}
	a.Log.Info("Server stopped")
	return nil
}

// Shutdown method shuts server down
func (a *Server) Shutdown() {
	a.Log.Info("Stoping Server...")
	a.srv.GracefulStop()
}
