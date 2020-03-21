package client

import "os"

import "pcloud/api"

type FileUploader struct {
	client api.MetadataStorageServerClient
}

func NewFileUploader(client api.MetadataStorageServerClient) *FileUploader {
	return FileUploader{client}
}

func (fu *FileUploader) Upload(f *os.File) (n int64, err error) {

	buf := make([]byte, 1000)
	for {
		n, err := f.Read(buf)
	}
}
