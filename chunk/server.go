package chunk

import (
	"context"
	"fmt"
	"log"

	"sync"

	"pcloud/api"
)

type ChunkServer struct {
	factory          ChunkFactory
	chunks           sync.Map
	replicatorCancel sync.Map
}

func NewChunkServer(factory ChunkFactory) *ChunkServer {
	return &ChunkServer{factory: factory}
}

func (s *ChunkServer) ListChunks(
	ctx context.Context,
	req *api.ListChunksRequest) (resp *api.ListChunksResponse, err error) {
	resp = &api.ListChunksResponse{}
	s.chunks.Range(func(k, v interface{}) bool {
		resp.ChunkId = append(resp.ChunkId, k.(string))
		return true
	})
	return
}

func (s *ChunkServer) CreateChunk(
	ctx context.Context,
	req *api.CreateChunkRequest) (resp *api.CreateChunkResponse, err error) {
	chunk := s.factory.New(int(req.Size))
	s.chunks.Store(req.ChunkId, chunk)
	switch req.Role {
	case api.ReplicaRole_SECONDARY:
		ctx, cancel := context.WithCancel(context.Background())
		s.replicatorCancel.Store(req.ChunkId, cancel)
		primaryListener := NewNonChangingPrimaryReplicaChangeListener(
			req.ChunkId,
			req.PrimaryAddress)
		go ReplicateFromPrimary(ctx, chunk, primaryListener)
	case api.ReplicaRole_PRIMARY:
		{
		}
	}
	resp = &api.CreateChunkResponse{}
	log.Printf("Created chunk: %s\n", req.ChunkId)
	return

}

func (s *ChunkServer) GetChunkStatus(
	ctx context.Context,
	req *api.GetChunkStatusRequest) (resp *api.GetChunkStatusResponse, err error) {
	if chunk, ok := s.chunks.Load(req.ChunkId); ok {
		c := chunk.(Chunk)
		var info ChunkInfo
		info, err = c.Stats()
		if err != nil {
			return
		}
		resp = &api.GetChunkStatusResponse{
			Status:         info.Status,
			TotalBytes:     int32(info.Size),
			CommittedBytes: int32(info.Committed)}
		return
	}
	return nil, fmt.Errorf("Could not fund chunk: %s", req.ChunkId)
}

func (s *ChunkServer) ReadChunk(
	ctx context.Context,
	req *api.ReadChunkRequest) (resp *api.ReadChunkResponse, err error) {
	if value, ok := s.chunks.Load(req.ChunkId); ok {
		chunk := value.(Chunk)
		b := make([]byte, req.NumBytes)
		var n int
		n, err = chunk.ReaderAt().ReadAt(b, int64(req.Offset))
		if n == 0 {
			return
		}
		return &api.ReadChunkResponse{Data: b[:n]}, nil

	} else {
		return nil, fmt.Errorf("Chunk not found: %s", req.ChunkId)
	}
}

func (s *ChunkServer) WriteChunk(
	ctx context.Context,
	req *api.WriteChunkRequest) (resp *api.WriteChunkResponse, err error) {
	if value, ok := s.chunks.Load(req.ChunkId); ok {
		chunk := value.(Chunk)
		var n int
		n, err = chunk.WriterAt().WriteAt(req.Data, int64(req.Offset))
		if n == 0 {
			return
		}
		return &api.WriteChunkResponse{BytesWritten: int32(n)}, nil

	} else {
		return nil, fmt.Errorf("Chunk not found: %s", req.ChunkId)
	}
}

func (s *ChunkServer) RemoveChunk(
	ctx context.Context,
	req *api.RemoveChunkRequest) (resp *api.RemoveChunkResponse, err error) {
	if cancel, ok := s.replicatorCancel.Load(req.ChunkId); ok {
		cancel.(context.CancelFunc)()
		s.replicatorCancel.Delete(req.ChunkId)
	}
	s.chunks.Delete(req.ChunkId)
	return &api.RemoveChunkResponse{}, nil
}
