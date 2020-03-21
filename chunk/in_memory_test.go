package chunk

import (
	"bytes"
	"testing"
)

func TestConcurrentReads(t *testing.T) {
	c := InMemoryChunkFactory{}.New()
	if _, err := c.Writer().Write([]byte("abcd")); err != nil {
		panic(err)
	}
	d1 := make([]byte, 2)
	d2 := make([]byte, 3)
	if _, err := c.ReadSeeker().Read(d1); err != nil {
		t.Error(err)
	}
	if bytes.Compare(d1, []byte("ab")) != 0 {
		t.Errorf("Expected: %s\nActual: %s", "ab", d1)
	}
	if _, err := c.ReadSeeker().Read(d2); err != nil {
		t.Error(err)
	}
	if bytes.Compare(d2, []byte("abc")) != 0 {
		t.Errorf("Expected: %s\nActual: %s", "abc", d2)
	}
}
