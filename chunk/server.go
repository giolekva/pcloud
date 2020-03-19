package chunk

import "bytes"
import "context"
import "io"
import "sync"

import "pcloud/api"

type ChunkServer struct {
	chunks sync.Map
}

func NewChunkServer() *ChunkServer {
	return &ChunkServer{}
}

func (s *ChunkServer) ListChunks(
	ctx context.Context,
	req *api.ListChunksRequest) (*api.ListChunksResponse, error) {
	resp := api.ListChunksResponse{}
	s.chunks.Range(func(k, v interface{}) bool {
		resp.ChunkId = append(resp.ChunkId, k.(string))
		return true
	})
	return &resp, nil
}

func (s *ChunkServer) ReadChunk(
	ctx context.Context,
	req *api.ReadChunkRequest) (resp *api.ReadChunkResponse, err error) {
	if value, ok := s.chunks.Load(req.ChunkId); ok {
		chunk := value.(Chunk)
		src := chunk.ReadSeeker()
		if req.Offset != 0 {
			_, err = src.Seek(int64(req.Offset), io.SeekStart)
			if err != nil {
				return
			}
		}
		var dst bytes.Buffer
		if req.NumBytes != 0 {
			_, err = io.CopyN(&dst, src, int64(req.NumBytes))
		} else {
			_, err = io.Copy(&dst, src)
		}
		if err == nil {
			resp = &api.ReadChunkResponse{Data: dst.Bytes()}
		}
	}
	return
}

func (s *ChunkServer) StoreChunk(
	ctx context.Context,
	req *api.StoreChunkRequest) (*api.StoreChunkResponse, error) {
	data := req.Data
	chunk := NewInMemoryChunk(&data)
	s.chunks.Store(req.ChunkId, chunk)
	return &api.StoreChunkResponse{}, nil
}
