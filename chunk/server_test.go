package chunk

import (
	"bytes"
	"context"
	"testing"

	"pcloud/api"
)

func TestStoreChunk(t *testing.T) {
	s := ChunkServer{factory: &InMemoryChunkFactory{}}
	_, err := s.CreateChunk(context.Background(), &api.CreateChunkRequest{
		ChunkId: "foo",
		Size:    11,
		Role:    api.ReplicaRole_PRIMARY})
	if err != nil {
		t.Error(err)
	}
	_, err = s.WriteChunk(context.Background(), &api.WriteChunkRequest{
		ChunkId: "foo",
		Offset:  0,
		Data:    []byte("hello world")})
	if err != nil {
		t.Error(err)
	}
}

func TestStoreAndReadChunk(t *testing.T) {
	s := ChunkServer{factory: &InMemoryChunkFactory{}}
	_, err := s.CreateChunk(context.Background(), &api.CreateChunkRequest{
		ChunkId: "foo",
		Size:    11,
		Role:    api.ReplicaRole_PRIMARY})
	if err != nil {
		t.Error(err)
	}
	_, err = s.WriteChunk(context.Background(), &api.WriteChunkRequest{
		ChunkId: "foo",
		Offset:  0,
		Data:    []byte("hello world")})
	if err != nil {
		t.Error(err)
	}
	resp, err := s.ReadChunk(context.Background(), &api.ReadChunkRequest{
		ChunkId:  "foo",
		NumBytes: 100})
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(resp.Data, []byte("hello world")) != 0 {
		t.Errorf("Expected: %s\nGot: %s\n", "hello world", resp.Data)
	}
}

func TestReadWithOffsets(t *testing.T) {
	s := ChunkServer{factory: &InMemoryChunkFactory{}}
	_, err := s.CreateChunk(context.Background(), &api.CreateChunkRequest{
		ChunkId: "foo",
		Size:    11,
		Role:    api.ReplicaRole_PRIMARY})
	if err != nil {
		t.Error(err)
	}
	_, err = s.WriteChunk(context.Background(), &api.WriteChunkRequest{
		ChunkId: "foo",
		Offset:  0,
		Data:    []byte("hello world")})
	if err != nil {
		t.Error(err)
	}
	resp, err := s.ReadChunk(context.Background(), &api.ReadChunkRequest{
		ChunkId:  "foo",
		Offset:   0,
		NumBytes: 2})
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(resp.Data, []byte("he")) != 0 {
		t.Errorf("Expected: %s\nGot: %s\n", "he", resp.Data)
	}
	resp, err = s.ReadChunk(context.Background(), &api.ReadChunkRequest{
		ChunkId:  "foo",
		Offset:   2,
		NumBytes: 2})
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(resp.Data, []byte("ll")) != 0 {
		t.Errorf("Expected: %s\nGot: %s\n", "ll", resp.Data)
	}
	resp, err = s.ReadChunk(context.Background(), &api.ReadChunkRequest{
		ChunkId:  "foo",
		Offset:   4,
		NumBytes: 100})
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(resp.Data, []byte("o world")) != 0 {
		t.Errorf("Expected: %s\nGot: %s\n", "o world", resp.Data)
	}

}
