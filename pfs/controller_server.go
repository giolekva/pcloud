package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/giolekva/pcloud/pfs/api"
	"github.com/giolekva/pcloud/pfs/controller"
)

var port = flag.Int("port", 123, "Port to listen on.")

func main() {
	flag.Parse()
	log.Print("Master server starting")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", *port, err)
	}
	log.Printf("Listening on port: %d", *port)
	server := grpc.NewServer()
	api.RegisterMetadataStorageServer(server, controller.NewMasterServer())
	log.Print("Master serving")
	server.Serve(lis)
}
