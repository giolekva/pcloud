package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/store"
	"github.com/gorilla/mux"
)

// HTTPServerImpl http server implementation
type HTTPServerImpl struct {
	Log    *log.Logger
	srv    *http.Server
	root   *mux.Router
	config *model.Config
	store  store.Store
}

var _ Server = &HTTPServerImpl{}

// NewHTTPServer creates new HTTP Server
func NewHTTPServer(logger *log.Logger, config *model.Config, store store.Store) Server {
	a := &HTTPServerImpl{
		Log:    logger,
		root:   mux.NewRouter(),
		config: config,
		store:  store,
	}

	pwd, _ := os.Getwd()
	a.Log.Info("HTTP server current working", log.String("directory", pwd))
	return a
}

// Start method starts a http server
func (a *HTTPServerImpl) Start() error {
	a.Log.Info("Starting HTTP Server...")

	a.srv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", a.config.HTTPSettings.Host, a.config.HTTPSettings.Port),
		Handler:      a.root,
		ReadTimeout:  time.Duration(a.config.HTTPSettings.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(a.config.HTTPSettings.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(a.config.HTTPSettings.IdleTimeout) * time.Second,
	}

	a.Log.Info("HTTP Server is listening on", log.Int("port", a.config.HTTPSettings.Port))
	if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.Log.Error("failed to listen and serve: %v", log.Err(err))
		return err
	}
	return nil
}

// Shutdown method shuts http server down
func (a *HTTPServerImpl) Shutdown() error {
	a.Log.Info("Stopping HTTP Server...")
	if a.srv == nil {
		return errors.New("no http server present")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := a.srv.Shutdown(ctx); err != nil {
		a.Log.Error("Unable to shutdown server", log.Err(err))
	}

	// a.srv.Close()
	// a.srv = nil
	a.Log.Info("HTTP Server stopped")
	return nil
}
