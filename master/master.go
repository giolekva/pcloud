package main

import "context"
import "flag"
import "fmt"
import "log"
import "net"

import "google.golang.org/grpc"

import pc "pcloud"

var port int

func init() {
	flag.IntVar(&port, "port", 123, "Port to listen on.")
}

type chunkServer struct {
	address string
}

type metadataStorage struct {
	chunkServer []string
}

func (s *metadataStorage) AddChunkServer(
	ctx context.Context,
	request *pc.AddChunkServerRequest) (*pc.AddChunkServerResponse, error) {
	s.chunkServer = append(s.chunkServer, request.GetAddress())
	log.Printf("Registered Chunk server: %s", request.GetAddress())
	return &pc.AddChunkServerResponse{}, nil
}

func (s *metadataStorage) CreateBlob(
	ctx context.Context,
	request *pc.CreateBlobRequest) (*pc.CreateBlobResponse, error) {
	return nil, nil
}

func (s *metadataStorage) GetBlobMetadata(
	ctx context.Context,
	request *pc.GetBlobMetadataRequest) (*pc.GetBlobMetadataResponse, error) {
	return nil, nil
}

func main() {
	flag.Parse()
	log.Print("Master server starting")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	log.Printf("Listening on port: %d", port)
	server := grpc.NewServer()
	pc.RegisterMetadataStorageServer(server, &metadataStorage{})
	log.Print("Master serving")
	server.Serve(lis)
}
