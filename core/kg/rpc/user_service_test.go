package rpc_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/giolekva/pcloud/core/kg/model/proto"
	"github.com/giolekva/pcloud/core/kg/server"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestUserService(t *testing.T) {
	ts := server.Setup(t)
	defer ts.ShutdownServers()
	_, err := ts.App.GetUser("id")
	assert.NotNil(t, err)

	ctx := context.Background()
	address := fmt.Sprintf("localhost:%d", ts.Config.GRPC.Port)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := proto.NewUserServiceClient(conn)
	request := &proto.GetUserRequest{Id: "id"}
	_, err = client.GetUser(ctx, request)
	assert.NotNil(t, err)
}
