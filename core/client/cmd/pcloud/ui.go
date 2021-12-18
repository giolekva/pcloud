package main

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type UI struct {
	th     *material.Theme
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

func NewUI(th *material.Theme) *UI {
	return &UI{th: th}
}

func (ui *UI) InviteQRGenerated(img image.Image) {
	ui.invite.qr = img
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
		events = append(events, EventGetInviteQRCode{})
	} else if ui.join.open.Clicked() {
		// ui.invite.show = false
		// ui.join.show = true
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
			return material.Button(ui.th, &ui.invite.open, "Invite").Layout(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			return material.Button(ui.th, &ui.join.open, "Join").Layout(gtx)
		}),
	)
}

func (ui *UI) layoutInvite(gtx C) D {
	if ui.invite.qr == nil {
		return ColorBox(gtx, gtx.Constraints.Max, color.NRGBA{})
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
