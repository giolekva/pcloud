package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/skip2/go-qrcode"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type UI struct {
	vc VPNClient

	invite struct {
		open widget.Clickable
		show bool
		qr   image.Image
	}

	join struct {
		open   widget.Clickable
		show   bool
		qrcode string
	}
}

func NewUI(vc VPNClient) *UI {
	return &UI{
		vc: vc,
	}
}

func (ui *UI) OnBack() bool {
	if ui.invite.show {
		ui.invite.show = false
		return true
	} else if ui.join.show {
		ui.join.show = false
		return true
	}
	return false
}

func (ui *UI) Layout(gtx C) []UIEvent {
	var events []UIEvent
	if ui.invite.open.Clicked() {
		ui.join.show = false
		ui.invite.show = true
		ui.invite.qr = nil
	} else if ui.join.open.Clicked() {
		ui.invite.show = false
		ui.join.show = true
		events = append(events, EventScanBarcode{})
	}
	if ui.invite.show {
		ui.layout(gtx, ui.layoutInvite)
	} else if ui.join.show {
		ui.layout(gtx, ui.layoutJoin)
	} else {
		ui.layout(gtx, ui.layoutMain)
	}
	return events
}

func (ui *UI) layout(gtx C, mainPanel layout.Widget) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(10, func(gtx C) D {
			return mainPanel(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			return ui.layoutActions(gtx)
		}),
	)
}

func ColorBox(gtx layout.Context, size image.Point, color color.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: size}
}

func (ui *UI) layoutMain(gtx C) D {
	return ColorBox(gtx, gtx.Constraints.Min, color.NRGBA{B: 255, A: 255})
}

func (ui *UI) layoutActions(gtx C) D {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			return ui.invite.open.Layout(gtx, func(gtx C) D {
				return ColorBox(gtx, gtx.Constraints.Min, color.NRGBA{R: 255, A: 255})
			})

		}),
		layout.Flexed(1, func(gtx C) D {
			return ui.join.open.Layout(gtx, func(gtx C) D {
				return ColorBox(gtx, gtx.Constraints.Min, color.NRGBA{G: 128, A: 255})
			})
		}),
	)
}

func (ui *UI) layoutInvite(gtx C) D {
	if ui.invite.qr == nil {
		img, err := prepareConfigQRCode(ui.vc)
		if err != nil {
			panic(err)
		}
		ui.invite.qr = img
	}
	d := ui.invite.qr.Bounds().Max.Sub(ui.invite.qr.Bounds().Min)
	return layout.Inset{
		Left: unit.Px(0.5 * float32(gtx.Constraints.Max.X-d.X)),
		Top:  unit.Px(0.5 * float32(gtx.Constraints.Max.Y-d.Y)),
	}.Layout(gtx, func(gtx C) D {
		paint.NewImageOp(ui.invite.qr).Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		return D{Size: gtx.Constraints.Max}
	})
}

func (ui *UI) layoutJoin(gtx C) D {
	if ui.join.qrcode == "" {
		return ColorBox(gtx, gtx.Constraints.Min, color.NRGBA{R: 255, A: 255})
	}
	return ColorBox(gtx, gtx.Constraints.Min, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
}

// helpers

type qrCodeData struct {
	VPNApiAddr string `json:"vpn_api_addr"`
	Message    []byte `json:"message"`
	Signature  []byte `json:"signature"`
}

func prepareConfigQRCode(vc VPNClient) (image.Image, error) {
	message := []byte("Hello PCloud")
	signature, err := vc.Sign(message)
	if err != nil {
		return nil, err
	}
	c := qrCodeData{
		vc.Address(),
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
