package rest_test

import (
	"net/http"
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
		assert.Contains(t, resp.Error.Error(), model.ErrNotFound.Error())
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Equal(t, "", resp.RequestID)
	})

	t.Run("Should create and get user", func(t *testing.T) {
		user := &model.User{Username: "bla", Password: "bla"}
		uUser, resp := ts.RestClient.CreateUser(user)
		assert.Equal(t, 200, resp.StatusCode)
		assert.NotNil(t, uUser)

		user, resp = ts.RestClient.GetUser(uUser.ID)
		assert.Equal(t, 200, resp.StatusCode)
		assert.NotNil(t, user)
		assert.Equal(t, uUser.ID, user.ID)
		assert.Equal(t, "bla", user.Username)

		users, resp := ts.RestClient.GetUsers(0, 0)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Len(t, users, 0)

		users, resp = ts.RestClient.GetUsers(0, 10)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Len(t, users, 1)
		assert.Equal(t, uUser.ID, users[0].ID)
	})
}

func TestLogin(t *testing.T) {
	ts := server.Setup(t)
	defer ts.ShutdownServers()

	user := &model.User{Username: "bla", Password: "bla"}
	uUser, resp := ts.RestClient.CreateUser(user)
	assert.Equal(t, 200, resp.StatusCode)
	assert.NotNil(t, uUser)

	t.Run("Should login user with id", func(t *testing.T) {
		loggedInUser, resp := ts.RestClient.LoginByUserID(uUser.ID, "bla")
		assert.Equal(t, 200, resp.StatusCode)
		assert.NotNil(t, loggedInUser)
		assert.Equal(t, uUser.ID, loggedInUser.ID)
	})

	t.Run("Should login user with username", func(t *testing.T) {
		loggedInUser, resp := ts.RestClient.LoginByUsername("bla", "bla")
		assert.Equal(t, 200, resp.StatusCode)
		assert.NotNil(t, loggedInUser)
		assert.Equal(t, uUser.ID, loggedInUser.ID)
	})

	t.Run("Should not login user with id", func(t *testing.T) {
		loggedInUser, resp := ts.RestClient.LoginByUserID(uUser.ID, "bla2")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Nil(t, loggedInUser)
		assert.Contains(t, resp.Error.Error(), model.ErrUnauthorized.Error())
	})

	t.Run("Should not login user with username", func(t *testing.T) {
		loggedInUser, resp := ts.RestClient.LoginByUsername("bla", "bla2")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Nil(t, loggedInUser)
		assert.Contains(t, resp.Error.Error(), model.ErrUnauthorized.Error())
	})
}
