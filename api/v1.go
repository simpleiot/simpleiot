package api

import (
	"net/http"
)

// V1 handles v1 api requests
type V1 struct {
	GroupsHandler http.Handler
	UsersHandler  http.Handler
	NodesHandler  http.Handler
	AuthHandler   http.Handler
	MsgHandler    http.Handler
}

// Top level handler for http requests in the coap-server process
func (h *V1) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)
	switch head {
	case "nodes":
		h.NodesHandler.ServeHTTP(res, req)
	case "devices":
		h.NodesHandler.ServeHTTP(res, req)
	case "auth":
		h.AuthHandler.ServeHTTP(res, req)
	default:
		http.Error(res, "Not Found", http.StatusNotFound)
	}
}

// NewV1Handler returns a handle for V1 API
func NewV1Handler(args ServerArgs) http.Handler {

	return &V1{
		NodesHandler: NewNodesHandler(args.DbInst, args.JwtAuth,
			args.AuthToken, args.Nc),
		AuthHandler: NewAuthHandler(args.DbInst, args.JwtAuth),
	}
}
