package chunk

import "io"

type Chunk interface {
	SizeBytes() int
	ReadSeeker() io.ReadSeeker
	Writer() io.Writer
}

type ChunkFactory interface {
	New() Chunk
}
