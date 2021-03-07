package app

import (
	"github.com/giolekva/pcloud/core/kg/store"
)

// App represents an application layer of the kg
type App struct {
	store  store.Store
	logger logger
}

// NewApp creates new app
func NewApp(store store.Store, logger logger) *App {
	return &App{
		store:  store,
		logger: logger,
	}
}
