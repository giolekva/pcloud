package server

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/giolekva/pcloud/core/kg/common"
	"github.com/giolekva/pcloud/core/kg/log"
)

// Server interface
type Server interface {
	Start() error
	Shutdown() error
}

// Servers represents different server services
type Servers struct {
	servers []Server
	logger  common.LoggerIface
}

// New provides new service application
func New(logger common.LoggerIface) *Servers {
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
	// wait for kill signal before attempting to gracefully shutdown
	// the running service
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interruptChan
	ss.logger.Info("os.Interrupt...")
	ss.shutdown()
}

func (ss *Servers) shutdown() {
	ss.logger.Info("Shutting down...")

	errCh := make(chan error, len(ss.servers))

	for _, server := range ss.servers {
		go func(server Server) {
			errCh <- server.Shutdown()
		}(server)
	}

	for i := 0; i < len(ss.servers); i++ {
		if err := <-errCh; err != nil {
			go func(err error) {
				ss.logger.Error("Shutdown error", log.Err(err))
				os.Exit(1)
			}(err)
			return
		}
	}

	ss.logger.Info("Gracefully stopped")
}
