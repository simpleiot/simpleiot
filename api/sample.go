package api

import (
	"log"
	"net/http"
)

// Sample handles sample requests
type Sample struct {
}

// Top level handler for http requests in the coap-server process
func (h *Sample) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	log.Println("CLIFF: sample received: ")
}

// NewSampleHandler returns a new sample handler
func NewSampleHandler() http.Handler {
	return &Sample{}
}
