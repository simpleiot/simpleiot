package api

import (
	"net/http"

	"github.com/simpleiot/simpleiot/db"
)

// V1 handles v1 api requests
type V1 struct {
	UsersHandler   http.Handler
	DevicesHandler http.Handler
	AuthHandler    http.Handler
}

// Top level handler for http requests in the coap-server process
func (h *V1) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)
	switch head {
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
func NewV1Handler(db *Db, influx *db.Influx, auth Authorizer) http.Handler {
	return &V1{
		UsersHandler:   NewUsersHandler(db),
		DevicesHandler: NewDevicesHandler(db, influx, auth),
		AuthHandler:    NewAuthHandler(db, auth),
	}
}
