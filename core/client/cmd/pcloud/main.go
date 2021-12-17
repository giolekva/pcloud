package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/skip2/go-qrcode"
)

var vpnApiAddr = flag.String("vpn-api-addr", "", "VPN API server address")

var p *processor

type processor struct {
	vc  VPNClient
	app App
	ui  *UI

	inviteQrCh        chan image.Image
	inviteQrScannedCh chan []byte
}

func newProcessor() *processor {
	return &processor{
		vc:                NewDirectVPNClient(*vpnApiAddr),
		app:               createApp(),
		ui:                NewUI(),
		inviteQrCh:        make(chan image.Image, 1),
		inviteQrScannedCh: make(chan []byte, 1),
	}
}

func (p *processor) InviteQRCodeScanned(code []byte) {
	p.inviteQrScannedCh <- code
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
			go func() {
				p.JoinNetworkAndConnect(code)
			}()
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
	message := []byte("Hello PCloud")
	signature, err := p.vc.Sign(message)
	if err != nil {
		return nil, err
	}
	c := qrCodeData{
		p.vc.Address(),
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

func (p *processor) JoinNetworkAndConnect(code []byte) {
	var invite qrCodeData
	if err := json.NewDecoder(bytes.NewReader(code)).Decode(&invite); err != nil {
		panic(err)
	}
	config, err := p.vc.Join(invite.VPNApiAddr, invite.Message, invite.Signature)
	if err != nil {
		panic(err)
	}
	fmt.Printf("-- VPN CONFIG %s\n", string(config))

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

// fmt.Println(m["pki"])
// c := nc.NewC(logrus.StandardLogger())
// if err := c.LoadString(string(tmpl)); err != nil {
// 	return nil, err
// }
// fmt.Println(c.Settings["pki"])
// return c, nil
