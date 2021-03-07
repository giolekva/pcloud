package app

import (
	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/store/memory"
)

type MockApp struct {
	*App
}

// NewTestApp creates app for testing
func NewTestApp() *MockApp {
	memStore := memory.New()
	logger := &log.NoOpLogger{}
	a := NewApp(memStore, logger)
	return &MockApp{a}
}
