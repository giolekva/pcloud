package main

import "gioui.org/app"

type App interface {
	LaunchBarcodeScanner() error
	OnView(app.ViewEvent) error
}
