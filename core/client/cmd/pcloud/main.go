package main

import (
	"flag"
	"fmt"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

var vpnApiAddr = flag.String("vpn-api-addr", "https://vpn.lekva.me", "VPN API server address")

func processUIEvents(a App, events []UIEvent) error {
	for _, e := range events {
		switch e.(type) {
		case EventScanBarcode:
			return a.LaunchBarcodeScanner()
		default:
			return fmt.Errorf("Unhandled event: %#v", e)
		}
	}
	return nil
}

func run() error {
	a := createApp()
	vc := NewDirectVPNClient(*vpnApiAddr)
	ui := NewUI(vc)
	w := app.NewWindow(
		app.Size(unit.Px(1500), unit.Px(1500)),
		app.Title("PCloud"),
	)
	var ops op.Ops
	for {
		select {
		case e := <-w.Events():
			switch e := e.(type) {
			case app.ViewEvent:
				if err := a.OnView(e); err != nil {
					return err
				} else {
					w.Invalidate()
				}
			case *system.CommandEvent:
				if e.Type == system.CommandBack {
					if ui.OnBack() {
						e.Cancel = true
						w.Invalidate()
					}
				}
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				events := ui.Layout(gtx)
				e.Frame(&ops)
				if err := processUIEvents(a, events); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func main() {
	flag.Parse()
	go func() {
		if err := run(); err != nil {
			panic(err)
		}
	}()
	app.Main()
}
