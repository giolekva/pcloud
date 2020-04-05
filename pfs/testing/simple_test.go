package testing

import (
	"testing"
)

func TestSetup(t *testing.T) {
	env, err := NewInMemoryEnv(3)
	if err != nil {
		t.Error(err)
	}
	defer env.Stop()
}
