package api

import (
	"log"
	"net/http"
)

// V1 handles v1 api requests
type V1 struct {
	GroupsHandler http.Handler
	UsersHandler  http.Handler
	NodesHandler  http.Handler
	AuthHandler   http.Handler
	MsgHandler    http.Handler
	// Version is the running binary's version; the UI compares it on
	// each reconnect, so an open tab reloads after a new release starts.
	Version string
}

// Top level handler for http requests in the coap-server process
func (h *V1) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)
	switch head {
	case "nodes":
		h.NodesHandler.ServeHTTP(res, req)
	case "auth":
		h.AuthHandler.ServeHTTP(res, req)
	case "version":
		h.serveVersion(res, req)
	default:
		http.Error(res, "Not Found", http.StatusNotFound)
	}
}

// NewV1Handler returns a handle for V1 API
func NewV1Handler(args ServerArgs) http.Handler {
	return &V1{
		NodesHandler: NewNodesHandler(args.Users,
			args.AuthToken, args.Nc, args.DeviceAuth, args.DeviceAuthRequired),
		AuthHandler: NewAuthHandler(args.Nc, args.AuthToken != ""),
		Version:     args.AppVersion,
	}
}

// serveVersion answers GET /v1/version with the binary's version. It is
// not cached, since a new binary answers differently.
func (h *V1) serveVersion(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Cache-Control", "no-cache")
	if err := encode(res, map[string]string{"version": h.Version}); err != nil {
		log.Println("Error encoding version:", err)
	}
}
