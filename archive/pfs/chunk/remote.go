package chunk

import (
	"context"
	"io"

	"github.com/giolekva/pcloud/pfs/api"
)

type RemoteChunk struct {
	chunkId string
	client  api.ChunkStorageClient
}

func (r *RemoteChunk) Stats() (info ChunkInfo, err error) {
	resp, err := r.client.GetChunkStatus(
		context.Background(),
		&api.GetChunkStatusRequest{ChunkId: r.chunkId})
	if err != nil {
		return
	}
	info = ChunkInfo{
		resp.Status,
		int(resp.TotalBytes),
		int(resp.CommittedBytes)}
	return
}

func (r *RemoteChunk) ReaderAt() io.ReaderAt {
	return &remoteChunkReaderAt{
		chunkId: r.chunkId,
		client:  r.client}
}

func (r *RemoteChunk) WriterAt() io.WriterAt {
	return &remoteChunkWriterAt{
		chunkId: r.chunkId,
		client:  r.client}
}

type remoteChunkReaderAt struct {
	chunkId string
	client  api.ChunkStorageClient
}

func (c *remoteChunkReaderAt) ReadAt(p []byte, offset int64) (n int, err error) {
	req := api.ReadChunkRequest{
		ChunkId:  c.chunkId,
		Offset:   int32(offset),
		NumBytes: int32(len(p))}
	resp, err := c.client.ReadChunk(context.Background(), &req)
	if err != nil {
		return
	}
	n = copy(p, resp.Data)
	return
}

type remoteChunkWriterAt struct {
	chunkId string
	client  api.ChunkStorageClient
}

func (c *remoteChunkWriterAt) WriteAt(p []byte, offset int64) (n int, err error) {
	req := api.WriteChunkRequest{
		ChunkId: c.chunkId,
		Offset:  int32(offset),
		Data:    p}
	resp, err := c.client.WriteChunk(context.Background(), &req)
	if resp != nil {
		n = int(resp.BytesWritten)
	}
	return
}
