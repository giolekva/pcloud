//go:build darwin && !ios
// +build darwin,!ios

package main

import (
	"errors"
	"os"

	"gioui.org/app"
	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula"
	nc "github.com/slackhq/nebula/config"
)

type macosApp struct {
	ctrl *nebula.Control
}

func createApp() App {
	return &macosApp{}
}

func (a *macosApp) Capabilities() DeviceCapabilities {
	return DeviceCapabilities{
		HasCamera: false,
	}
}

func (a *macosApp) LaunchBarcodeScanner() error {
	return errors.New("no camera")
}

func (a *macosApp) OnView(e app.ViewEvent) error {
	return nil
}

func (a *macosApp) Connect(config Config) error {
	if config.Network.Config == nil {
		return nil
	}
	nebulaConfig := nc.NewC(logrus.StandardLogger())
	if err := nebulaConfig.LoadString(string(config.Network.Config)); err != nil {
		return err
	}
	ctrl, err := nebula.Main(nebulaConfig, false, "pcloud", logrus.StandardLogger(), nil)
	if err != nil {
		return err
	}
	ctrl.Start()
	a.ctrl = ctrl
	return nil
}

func (a *macosApp) UpdateService(serv interface{}) error {
	return nil
}

func (a *macosApp) TriggerService() error {
	p.ConnectRequested(nil)
	return nil
}

func (a *macosApp) CreateStorage() Storage {
	return CreateStorage()
}

func (a *macosApp) GetHostname() (string, error) {
	return os.Hostname()
}
