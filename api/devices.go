package api

import (
	"encoding/json"
	"net/http"

	"github.com/simpleiot/simpleiot/data"
)

// Devices handles device requests
type Devices struct {
	state *data.State
}

func (h *Devices) processSample(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var s data.Sample
	err := decoder.Decode(&s)
	if err != nil {
		panic(err)
	}

	h.state.UpdateDevice(id, s)
}

// Top level handler for http requests in the coap-server process
func (h *Devices) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	switch head {
	case "sample":
		h.processSample(res, req, id)
	default:
		if id == "" {
			en := json.NewEncoder(res)
			en.Encode(h.state.Devices())
		} else {
			device, err := h.state.Device(id)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
			} else {
				en := json.NewEncoder(res)
				en.Encode(device)
			}
		}
	}
}

// NewDevicesHandler returns a new device handler
func NewDevicesHandler(state *data.State) http.Handler {
	return &Devices{state}
}
