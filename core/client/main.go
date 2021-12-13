package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"image"
	"image/png"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	qrcode "github.com/skip2/go-qrcode"
)

var vpnApiAddr = flag.String("vpn-api-addr", "", "VPN API server address")

type config struct {
	VPNApiAddr string `json:"vpn_api_addr"`
	Message    []byte `json:"message"`
	Signature  []byte `json:"signature"`
}

func prepareConfigQRCode(vc *VPNApiClient) (*image.Image, error) {
	message := []byte("Hello PCloud")
	signature, err := vc.Sign(message)
	if err != nil {
		return nil, err
	}
	c := config{
		*vpnApiAddr,
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
	return &img, nil
}

func main() {
	flag.Parse()
	go func() {
		vc := &VPNApiClient{
			*vpnApiAddr,
		}
		img, err := prepareConfigQRCode(vc)
		if err != nil {
			panic(err)
		}
		w := app.NewWindow(
			app.Size(unit.Px(1024), unit.Px(1024)),
			app.Title("PCloud"),
		)
		ops := new(op.Ops)
		for e := range w.Events() {
			switch e := e.(type) {
			case system.FrameEvent:
				ops.Reset()
				paint.NewImageOp(*img).Add(ops)
				paint.PaintOp{}.Add(ops)
				e.Frame(ops)
			}
		}
	}()
	app.Main()
}
