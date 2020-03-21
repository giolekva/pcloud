package chunk

import "bytes"
import "context"
import "io"
import "sync"

import "pcloud/api"

type ChunkServer struct {
	factory          ChunkFactory
	chunks           sync.Map
	replicatorCancel sync.Map
}

func NewChunkServer() *ChunkServer {
	return &ChunkServer{}
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
	req *api.StoreChunkRequest) (resp *api.StoreChunkResponse, err error) {
	chunk := s.factory.New()
	_, err = chunk.Writer().Write(req.Data)
	s.chunks.Store(req.ChunkId, chunk)
	if err == nil {
		resp = &api.StoreChunkResponse{}
	}
	return
}

func (s *ChunkServer) ReplicateChunk(
	ctx context.Context,
	req *api.ReplicateChunkRequest) (resp *api.ReplicateChunkResponse, err error) {
	chunk := s.factory.New()
	s.chunks.Store(req.ChunkId, chunk)
	ctx, cancel := context.WithCancel(context.Background())
	s.replicatorCancel.Store(req.ChunkId, cancel)
	go replicateFromPrimary(ctx, chunk, NonChangingPrimaryReplicaChangeListener{req.ChunkId, req.PrimaryChunkServer})
	resp = &api.ReplicateChunkResponse{}
	return

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
