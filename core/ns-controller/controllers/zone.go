package controllers

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type ZoneFile struct {
	rrs []dns.RR
}

func NewZoneFile(r io.Reader) (*ZoneFile, error) {
	rrs := make([]dns.RR, 0)
	p := dns.NewZoneParser(r, "", "")
	p.SetIncludeAllowed(false)
	for {
		if rr, ok := p.Next(); ok {
			rrs = append(rrs, rr)
		} else {
			if err := p.Err(); err != nil {
				return nil, err
			}
			break
		}
	}
	return &ZoneFile{rrs}, nil
}

func (z *ZoneFile) DeleteTxtRecord(name, value string) {
	for i, rr := range z.rrs {
		if txt, ok := rr.(*dns.TXT); ok {
			if txt.Hdr.Name == name && strings.Join(txt.Txt, "") == value {
				z.rrs = append(z.rrs[:i], z.rrs[i+1:]...)
			}
		}
	}
}

func (z *ZoneFile) CreateOrReplaceTxtRecord(name, value string) {
	for i, rr := range z.rrs {
		if txt, ok := rr.(*dns.TXT); ok {
			if txt.Hdr.Name == name && strings.Join(txt.Txt, "") == value {
				txt.Txt = []string{value}
				z.rrs = append(z.rrs[:i], z.rrs[i+1:]...)
				z.rrs = append(z.rrs, txt)
				return
			}
		}
	}
	z.rrs = append(z.rrs, &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		Txt: []string{value},
	})
}

func (z *ZoneFile) Write(w io.Writer) error {
	for _, rr := range z.rrs {
		if soa, ok := rr.(*dns.SOA); ok {
			soa.Serial = NowUnix()
		}
		if _, err := fmt.Fprintf(w, "%s\n", rr.String()); err != nil {
			return err
		}
	}
	return nil
}

// TODO(gio): not going to work in 15 years?
// TODO(gio): remove 10 *
func NowUnix() uint32 {
	return 10 * uint32(time.Now().Unix())
}
