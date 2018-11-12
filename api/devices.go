package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/simpleiot/simpleiot/data"
)

// Devices handles device requests
type Devices struct {
	SampleHandler http.Handler
}

func processSample(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var s data.Sample
	err := decoder.Decode(&s)
	if err != nil {
		panic(err)
	}
	log.Printf("CLIFF: sample ID: %v, value: %+v\n", id, s)
}

// Top level handler for http requests in the coap-server process
func (h *Devices) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	switch head {
	case "sample":
		processSample(res, req, id)
	default:
		http.Error(res, "Not Found", http.StatusNotFound)
	}
}

// NewDevicesHandler returns a new device handler
func NewDevicesHandler() http.Handler {
	return &Devices{}
}
