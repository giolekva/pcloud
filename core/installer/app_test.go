package installer

import (
	"testing"
)

func TestHeadscaleUser(t *testing.T) {
	r := NewInMemoryAppRepository(CreateAllApps())
	a, err := r.Find("headscale-user")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("returned app is nil")
	}
}
