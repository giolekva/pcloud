package server

import (
	"testing"
	"time"

	"github.com/giolekva/pcloud/core/kg/app"
	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/model"
)

type MockServer struct {
	App     appIface
	Servers []Server
	Config  *model.Config
}

func Setup(tb testing.TB) *MockServer {
	if testing.Short() {
		tb.SkipNow()
	}
	app := app.NewTestApp()
	config := model.NewConfig()
	logger := &log.NoOpLogger{}
	grpcServer := NewGRPCServer(logger, config, app)
	httpServer := NewHTTPServer(logger, config, nil)
	ts := &MockServer{
		App:     app,
		Servers: []Server{grpcServer, httpServer},
		Config:  config,
	}
	go grpcServer.Start()
	go httpServer.Start()
	time.Sleep(1 * time.Second)
	return ts
}

func (ts *MockServer) ShutdownServers() {
	done := make(chan bool)
	go func() {
		for _, server := range ts.Servers {
			server.Shutdown()
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		// panic instead of fatal to terminate all tests in this package, otherwise the
		// still running server could spuriously fail subsequent tests.
		panic("failed to shutdown server within 30 seconds")
	}
}
