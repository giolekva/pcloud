package main

import (
	"errors"

	"gioui.org/app"
)

type darwinApp struct {
}

func createApp() App {
	return &darwinApp{}
}

func (a *darwinApp) LaunchBarcodeScanner() error {
	return errors.New("no camera")
}

func (a *darwinApp) OnView(e app.ViewEvent) error {
	return nil
}
