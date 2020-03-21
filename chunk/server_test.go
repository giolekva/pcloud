package chunk

import (
	"bytes"
	"context"
	"testing"

	"pcloud/api"
)

func TestStoreChunk(t *testing.T) {
	s := ChunkServer{factory: &InMemoryChunkFactory{}}
	_, err := s.StoreChunk(context.Background(), &api.StoreChunkRequest{
		ChunkId: "foo",
		Data:    []byte("hello world")})
	if err != nil {
		t.Error(err)
	}
}

func TestStoreAndReadChunk(t *testing.T) {
	s := ChunkServer{factory: &InMemoryChunkFactory{}}
	_, err := s.StoreChunk(context.Background(), &api.StoreChunkRequest{
		ChunkId: "foo",
		Data:    []byte("hello world")})
	if err != nil {
		t.Error(err)
	}
	resp, err := s.ReadChunk(context.Background(), &api.ReadChunkRequest{
		ChunkId: "foo"})
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(resp.Data, []byte("hello world")) != 0 {
		t.Errorf("Expected: %s\nGot: %s\n", "hello world", resp.Data)
	}
}

func TestReadWithOffsets(t *testing.T) {
	s := ChunkServer{factory: &InMemoryChunkFactory{}}
	_, err := s.StoreChunk(context.Background(), &api.StoreChunkRequest{
		ChunkId: "foo",
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
		ChunkId: "foo",
		Offset:  4})
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(resp.Data, []byte("o world")) != 0 {
		t.Errorf("Expected: %s\nGot: %s\n", "o world", resp.Data)
	}

}
