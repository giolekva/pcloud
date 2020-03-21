package chunk

import "context"
import "errors"
import "io"

import "pcloud/api"

type RemoteChunk struct {
	chunkId string
	client  api.ChunkStorageClient
}

func (r *RemoteChunk) SizeBytes() int {
	return 0
}

func (r *RemoteChunk) ReadSeeker() io.ReadSeeker {
	return &remoteChunkReadSeeker{
		chunkId: r.chunkId,
		client:  r.client}
}

func (r *RemoteChunk) Writer() io.Writer {
	return nil
}

type remoteChunkReadSeeker struct {
	chunkId string
	client  api.ChunkStorageClient
	offset  int64
}

func (c *remoteChunkReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if whence != io.SeekStart {
		return 0, errors.New("Seek: RemoteChunk only supports SeekStart whence")
	}
	c.offset = offset
	return offset, nil
}

func (c *remoteChunkReadSeeker) Read(p []byte) (n int, err error) {
	req := api.ReadChunkRequest{
		ChunkId:  c.chunkId,
		Offset:   int32(c.offset), // TODO(lekva): must be int64
		NumBytes: int32(len(p))}
	resp, err := c.client.ReadChunk(context.Background(), &req)
	if err != nil {
		return
	}
	n = copy(p, resp.Data)
	c.offset += int64(n)
	return
}

type PrimaryReplicaChunk struct {
	chunkId string
}
