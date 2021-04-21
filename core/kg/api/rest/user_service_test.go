package rest_test

import (
	"fmt"
	"testing"

	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/server"
	"github.com/stretchr/testify/assert"
)

func TestUserService(t *testing.T) {
	ts := server.Setup(t)
	defer ts.ShutdownServers()

	t.Run("Should not find user", func(t *testing.T) {
		user, resp := ts.RestClient.GetUser("id")
		assert.Nil(t, user)
		assert.Contains(t, resp.Error.Error(), "User not found")
		assert.Equal(t, 400, resp.StatusCode)
		assert.Equal(t, "", resp.RequestID)
	})

	t.Run("Should create and get user", func(t *testing.T) {
		user := &model.User{Username: "bla", Password: "bla"}
		uUser, resp := ts.RestClient.CreateUser(user)
		println(fmt.Sprintf("res = %v", resp))
		assert.Equal(t, 200, resp.StatusCode)
		assert.NotNil(t, uUser)
		user, resp = ts.RestClient.GetUser(uUser.ID)
		assert.Equal(t, 200, resp.StatusCode)
		assert.NotNil(t, user)
		assert.Equal(t, uUser.ID, user.ID)
		assert.Equal(t, "bla", user.Username)
		assert.Equal(t, "", user.Password)
	})
}
