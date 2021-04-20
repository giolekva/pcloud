package rest_test

import (
	"testing"

	"github.com/giolekva/pcloud/core/kg/server"
	"github.com/stretchr/testify/assert"
)

func TestUserService(t *testing.T) {
	ts := server.Setup(t)
	defer ts.ShutdownServers()
	user, resp := ts.RestClient.GetUser("id")
	assert.Nil(t, user)
	assert.Contains(t, resp.Error.Error(), "User not found")
	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "", resp.RequestID)
}
