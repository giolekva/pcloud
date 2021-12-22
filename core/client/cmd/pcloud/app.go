package main

import "gioui.org/app"

type App interface {
	Capabilities() DeviceCapabilities
	LaunchBarcodeScanner() error
	OnView(app.ViewEvent) error
	UpdateService(service interface{}) error
	TriggerService() error
	Connect(config Config) error
	CreateStorage() Storage
	GetHostname() (string, error)
}
