package main

import (
	"strings"
	"sync"
)

type Log struct {
	l sync.Mutex
	d strings.Builder
}

func (l *Log) Write(p []byte) (n int, err error) {
	l.l.Lock()
	defer l.l.Unlock()
	// TODO(gio): Reset s.logs periodically
	return l.d.Write(p)
}

func (l *Log) Contents() string {
	l.l.Lock()
	defer l.l.Unlock()
	return l.d.String()
}
