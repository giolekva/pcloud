package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

type RecordsFile struct {
	lock sync.Locker
	rrs  []dns.RR
}

func NewRecordsFile(r io.Reader) (*RecordsFile, error) {
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
	return &RecordsFile{&sync.Mutex{}, rrs}, nil
}

func (z *RecordsFile) DeleteTxtRecord(name, value string) {
	z.lock.Lock()
	defer z.lock.Unlock()
	fmt.Printf("%s %s\n", name, value)
	for i, rr := range z.rrs {
		fmt.Printf("%+v\n", rr)
		if txt, ok := rr.(*dns.TXT); ok {
			fmt.Printf("%+v\n", txt)
			if txt.Hdr.Name == name && strings.Join(txt.Txt, "") == value {
				z.rrs = append(z.rrs[:i], z.rrs[i+1:]...)
			}
		}
	}
}

// func (z *RecordsFile) DeleteRecordsFor(name string) {
// 	z.lock.Lock()
// 	defer z.lock.Unlock()
// 	rrs := make([]dns.RR, 0)
// 	for _, rr := range z.rrs {
// 		if rr.Header().Name != name {
// 			rrs = append(rrs, rr)
// 		}
// 	}
// 	z.rrs = rrs
// }

func (z *RecordsFile) CreateOrReplaceTxtRecord(name, value string) {
	z.lock.Lock()
	defer z.lock.Unlock()
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

func (z *RecordsFile) CreateARecord(name, value string) {
	z.lock.Lock()
	defer z.lock.Unlock()
	z.rrs = append(z.rrs, &dns.A{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		A: net.ParseIP(value),
	})
}

func (z *RecordsFile) Write(w io.Writer) error {
	z.lock.Lock()
	defer z.lock.Unlock()
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
