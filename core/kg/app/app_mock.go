package app

import (
	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/store/memory"
)

type MockApp struct {
	*App
}

// NewTestApp creates app for testing
func NewTestApp() *MockApp {
	memStore := memory.New()
	logger := &log.NoOpLogger{}
	config := model.NewConfig()
	a := NewApp(memStore, config, logger)
	return &MockApp{a}
}
