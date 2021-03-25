package rest

import (
	"net/http"

	"github.com/gorilla/mux"
)

const APIURLSuffix = "/api/v1"

type Routers struct {
	Root    *mux.Router // ''
	APIRoot *mux.Router // 'api/v1'
	Users   *mux.Router // 'api/v1/users'
	User    *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}'
}

func NewRouter(root *mux.Router) *Routers {
	apiRoot := root.PathPrefix(APIURLSuffix).Subrouter()
	users := apiRoot.PathPrefix("/users").Subrouter()
	user := apiRoot.PathPrefix("/users/{user_id:[A-Za-z0-9]+}").Subrouter()

	routers := &Routers{
		Root:    root,
		APIRoot: apiRoot,
		Users:   users,
		User:    user,
	}
	root.Handle("/api/v1/{anything:.*}", http.HandlerFunc(http.NotFound))
	routers.initUsers()

	return routers
}

func (r *Routers) initUsers() {
	r.Users.Handle("", http.HandlerFunc(createUser)).Methods("POST")
	r.Users.Handle("", http.HandlerFunc(getUsers)).Methods("GET")
	r.User.Handle("", http.HandlerFunc(getUser)).Methods("GET")
}
