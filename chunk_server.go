package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/giolekva/pcloud/api"
	"github.com/giolekva/pcloud/chunk"
)

var masterAddress = flag.String("master", "localhost:123", "Metadata storage address.")
var selfAddress = flag.String("self", "", "Metadata storage address.")

func main() {
	flag.Parse()
	log.Print("Chunk server starting")

	// Create Master server client.
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(*masterAddress, opts...)
	if err != nil {
		log.Fatalf("Failed to dial %s: %v", *masterAddress, err)
	}
	defer conn.Close()
	client := api.NewMetadataStorageClient(conn)

	// Register current Chunk server with Master.
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = client.AddChunkServer(
		ctx,
		&api.AddChunkServerRequest{Address: *selfAddress})
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
	api.RegisterChunkStorageServer(server, chunk.NewChunkServer(
		&chunk.InMemoryChunkFactory{},
		&chunk.NonChangingReplicaAssignment{}))
	server.Serve(lis)
}
