package rest

import (
	"net/http"

	"github.com/giolekva/pcloud/core/kg/common"
	"github.com/gorilla/mux"
)

const APIURLSuffix = "/api/v1"

type Router struct {
	App    common.AppIface
	Logger common.LoggerIface

	Root    *mux.Router // ''
	APIRoot *mux.Router // 'api/v1'
	Users   *mux.Router // 'api/v1/users'
	User    *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}'
}

func NewRouter(root *mux.Router, app common.AppIface, logger common.LoggerIface) *Router {
	apiRoot := root.PathPrefix(APIURLSuffix).Subrouter()
	users := apiRoot.PathPrefix("/users").Subrouter()
	user := apiRoot.PathPrefix("/users/{user_id:[A-Za-z0-9]+}").Subrouter()

	routers := &Router{
		App:    app,
		Logger: logger,

		Root:    root,
		APIRoot: apiRoot,
		Users:   users,
		User:    user,
	}

	root.Handle("/api/v1/{anything:.*}", http.HandlerFunc(http.NotFound))
	routers.initUsers()

	return routers
}

func (router *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	router.Root.ServeHTTP(w, req)
}
