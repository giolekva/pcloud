package chunk

import "context"
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
	request *api.ListChunksRequest) (*api.ListChunksResponse, error) {
	resp := api.ListChunksResponse{}
	s.chunks.Range(func(k, v interface{}) bool {
		resp.ChunkId = append(resp.ChunkId, k.(string))
		return true
	})
	return &resp, nil
}

func (s *ChunkServer) ReadChunk(
	ctx context.Context,
	request *api.ReadChunkRequest) (*api.ReadChunkResponse, error) {
	if data, ok := s.chunks.Load(request.ChunkId); ok {
		return &api.ReadChunkResponse{Data: data.([]byte)}, nil
	}
	return nil, nil
}

func (s *ChunkServer) StoreChunk(
	ctx context.Context,
	request *api.StoreChunkRequest) (*api.StoreChunkResponse, error) {
	s.chunks.Store(request.ChunkId, request.Data)
	return &api.StoreChunkResponse{}, nil
}
