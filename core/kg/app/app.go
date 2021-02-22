package app

import (
	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/store"
)

// App represents an application layer of the kg
type App struct {
	store  store.Store
	logger *log.Logger
}

// NewApp creates new app
func NewApp(store store.Store, logger *log.Logger) *App {
	return &App{
		store:  store,
		logger: logger,
	}
}
