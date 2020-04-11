package client

import (
	"context"
	"os"

	"github.com/giolekva/pcloud/pfs/api"
	"github.com/giolekva/pcloud/pfs/chunk"
)

type FileUploader struct {
	client api.MetadataStorageClient
}

func NewFileUploader(client api.MetadataStorageClient) *FileUploader {
	return &FileUploader{client}
}

func (fu *FileUploader) Upload(f *os.File, numReplicas int) {
	info, err := f.Stat()
	if err != nil {
		return
	}
	resp, err := fu.client.CreateBlob(
		context.Background(), &api.CreateBlobRequest{
			SizeBytes:   int32(info.Size()),
			NumReplicas: int32(numReplicas)})
	if err != nil {
		panic(err)
	}
	if len(resp.Chunk) != 1 {
		panic(resp)
	}
	lis := &chunk.NonChangingReplicaAssignment{}
	primaryAddressCh := lis.Primary(
		resp.Chunk[0].ChunkId,
		resp.Chunk[0].Server[0])
	chunk.WriteToPrimary(
		context.Background(),
		resp.Chunk[0].ChunkId,
		chunk.NewReadOnlyFileChunk(f, 0, int(info.Size())),
		primaryAddressCh)
}
