package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/giolekva/pcloud/core/kg/api/rest"
	"github.com/giolekva/pcloud/core/kg/common"
	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/gorilla/mux"
)

// HTTPServerImpl http server implementation
type HTTPServerImpl struct {
	srv     *http.Server
	routers *rest.Router
	config  *model.Config
	app     common.AppIface
	logger  common.LoggerIface
}

var _ Server = &HTTPServerImpl{}

// NewHTTPServer creates new HTTP Server
func NewHTTPServer(logger common.LoggerIface, config *model.Config, app common.AppIface) Server {
	a := &HTTPServerImpl{
		logger:  logger,
		routers: rest.NewRouter(mux.NewRouter(), app, logger),
		config:  config,
		app:     app,
	}

	pwd, _ := os.Getwd()
	a.logger.Info("HTTP server current working", log.String("directory", pwd))
	return a
}

// Start method starts a http server
func (a *HTTPServerImpl) Start() error {
	a.logger.Info("Starting HTTP Server...")

	a.srv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", a.config.HTTP.Host, a.config.HTTP.Port),
		Handler:      a.routers.Root,
		ReadTimeout:  time.Duration(a.config.HTTP.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(a.config.HTTP.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(a.config.HTTP.IdleTimeout) * time.Second,
	}

	a.logger.Info("HTTP Server is listening on", log.Int("port", a.config.HTTP.Port))
	if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.logger.Error("Failed to listen and serve: %v", log.Err(err))
		return err
	}
	return nil
}

// Shutdown method shuts http server down
func (a *HTTPServerImpl) Shutdown() error {
	a.logger.Info("Stopping HTTP Server...")
	if a.srv == nil {
		return errors.New("No http server present")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := a.srv.Shutdown(ctx); err != nil {
		a.logger.Error("Unable to shutdown server", log.Err(err))
	}

	// a.srv.Close()
	// a.srv = nil
	a.logger.Info("HTTP Server stopped")
	return nil
}
