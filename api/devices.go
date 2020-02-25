package api

import (
	"encoding/json"
	"net/http"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
)

// Devices handles device requests
type Devices struct {
	db     *db.Db
	influx *db.Influx
	check  HeaderValidator
}

func (h *Devices) processCmd(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var c data.DeviceCmd
	err := decoder.Decode(&c)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// set ID in case it is not set in API call
	c.ID = id

	err = h.db.DeviceSetCmd(c)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}

func (h *Devices) processVersion(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var v data.DeviceVersion
	err := decoder.Decode(&v)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.db.DeviceSetVersion(id, v)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}

func (h *Devices) processConfig(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var c data.DeviceConfig
	err := decoder.Decode(&c)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.db.DeviceUpdateConfig(id, c)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}

func (h *Devices) processSamples(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var samples []data.Sample
	err := decoder.Decode(&samples)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	for _, s := range samples {
		err = h.db.DeviceSample(id, s)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
	}

	if h.influx != nil {
		err = h.influx.WriteSamples(samples)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}

// Top level handler for http requests in the coap-server process
func (h *Devices) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if !h.check.ValidHeader(req) {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	switch head {
	case "samples":
		if req.Method == http.MethodPost {
			h.processSamples(res, req, id)
		} else {
			http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		}
	case "config":
		if req.Method == http.MethodPost {
			h.processConfig(res, req, id)
		} else {
			http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		}
	case "cmd":
		if req.Method == http.MethodGet {
			cmd, err := h.db.DeviceGetCmd(id)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}

			// id is not required
			cmd.ID = ""

			en := json.NewEncoder(res)
			en.Encode(cmd)
		} else if req.Method == http.MethodPost {
			h.processCmd(res, req, id)
		} else {
			http.Error(res, "only GET allowed", http.StatusMethodNotAllowed)
		}
	case "version":
		if req.Method == http.MethodPost {
			h.processVersion(res, req, id)
		} else {
			http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		}
	default:
		if id == "" {
			switch req.Method {
			case http.MethodGet:
				devices, err := h.db.Devices()
				if err != nil {
					http.Error(res, err.Error(), http.StatusNotFound)
					return
				}
				en := json.NewEncoder(res)
				en.Encode(devices)
			default:
				http.Error(res, "invalid method", http.StatusMethodNotAllowed)
			}
		} else {
			switch req.Method {
			case http.MethodGet:
				device, err := h.db.Device(id)
				if err != nil {
					http.Error(res, err.Error(), http.StatusNotFound)
				} else {
					en := json.NewEncoder(res)
					en.Encode(device)
				}
			case http.MethodDelete:
				err := h.db.DeviceDelete(id)
				if err != nil {
					http.Error(res, err.Error(), http.StatusNotFound)
				} else {
					en := json.NewEncoder(res)
					en.Encode(data.StandardResponse{Success: true, ID: id})
				}

			default:
				http.Error(res, "invalid method", http.StatusMethodNotAllowed)
			}
		}
	}
}

type HeaderValidator interface {
	ValidHeader(req *http.Request) bool
}

// NewDevicesHandler returns a new device handler
func NewDevicesHandler(db *db.Db, influx *db.Influx, v HeaderValidator) http.Handler {
	return &Devices{db, influx, v}
}
