package chunk

import (
	"bytes"
	"io"
)

type InMemoryChunk struct {
	payload bytes.Buffer
}

func (c *InMemoryChunk) SizeBytes() int {
	return len(c.payload.Bytes())
}

func (c *InMemoryChunk) ReadSeeker() io.ReadSeeker {
	return bytes.NewReader(c.payload.Bytes())
}

func (c *InMemoryChunk) Writer() io.Writer {
	return &c.payload
}

type InMemoryChunkFactory struct {
}

func (f InMemoryChunkFactory) New() Chunk {
	return &InMemoryChunk{}
}
