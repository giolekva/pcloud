package chunk

import (
	"bytes"
	"errors"
	"io"

	"pcloud/api"
)

type InMemoryChunk struct {
	status    api.ChunkStatus
	payload   []byte
	committed int
}

func (c *InMemoryChunk) Stats() (ChunkInfo, error) {
	return ChunkInfo{c.status, len(c.payload), c.committed}, nil
}

func (c *InMemoryChunk) ReaderAt() io.ReaderAt {
	return bytes.NewReader(c.payload[:c.committed])
}

func (c *InMemoryChunk) WriterAt() io.WriterAt {
	return &byteWriter{c}
}

type byteWriter struct {
	c *InMemoryChunk
}

func (w *byteWriter) WriteAt(p []byte, offset int64) (n int, err error) {
	if int(offset) > w.c.committed {
		panic(1)
		return 0, errors.New("Gaps are not allowed when writing in chunks")
	}
	if int(offset) < w.c.committed {
		if int(offset)+len(p) <= w.c.committed {
			if bytes.Compare(w.c.payload[int(offset):int(offset)+len(p)], p) != 0 {
				panic(2)
				return 0, errors.New("Can not change contents of allready committed chunk bytes")
			}
			panic(3)
			return len(p), nil
		}
		n = w.c.committed - int(offset)
		p = p[n:]
		offset = int64(w.c.committed)
	}
	if w.c.committed+len(p) > len(w.c.payload) {
		panic(4)
		return 0, errors.New("In memory chunk does not have enough space available")
	}
	n += copy(w.c.payload[w.c.committed:], p)
	w.c.committed += n
	return
}

type InMemoryChunkFactory struct {
}

func (f InMemoryChunkFactory) New(size int) Chunk {
	return &InMemoryChunk{
		status:    api.ChunkStatus_CREATED,
		payload:   make([]byte, size),
		committed: 0}
}
