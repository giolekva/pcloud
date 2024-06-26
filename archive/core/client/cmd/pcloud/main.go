package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"errors"
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

type ScanQRFor int

const (
	ScanQRForJoin    ScanQRFor = 0
	ScanQRForApprove           = 1
)

type processor struct {
	vc  VPNClient
	app App
	st  Storage
	ui  *UI

	scanQRFor   ScanQRFor
	inviteQrCh  chan image.Image
	qrScannedCh chan []byte
	joinQrCh    chan image.Image

	onConnectCh    chan interface{}
	onDisconnectCh chan interface{}

	onConfigCh chan struct{}
}

func newProcessor() *processor {
	th := material.NewTheme(gofont.Collection())
	app := createApp()
	return &processor{
		vc:             NewDirectVPNClient(*vpnApiAddr),
		app:            app,
		st:             app.CreateStorage(),
		ui:             NewUI(th, app.Capabilities()),
		inviteQrCh:     make(chan image.Image, 1),
		qrScannedCh:    make(chan []byte, 1),
		joinQrCh:       make(chan image.Image, 1),
		onConnectCh:    make(chan interface{}, 1),
		onDisconnectCh: make(chan interface{}, 1),
		onConfigCh:     make(chan struct{}, 1),
	}
}

func (p *processor) QRCodeScanned(code []byte) {
	p.qrScannedCh <- code
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

func (p *processor) generatePublicPrivateKey() error {
	config, err := p.st.Get()
	if err != nil {
		return err
	}
	if config.Enc.PrivateKey != nil {
		return nil
	}
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	config.Enc.PrivateKey = x509.MarshalPKCS1PrivateKey(privKey)
	config.Enc.PublicKey = x509.MarshalPKCS1PublicKey(&privKey.PublicKey)
	config.Network.PublicKey, config.Network.PrivateKey, err = x25519Keypair()
	if err != nil {
		return err
	}
	return p.st.Store(config)
}

func (p *processor) run() error {
	if err := p.generatePublicPrivateKey(); err != nil {
		return err
	}
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
					if config.Network.Config != nil {
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
		case code := <-p.qrScannedCh:
			switch p.scanQRFor {
			case ScanQRForJoin:
				if err := p.JoinAndGetNetworkConfig(code); err != nil {
					return err
				}
			case ScanQRForApprove:
				if err := p.ApproveOther(code); err != nil {
					return err
				}
			default:
				return errors.New("Must not reach!")
			}
		case img := <-p.joinQrCh:
			p.ui.JoinQRGenerated(img)
			w.Invalidate()
			go func() {
				cnt := 0
				for {
					fmt.Println(cnt)
					cnt++
					if cnt > 5 {
						break
					}
					time.Sleep(time.Second)
					config, err := p.st.Get()
					if err != nil {
						continue
					}
					privKey, err := x509.ParsePKCS1PrivateKey(config.Enc.PrivateKey)
					if err != nil {
						continue
					}
					hostname, err := p.app.GetHostname()
					if err != nil {
						continue
					}
					hostname = sanitizeHostname(hostname)
					network, err := p.vc.Get("https://vpn.lekva.me", hostname, privKey, config.Network.PrivateKey)
					if err != nil {
						continue
					}
					config.ApiAddr = "https://vpn.lekva.me"
					config.Network.Config = network
					if err := p.st.Store(config); err != nil {
						continue
					}
					p.onConfigCh <- struct{}{}
					break
				}
			}()
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
		case EventGetJoinQRCode:
			go func() {
				if img, err := p.generateJoinQRCode(); err == nil {
					p.joinQrCh <- img
				} else {
					// TODO(giolekva): do not panic
					panic(err)
				}
			}()
		case EventApproveOther:
			p.scanQRFor = ScanQRForApprove
			if err := p.app.LaunchBarcodeScanner(); err != nil {
				return err
			}
		case EventScanBarcode:
			p.scanQRFor = ScanQRForJoin
			if err := p.app.LaunchBarcodeScanner(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unhandled event: %#v", e)
		}
	}
	return nil
}

type inviteQrCodeData struct {
	VPNApiAddr string `json:"vpn_api_addr"`
	Message    []byte `json:"message"`
	Signature  []byte `json:"signature"`
}

type joinQrCodeData struct {
	EncPublicKey []byte `json:"enc_public_key"`
	Name         string `json:"name"`
	NetPublicKey []byte `json:"net_public_key"`
	IPCidr       string `json:"ip_cidr"`
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
	c := inviteQrCodeData{
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

func (p *processor) generateJoinQRCode() (image.Image, error) {
	config, err := p.st.Get()
	if err != nil {
		return nil, err
	}
	hostname, err := p.app.GetHostname()
	if err != nil {
		return nil, err
	}
	hostname = sanitizeHostname(hostname)
	c := joinQrCodeData{
		EncPublicKey: config.Enc.PublicKey,
		Name:         hostname,
		NetPublicKey: config.Network.PublicKey,
		IPCidr:       "111.0.0.14/24",
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
	config, err := p.st.Get()
	if err != nil {
		return err
	}
	var invite inviteQrCodeData
	if err := json.NewDecoder(bytes.NewReader(code)).Decode(&invite); err != nil {
		return err
	}
	hostname, err := p.app.GetHostname()
	if err != nil {
		return err
	}
	hostname = sanitizeHostname(hostname)
	network, err := p.vc.Join(invite.VPNApiAddr, hostname, config.Network.PublicKey, config.Network.PrivateKey, invite.Message, invite.Signature)
	if err != nil {
		return err
	}
	config.ApiAddr = invite.VPNApiAddr
	config.Network.Config = network
	if err := p.st.Store(config); err != nil {
		return err
	}
	p.onConfigCh <- struct{}{}
	return nil
}

func (p *processor) ApproveOther(code []byte) error {
	config, err := p.st.Get()
	if err != nil {
		return err
	}
	var approve joinQrCodeData
	if err := json.NewDecoder(bytes.NewReader(code)).Decode(&approve); err != nil {
		return err
	}
	return p.vc.Approve(config.ApiAddr, approve.Name, approve.IPCidr, approve.EncPublicKey, approve.NetPublicKey)
}

func sanitizeHostname(hostname string) string {
	return strings.ToLower(
		strings.ReplaceAll(
			strings.ReplaceAll(hostname, " ", "-"),
			".", "-"))
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
