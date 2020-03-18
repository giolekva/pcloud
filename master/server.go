package master

import "context"
import "log"
import "math/rand"

import "github.com/google/uuid"

import "pcloud/api"

type chunkServers struct {
	address string
}

type BlobStatus int

const (
	NEW BlobStatus = iota
)

type ChunkStatus int

const (
	ASSIGNED ChunkStatus = iota
	STORED
)

type chunkReplica struct {
	chunkServer string
	status      ChunkStatus
}

type chunk struct {
	id      string
	replica []chunkReplica
}

type blob struct {
	id     string
	status BlobStatus
	chunks []chunk
}

type MasterServer struct {
	chunkServers []string
	blobs        []*blob
}

func NewMasterServer() *MasterServer {
	return &MasterServer{}
}

func (s *MasterServer) AddChunkServer(
	ctx context.Context,
	request *api.AddChunkServerRequest) (*api.AddChunkServerResponse, error) {
	s.chunkServers = append(s.chunkServers, request.Address)
	log.Printf("Registered Chunk server: %s", request.Address)
	return &api.AddChunkServerResponse{}, nil
}

func (s *MasterServer) CreateBlob(
	ctx context.Context,
	request *api.CreateBlobRequest) (*api.CreateBlobResponse, error) {
	if int(request.NumReplicas) > len(s.chunkServers) {
		return nil, nil
	}
	resp := api.CreateBlobResponse{
		BlobId: uuid.New().String(),
		Chunk: []*api.ChunkStorageMetadata{
			{ChunkId: uuid.New().String()},
		}}
	ids := rand.Perm(len(s.chunkServers))
	for i := 0; i < int(request.NumReplicas); i++ {
		resp.Chunk[0].Server = append(
			resp.Chunk[0].Server,
			s.chunkServers[ids[i]])
	}
	return &resp, nil
}

func (s *MasterServer) GetBlobMetadata(
	ctx context.Context,
	request *api.GetBlobMetadataRequest) (*api.GetBlobMetadataResponse, error) {
	return nil, nil
}
