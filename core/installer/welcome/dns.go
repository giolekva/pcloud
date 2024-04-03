package welcome

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/libdns/gandi"
	"github.com/libdns/libdns"
)

type DNSUpdater interface {
	Update(zone string, records []string) error
}

func ParseRecords(zone string, records []string) ([]libdns.Record, error) {
	var rrs []libdns.Record
	for _, r := range records {
		if r == "" {
			continue
		}
		fmt.Println(r)
		var name string
		var ttl time.Duration
		var tmp string
		var t string
		l := strings.NewReader(r)
		if _, err := fmt.Fscanf(l, "%s %d %s %s", &name, &ttl, &tmp, &t); err != nil {
			return nil, err
		}
		var value strings.Builder
		if _, err := io.Copy(&value, l); err != nil {
			return nil, err
		}
		val := strings.TrimSpace(value.String())
		fmt.Printf("%s -- %d -- %s -- %s\n", name, ttl, t, val)
		rrs = append(rrs, libdns.Record{
			Type:  t,
			Name:  libdns.RelativeName(name, zone),
			Value: val,
			TTL:   ttl * time.Second,
		})
	}
	return rrs, nil
}

type gandiUpdater struct {
	provider libdns.RecordSetter
}

func NewGandiUpdater(apiToken string) *gandiUpdater {
	return &gandiUpdater{
		provider: &gandi.Provider{BearerToken: apiToken},
	}
}

func (u *gandiUpdater) Update(zone string, records []string) error {
	if rrs, err := ParseRecords(zone, records); err != nil {
		return err
	} else {
		fmt.Printf("%+v\n", rrs)
		_, err := u.provider.SetRecords(context.TODO(), zone, rrs)
		return err
	}
}
