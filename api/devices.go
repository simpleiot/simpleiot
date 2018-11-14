package api

import (
	"encoding/json"
	"net/http"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
)

// Devices handles device requests
type Devices struct {
	db *db.Db
}

func (h *Devices) processSample(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var s data.Sample
	err := decoder.Decode(&s)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.db.DeviceSample(id, s)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

// Top level handler for http requests in the coap-server process
func (h *Devices) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	switch head {
	case "sample":
		if req.Method == http.MethodPost {
			h.processSample(res, req, id)
		} else {
			http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		}
	default:
		if id == "" {
			devices, err := h.db.Devices()
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}
			en := json.NewEncoder(res)
			en.Encode(devices)
		} else {
			device, err := h.db.Device(id)
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
func NewDevicesHandler(db *db.Db) http.Handler {
	return &Devices{db}
}
