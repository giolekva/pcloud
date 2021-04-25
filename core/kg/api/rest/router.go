package rest

import (
	"net/http"
	"strings"

	"github.com/giolekva/pcloud/core/kg/common"
	"github.com/giolekva/pcloud/core/kg/log"
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
	root.Use(routers.loggerMiddleware)
	root.Use(routers.authMiddleware)
	return routers
}

func (router *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	router.Root.ServeHTTP(w, req)
}

func (router *Router) loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.Logger.Debug(r.Method, log.String("url", r.URL.String()))
		next.ServeHTTP(w, r)
	})
}

func (router *Router) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := common.NewID()
		w.Header().Set(HeaderRequestID, requestID)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			w.Header().Set("Expires", "0")
		}

		token := parseAuthTokenFromRequest(r)
		if token != "" {
			session, err := router.App.GetSession(token)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(err.Error()))
				return
			}
			router.App.SetSession(session)
		}
		next.ServeHTTP(w, r)
	})
}

func parseAuthTokenFromRequest(r *http.Request) string {
	authHeader := r.Header.Get(HeaderAuth)

	// Parse the token from the header
	if len(authHeader) > 6 && strings.ToUpper(authHeader[0:6]) == HeaderBearer {
		// Default session token
		return authHeader[7:]
	}

	if len(authHeader) > 5 && strings.ToLower(authHeader[0:5]) == HeaderToken {
		// OAuth token
		return authHeader[6:]
	}

	// Attempt to parse token out of the query string
	if token := r.URL.Query().Get("access_token"); token != "" {
		return token
	}

	return ""
}
