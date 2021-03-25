package rest

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func (router *Router) initUsers() {
	router.Users.Handle("", router.buildCreateUserHandler()).Methods("POST")
	router.Users.Handle("", router.buildGetUsersHandler()).Methods("GET")
	router.User.Handle("", router.buildGetUserHandler()).Methods("GET")
}

func (router *Router) buildCreateUserHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		router.Logger.Debug("Rest API: create user")
		return nil
	}
	return HandlerFunc(fn)
}

func (router *Router) buildGetUsersHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		router.Logger.Debug("Rest API: get users")
		return nil
	}
	return HandlerFunc(fn)
}

func (router *Router) buildGetUserHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		router.Logger.Debug("Rest API: get user")
		params := mux.Vars(r)

		var userID string
		var ok bool
		if userID, ok = params["user_id"]; !ok {
			return errors.New("missing parameter: user_id")
		}
		user, err := router.App.GetUser(userID)

		if err != nil {
			return errors.Wrapf(err, "can't get user from app")
		}

		jsoner(w, http.StatusOK, user)
		return nil
	}
	return HandlerFunc(fn)
}
