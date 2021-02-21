package server

import (
	"os"

	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/vardius/shutdown"
)

// Server interface
type Server interface {
	Start() error
	Shutdown() error
}

// Servers represents different server services
type Servers struct {
	servers []Server
	logger  *log.Logger
}

// New provides new service application
func New(logger *log.Logger) *Servers {
	return &Servers{
		logger: logger,
	}
}

// AddServers adds servers to service
func (ss *Servers) AddServers(servers ...Server) {
	ss.servers = append(ss.servers, servers...)
}

// Run runs the service application
func (ss *Servers) Run() {
	for _, server := range ss.servers {
		go func(server Server) {
			if err := server.Start(); err != nil {
				ss.logger.Error("can't start server", log.Err(err))
				os.Exit(1)
			}
		}(server)
	}
	shutdown.GracefulStop(func() { ss.shutdown() })
}

func (ss *Servers) shutdown() {
	ss.logger.Info("shutting down...")

	errCh := make(chan error, len(ss.servers))

	for _, server := range ss.servers {
		go func(server Server) {
			errCh <- server.Shutdown()
		}(server)
	}

	for i := 0; i < len(ss.servers); i++ {
		if err := <-errCh; err != nil {
			go func(err error) {
				ss.logger.Error("shutdown error", log.Err(err))
				os.Exit(1)
			}(err)
			return
		}
	}

	ss.logger.Info("gracefully stopped")
}
