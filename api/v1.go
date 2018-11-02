package api

import (
	"net/http"

	"pointwatch.com/httputil"
)

// V1 handles v1 api requests
type V1 struct {
	SampleHandler http.Handler
}

// Top level handler for http requests in the coap-server process
func (h *V1) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var head string

	head, req.URL.Path = httputil.ShiftPath(req.URL.Path)
	switch head {
	case "sample":
		h.SampleHandler.ServeHTTP(res, req)
	default:
		http.Error(res, "Not Found", http.StatusNotFound)
	}
}

// NewV1Handler returns a handle for V1 API
func NewV1Handler() http.Handler {
	return &V1{
		SampleHandler: NewSampleHandler(),
	}
}
