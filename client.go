package main

import (
	"flag"
	"log"
	"os"

	"google.golang.org/grpc"

	"pcloud/api"
	"pcloud/client"
)

var masterAddress = flag.String("master", "localhost:123", "Metadata storage address.")
var fileToUpload = flag.String("file", "", "File path to upload.")

func main() {
	flag.Parse()

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(*masterAddress, opts...)
	if err != nil {
		log.Fatalf("Failed to dial %s: %v", *masterAddress, err)
	}
	defer conn.Close()
	uploader := client.NewFileUploader(api.NewMetadataStorageClient(conn))

	f, err := os.Open(*fileToUpload)
	if err != nil {
		panic(err)
	}

	uploader.Upload(f)
}
