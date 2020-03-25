package chunk

import (
	"io"
	"os"

	"pcloud/api"
)

type ReadOnlyFileChunk struct {
	f      *os.File
	offset int
	size   int
}

func NewReadOnlyFileChunk(f *os.File, offset, size int) Chunk {
	return &ReadOnlyFileChunk{f, offset, size}
}

func (c *ReadOnlyFileChunk) Stats() (ChunkInfo, error) {
	return ChunkInfo{
		Status:    api.ChunkStatus_READY,
		Size:      c.size,
		Committed: c.size}, nil
}

func (c *ReadOnlyFileChunk) ReaderAt() io.ReaderAt {
	return &fileReader{c.f}
}

func (c *ReadOnlyFileChunk) WriterAt() io.WriterAt {
	return &fileWriter{c.f}
}

type fileReader struct {
	f *os.File
}

func (f *fileReader) ReadAt(b []byte, offset int64) (int, error) {
	return f.f.ReadAt(b, offset)
}

type fileWriter struct {
	f *os.File
}

func (f *fileWriter) WriteAt(b []byte, offset int64) (int, error) {
	return f.f.WriteAt(b, offset)
}
