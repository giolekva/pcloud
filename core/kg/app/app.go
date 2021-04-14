package app

import (
	"github.com/giolekva/pcloud/core/kg/common"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/store"
)

// App represents an application layer of the kg
type App struct {
	store  store.Store
	config *model.Config
	logger common.LoggerIface
}

// NewApp creates new app
func NewApp(store store.Store, config *model.Config, logger common.LoggerIface) *App {
	return &App{
		store:  store,
		config: config,
		logger: logger,
	}
}
