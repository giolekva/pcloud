package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/giolekva/pcloud/core/kg/app"
	"github.com/giolekva/pcloud/core/kg/common"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func (router *Router) initUsers() {
	router.Users.Handle("", router.buildCreateUserHandler()).Methods("POST")
	router.Users.Handle("", router.buildGetUsersHandler()).Methods("GET")
	router.User.Handle("", router.buildGetUserHandler()).Methods("GET")

	router.Users.Handle("/login", router.buildGetLoginHandler()).Methods("POST")
	router.Users.Handle("/logout", router.buildGetLogoutHandler()).Methods("POST")
}

func (router *Router) buildCreateUserHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		var user *model.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			return errors.Wrap(err, "can't decode request body")
		}
		if err := user.IsValidInput(); err != nil {
			return errors.Wrap(err, "invalid user input")
		}
		user.Password = app.HashPassword(user.Password)
		user.SanitizeInput()
		updatedUser, err := router.App.CreateUser(user)
		if err != nil {
			return errors.Wrap(err, "can't create user")
		}
		updatedUser.SanitizeOutput()

		jsoner(w, http.StatusOK, updatedUser)
		return nil
	}
	return HandlerFunc(fn)
}

func (router *Router) buildGetUsersHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		page := r.URL.Query().Get("page")
		perPage := r.URL.Query().Get("per_page")

		pageInt, err := strconv.Atoi(page)
		if err != nil {
			return errors.New("parameter page should be an int")
		}
		perPageInt, err := strconv.Atoi(perPage)
		if err != nil {
			return errors.New("parameter per_page should be an int")
		}
		users, err := router.App.GetUsers(pageInt, perPageInt)
		if err != nil {
			return errors.Wrap(err, "can't get users from app")
		}

		jsoner(w, http.StatusOK, users)
		return nil
	}
	return HandlerFunc(fn)
}

func (router *Router) buildGetUserHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
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

func (router *Router) buildGetLoginHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		params := common.MapFromJson(r.Body)
		userID := params["user_id"]
		username := params["username"]
		password := params["password"]

		user, err := router.App.AuthenticateUserForLogin(userID, username, password)
		if err != nil {
			return errors.Wrap(err, "can't authenticate user for login")
		}

		if err := router.App.DoLogin(w, r, user); err != nil {
			return errors.Wrap(err, "can't login")
		}

		user.SanitizeOutput()
		jsoner(w, http.StatusOK, user)
		return nil
	}
	return HandlerFunc(fn)
}

func (router *Router) buildGetLogoutHandler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) error {
		if router.App.Session().ID != "" {
			if err := router.App.RevokeSession(router.App.Session().ID); err != nil {
				return errors.Wrap(err, "can't revoke session")
			}
		}
		jsoner(w, http.StatusOK, "")
		return nil
	}
	return HandlerFunc(fn)
}
