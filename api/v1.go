package api

import (
	"net/http"

	"github.com/simpleiot/simpleiot/db"
)

// V1 handles v1 api requests
type V1 struct {
	GroupsHandler  http.Handler
	UsersHandler   http.Handler
	DevicesHandler http.Handler
	AuthHandler    http.Handler
}

// Top level handler for http requests in the coap-server process
func (h *V1) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)
	switch head {
	case "groups":
		h.GroupsHandler.ServeHTTP(res, req)
	case "users":
		h.UsersHandler.ServeHTTP(res, req)
	case "devices":
		h.DevicesHandler.ServeHTTP(res, req)
	case "auth":
		h.AuthHandler.ServeHTTP(res, req)
	default:
		http.Error(res, "Not Found", http.StatusNotFound)
	}
}

// NewV1Handler returns a handle for V1 API
func NewV1Handler(db *db.Db, auth Authorizer,
	authToken string, nh *NatsHandler) http.Handler {

	return &V1{
		GroupsHandler:  NewGroupsHandler(db, auth),
		UsersHandler:   NewUsersHandler(db, auth),
		DevicesHandler: NewDevicesHandler(db, auth, authToken, nh),
		AuthHandler:    NewAuthHandler(db, auth),
	}
}
