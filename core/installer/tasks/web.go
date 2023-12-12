package tasks

import (
	"fmt"
	"net/http"
	"time"
)

func waitForAddr(addr string) Task {
	t := newLeafTask(fmt.Sprintf("Wait for %s to come up", addr), func() error {
		for {
			if resp, err := http.Get(addr); err != nil || resp.StatusCode != http.StatusOK {
				time.Sleep(2 * time.Second)
			} else {
				return nil
			}
		}
	})
	return &t
}
