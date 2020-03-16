package main

import "context"
import "flag"
import "fmt"
import "log"
import "net"
import "time"

import "google.golang.org/grpc"

import pc "pcloud"

var masterAddress string
var selfAddress string

func init() {
	flag.StringVar(&masterAddress, "master", "localhost:123", "Metadata storage address.")
	flag.StringVar(&selfAddress, "self", "", "Metadata storage address.")
}

type chunkStorage struct {
}

func (s *chunkStorage) ListChunks(
	ctx context.Context,
	request *pc.ListChunksRequest) (*pc.ListChunksResponse, error) {
	return nil, nil
}

func (s *chunkStorage) ReadChunk(
	ctx context.Context,
	request *pc.ReadChunkRequest) (*pc.ReadChunkResponse, error) {
	return nil, nil
}

func (s *chunkStorage) StoreChunk(
	ctx context.Context,
	request *pc.StoreChunkRequest) (*pc.StoreChunkResponse, error) {
	return nil, nil
}

func main() {
	flag.Parse()
	log.Print("Chunk server starting")

	// Create Master server client.
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(masterAddress, opts...)
	if err != nil {
		log.Fatalf("Failed to dial %s: %v", masterAddress, err)
	}
	defer conn.Close()
	client := pc.NewMetadataStorageClient(conn)

	// Register current Chunk server with Master.
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = client.AddChunkServer(
		ctx,
		&pc.AddChunkServerRequest{Address: selfAddress})
	if err != nil {
		log.Fatalf("failed to register chunk server: %v", err)
	}
	log.Print("Registered myself")

	// Start RPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 234))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()
	pc.RegisterChunkStorageServer(server, &chunkStorage{})
	server.Serve(lis)
}
