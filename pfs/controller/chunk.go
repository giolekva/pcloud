package controller

import (
	"github.com/giolekva/pcloud/pfs/api"
)

type chunkServerStatus int

const (
	Healthy chunkServerStatus = iota
	UNREACHABLE
)

type chunkServer struct {
	address string
	status  chunkServerStatus
	chunks  map[string]api.ChunkStatus
}
