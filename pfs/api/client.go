package api

import (
	"google.golang.org/grpc"
)

func DialConn(address string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	return grpc.Dial(address, opts...)
}
