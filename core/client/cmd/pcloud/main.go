package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/skip2/go-qrcode"
)

var vpnApiAddr = flag.String("vpn-api-addr", "", "VPN API server address")

var p *processor

type processor struct {
	vc  VPNClient
	app App
	st  Storage
	ui  *UI

	inviteQrCh        chan image.Image
	inviteQrScannedCh chan []byte

	onConnectCh    chan interface{}
	onDisconnectCh chan interface{}

	onConfigCh chan struct{}
}

func newProcessor() *processor {
	th := material.NewTheme(gofont.Collection())
	app := createApp()
	return &processor{
		vc:                NewDirectVPNClient(*vpnApiAddr),
		app:               app,
		st:                app.CreateStorage(),
		ui:                NewUI(th),
		inviteQrCh:        make(chan image.Image, 1),
		inviteQrScannedCh: make(chan []byte, 1),
		onConnectCh:       make(chan interface{}, 1),
		onDisconnectCh:    make(chan interface{}, 1),
		onConfigCh:        make(chan struct{}, 1),
	}
}

func (p *processor) InviteQRCodeScanned(code []byte) {
	p.inviteQrScannedCh <- code
}

func (p *processor) ConnectRequested(service interface{}) {
	go func() {
		time.Sleep(1 * time.Second)
		p.onConnectCh <- service
	}()
}

func (p *processor) DisconnectRequested(service interface{}) {
	p.onDisconnectCh <- service
}

func (p *processor) run() error {
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
				if err := p.app.OnView(e); err != nil {
					return err
				} else {
					w.Invalidate()
				}
				if config, err := p.st.Get(); err != nil {
					return err
				} else {
					if config.Network != nil {
						p.onConfigCh <- struct{}{}
					}
				}
			case *system.CommandEvent:
				if e.Type == system.CommandBack {
					if p.ui.OnBack() {
						e.Cancel = true
						w.Invalidate()
					}
				}
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				events := p.ui.Layout(gtx)
				e.Frame(&ops)
				if err := p.processUIEvents(events); err != nil {
					return err
				}
			}
		case img := <-p.inviteQrCh:
			p.ui.InviteQRGenerated(img)
			w.Invalidate()
		case code := <-p.inviteQrScannedCh:
			if err := p.JoinAndGetNetworkConfig(code); err != nil {
				return err
			}
		case <-p.onConfigCh:
			if err := p.app.TriggerService(); err != nil {
				return err
			}
		case s := <-p.onConnectCh:
			if err := p.app.UpdateService(s); err != nil {
				return err
			}
			if config, err := p.st.Get(); err != nil {
				return err
			} else {
				if err := p.app.Connect(config); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (p *processor) processUIEvents(events []UIEvent) error {
	for _, e := range events {
		switch e.(type) {
		case EventGetInviteQRCode:
			go func() {
				if img, err := p.generateInviteQRCode(); err == nil {
					p.inviteQrCh <- img
				} else {
					// TODO(giolekva): do not panic
					panic(err)
				}
			}()
		case EventScanBarcode:
			return p.app.LaunchBarcodeScanner()
		default:
			return fmt.Errorf("Unhandled event: %#v", e)
		}
	}
	return nil
}

type qrCodeData struct {
	VPNApiAddr string `json:"vpn_api_addr"`
	Message    []byte `json:"message"`
	Signature  []byte `json:"signature"`
}

func (p *processor) generateInviteQRCode() (image.Image, error) {
	config, err := p.st.Get()
	if err != nil {
		return nil, err
	}
	message := []byte("Hello PCloud")
	signature, err := p.vc.Sign(config.ApiAddr, message)
	if err != nil {
		return nil, err
	}
	c := qrCodeData{
		config.ApiAddr,
		message,
		signature,
	}
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(c); err != nil {
		return nil, err
	}
	qr, err := qrcode.Encode(data.String(), qrcode.Medium, 1024)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(qr))
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (p *processor) JoinAndGetNetworkConfig(code []byte) error {
	var invite qrCodeData
	if err := json.NewDecoder(bytes.NewReader(code)).Decode(&invite); err != nil {
		return err
	}
	hostname, err := p.app.GetHostname()
	if err != nil {
		return err
	}
	hostname = strings.ToLower(strings.ReplaceAll(hostname, " ", "-"))
	fmt.Printf("------ %s\n", hostname)
	network, err := p.vc.Join(invite.VPNApiAddr, hostname, invite.Message, invite.Signature)
	if err != nil {
		return err
	}
	if err := p.st.Store(Config{invite.VPNApiAddr, network}); err != nil {
		return err
	}
	p.onConfigCh <- struct{}{}
	return nil
}

func main() {
	flag.Parse()
	p = newProcessor()
	go func() {
		if err := p.run(); err != nil {
			panic(err)
		}
	}()
	app.Main()
}
