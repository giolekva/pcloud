package chunk

import (
	"io"

	"github.com/giolekva/pcloud/api"
)

type ChunkInfo struct {
	Status    api.ChunkStatus
	Size      int
	Committed int
}

type Chunk interface {
	Stats() (ChunkInfo, error)
	ReaderAt() io.ReaderAt
	WriterAt() io.WriterAt
}

type ChunkFactory interface {
	New(size int) Chunk
}
