package master

import (
	"context"
	"log"
	"math/rand"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/giolekva/pcloud/api"
)

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
	chunkServers []*chunkServer
	blobs        []*blob
}

func NewMasterServer() *MasterServer {
	return &MasterServer{}
}

func (s *MasterServer) AddChunkServer(
	ctx context.Context,
	req *api.AddChunkServerRequest) (*api.AddChunkServerResponse, error) {
	s.chunkServers = append(s.chunkServers, &chunkServer{
		address: req.Address,
		status:  Healthy})
	log.Printf("Registered Chunk server: %s", req.Address)
	return &api.AddChunkServerResponse{}, nil
}

func (s *MasterServer) CreateBlob(
	ctx context.Context,
	req *api.CreateBlobRequest) (*api.CreateBlobResponse, error) {
	if int(req.NumReplicas) > len(s.chunkServers) {
		return nil, nil
	}
	resp := api.CreateBlobResponse{
		BlobId: uuid.New().String(),
		Chunk: []*api.ChunkStorageMetadata{
			{ChunkId: uuid.New().String()},
		}}
	assigned := 0
	chunkId := resp.Chunk[0].ChunkId
	for i := range rand.Perm(len(s.chunkServers)) {
		if assigned == int(req.NumReplicas) {
			break
		}
		address := s.chunkServers[i].address
		log.Printf(address)
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		opts = append(opts, grpc.WithBlock())
		conn, err := grpc.Dial(address, opts...)
		if err != nil {
			continue
		}
		defer conn.Close()
		client := api.NewChunkStorageClient(conn)
		createChunkReq := api.CreateChunkRequest{
			ChunkId: chunkId,
			Size:    req.SizeBytes}
		if assigned == 0 {
			createChunkReq.Role = api.ReplicaRole_PRIMARY
		} else {
			createChunkReq.Role = api.ReplicaRole_SECONDARY
			createChunkReq.PrimaryAddress = resp.Chunk[0].Server[0]
		}
		_, err = client.CreateChunk(ctx, &createChunkReq)
		if err == nil {
			resp.Chunk[0].Server = append(resp.Chunk[0].Server, address)
			assigned++
		}
	}
	return &resp, nil
}

func (s *MasterServer) GetBlobMetadata(
	ctx context.Context,
	req *api.GetBlobMetadataRequest) (*api.GetBlobMetadataResponse, error) {
	return nil, nil
}
