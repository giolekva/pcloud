package tasks

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestGoogle(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	d := NewDNSResolverTask(
		"welcome.t5.lekva.me",
		[]net.IP{
			net.ParseIP("135.181.48.180"),
			net.ParseIP("65.108.39.172"),
		},
		ctx,
		t.Logf,
	)
	d.FinalizeSubtasks()
	ch := make(chan struct{})
	d.OnDone(func(err error) {
		if err != nil {
			t.Logf("%s\n", err.Error())
		} else {
			t.Logf("Dooone")
		}
		ch <- struct{}{}
	})
	d.Start()
	<-ch
}
