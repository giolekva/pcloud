package tasks

import (
	"fmt"
	"net/http"
	"time"

	phttp "github.com/giolekva/pcloud/core/installer/http"
)

func waitForAddr(client phttp.Client, addr string) Task {
	t := newLeafTask(fmt.Sprintf("Wait for %s to come up", addr), func() error {
		for {
			if resp, err := client.Get(addr); err != nil || resp.StatusCode != http.StatusOK {
				time.Sleep(2 * time.Second)
			} else {
				return nil
			}
		}
	})
	return &t
}
