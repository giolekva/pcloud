package main

import "flag"
import "fmt"
import "log"
import "net"

import "google.golang.org/grpc"

import "pcloud/api"
import "pcloud/master"

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
	api.RegisterMetadataStorageServer(server, master.NewMasterServer())
	log.Print("Master serving")
	server.Serve(lis)
}
