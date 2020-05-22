package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
)

// Devices handles device requests
type Devices struct {
	db     *Db
	influx *db.Influx
	check  RequestValidator
}

// NewDevicesHandler returns a new device handler
func NewDevicesHandler(db *Db, influx *db.Influx, v RequestValidator) http.Handler {
	return &Devices{db, influx, v}
}

// Top level handler for http requests in the coap-server process
// TODO need to add device auth
func (h *Devices) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	if (head == "samples" && req.Method == http.MethodPost) ||
		(head == "cmd" && req.Method == http.MethodGet) ||
		(head == "version" && req.Method == http.MethodPost) {
		h.ServeHTTPDevice(res, req, id, head)
		return
	}

	// all user based requests require valid auth
	validUser, userID := h.check.Valid(req)

	if !validUser {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userUUID, err := uuid.Parse(userID)

	if err != nil {
		http.Error(res, "User UUID invalid", http.StatusUnauthorized)
		return
	}

	if id == "" {
		switch req.Method {
		case http.MethodGet:
			// get all devices
			// TODO: only get user devices
			devices, err := devicesForUser(h.db.store, userUUID)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}
			if len(devices) > 0 {
				en := json.NewEncoder(res)
				en.Encode(devices)
			} else {
				res.Write([]byte("[]"))
			}
		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
		}
		return
	}

	// process requests with an ID.
	// FIXME uses userID to gate requests
	switch head {
	case "":
		switch req.Method {
		case http.MethodGet:
			device, err := device(h.db.store, id)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
			} else {
				en := json.NewEncoder(res)
				en.Encode(device)
			}
		case http.MethodDelete:
			err := deviceDelete(h.db.store, id)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
			} else {
				en := json.NewEncoder(res)
				en.Encode(data.StandardResponse{Success: true, ID: id})
			}
		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
		}

	case "config":
		if req.Method == http.MethodPost {
			h.processConfig(res, req, id, userID)
		} else {
			http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		}
	case "groups":
		if req.Method == http.MethodPost {
			h.updateDeviceGroups(res, req, id)
		} else {
			http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		}

	case "cmd":
		if req.Method == http.MethodPost {
			// Post is typically done by UI
			if !validUser {
				http.Error(res, "Unauthorized", http.StatusUnauthorized)
				return
			}

			h.processCmd(res, req, id)
		} else {
			http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		}

	}
}

// ServeHTTPDevice is used to process requests from device, currently
// no auth and following URIs:
// - POST samples
// - GET cmd
// - POST version
// we have already checked for req type, so we can skip that check
// here
func (h *Devices) ServeHTTPDevice(res http.ResponseWriter, req *http.Request, id, opt string) {
	// we've heard from device, update last heard timestamp

	err := deviceActivity(h.db.store, id)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	switch opt {
	case "samples":
		h.processSamples(res, req, id)

	case "cmd":
		cmd, err := deviceGetCmd(h.db.store, id)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		// id is not required
		cmd.ID = ""

		en := json.NewEncoder(res)
		en.Encode(cmd)

	case "version":
		h.processVersion(res, req, id)
	}
}

// RequestValidator validates an HTTP request.
type RequestValidator interface {
	Valid(req *http.Request) (bool, string)
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

	err = deviceSetCmd(h.db.store, c)
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

	err = deviceSetVersion(h.db.store, id, v)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}

func (h *Devices) processConfig(res http.ResponseWriter, req *http.Request, id string, userID string) {
	decoder := json.NewDecoder(req.Body)
	var c data.DeviceConfig
	err := decoder.Decode(&c)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = deviceUpdateConfig(h.db.store, id, c)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}

func (h *Devices) updateDeviceGroups(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var groups []uuid.UUID
	err := decoder.Decode(&groups)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = deviceUpdateGroups(h.db.store, id, groups)
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
		err = deviceSample(h.db.store, id, s)
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
