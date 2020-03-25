package testing

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/giolekva/pcloud/api"
	"github.com/giolekva/pcloud/chunk"
	"github.com/giolekva/pcloud/master"
)

type InMemoryEnv struct {
	m          *grpc.Server
	c          []*grpc.Server
	masterConn *grpc.ClientConn
}

func NewInMemoryEnv(numChunkServers int) (*InMemoryEnv, error) {
	env := new(InMemoryEnv)
	syscall.Unlink("/tmp/pcloud/master")
	lis, err := net.Listen("unix", "/tmp/pcloud/master")
	if err != nil {
		return nil, err
	}
	server := grpc.NewServer()
	api.RegisterMetadataStorageServer(server, master.NewMasterServer())
	go server.Serve(lis)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial("unix:/tmp/pcloud/master", opts...)
	if err != nil {
		return nil, err
	}
	env.masterConn = conn
	client := api.NewMetadataStorageClient(conn)

	env.c = make([]*grpc.Server, numChunkServers)
	for i := 0; i < numChunkServers; i++ {
		unixSocket := fmt.Sprintf("/tmp/pcloud/chunk-%d", i)
		syscall.Unlink(unixSocket)
		lis, err := net.Listen("unix", unixSocket)
		if err != nil {
			return nil, err
		}
		server := grpc.NewServer()
		api.RegisterChunkStorageServer(server, chunk.NewChunkServer(&chunk.InMemoryChunkFactory{}))
		go server.Serve(lis)
		env.c[i] = server
	}

	for i := 0; i < numChunkServers; i++ {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = client.AddChunkServer(
			ctx,
			&api.AddChunkServerRequest{Address: fmt.Sprintf("unix:///tmp/pcloud/chunk-%d", i)})
		if err != nil {
			return nil, err
		}
	}
	return env, nil
}

func (e *InMemoryEnv) Stop() {
	if e.masterConn != nil {
		e.masterConn.Close()
	}
	for _, s := range e.c {
		if s != nil {
			s.GracefulStop()
		}
	}
	if e.m != nil {
		e.m.GracefulStop()
	}
}
