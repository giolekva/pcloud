package chunk

import "bytes"
import "io"

type Chunk interface {
	SizeBytes() int
	ReadSeeker() io.ReadSeeker
	Writer() io.Writer
}

type InMemoryChunk struct {
	payload *[]byte
}

func NewEmptyInMemoryChunk(sizeBytes int) Chunk {
	payload := make([]byte, sizeBytes)
	return &InMemoryChunk{payload: &payload}
}

func NewInMemoryChunk(p *[]byte) Chunk {
	return &InMemoryChunk{payload: p}
}

func (c *InMemoryChunk) SizeBytes() int {
	return len(*c.payload)
}

func (c *InMemoryChunk) ReadSeeker() io.ReadSeeker {
	return bytes.NewReader(*c.payload)
}

func (c *InMemoryChunk) Writer() io.Writer {
	return bytes.NewBuffer(*c.payload)
}
