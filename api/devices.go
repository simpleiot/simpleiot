package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
	"github.com/simpleiot/simpleiot/nats"
)

// Devices handles device requests
type Devices struct {
	db    *db.Db
	check RequestValidator
	nh    *NatsHandler
}

// NewDevicesHandler returns a new device handler
func NewDevicesHandler(db *db.Db, v RequestValidator, nh *NatsHandler) http.Handler {
	return &Devices{db, v, nh}
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
			devices, err := h.db.DevicesForUser(userUUID)
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
	switch head {
	case "":
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

	err := h.db.DeviceActivity(id)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	switch opt {
	case "samples":
		h.processSamples(res, req, id)

	case "cmd":
		cmd, err := h.db.DeviceGetCmd(id)
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
	var cmd data.DeviceCmd
	err := decoder.Decode(&cmd)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// set ID in case it is not set in API call
	cmd.ID = id

	// set cmd in DB for legacy devices that still fetch over http
	err = h.db.DeviceSetCmd(cmd)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO how to support old devices still fetching commands via http
	// perhaps check if device is connected via NATs
	err = nats.NatsSendCmd(h.nh.Nc, cmd, time.Second*10)
	if err != nil {
		log.Printf("Error sending command (%v) to device: ", err)
		// don't return HTTP error for now as some units still fetch over http
		//http.Error(res, "Error sending command to device", http.StatusInternalServerError)
		//return
	} else {
		err = h.db.DeviceDeleteCmd(cmd.ID)
		if err != nil {
			log.Printf("Error deleting command for device %v: %v", id, err)
		}
	}

	// process updates that are now pushed
	if cmd.Cmd == data.CmdUpdateApp {
		log.Printf("Sending %v to device %v\n", cmd.Detail, cmd.ID)
		err := h.nh.StartUpdate(cmd.ID, cmd.Detail)
		if err != nil {
			log.Println("Error starting app update: ", err)
			http.Error(res, "error starting update", http.StatusInternalServerError)
			return
		}
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

func (h *Devices) processConfig(res http.ResponseWriter, req *http.Request, id string, userID string) {
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

func (h *Devices) updateDeviceGroups(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var groups []uuid.UUID
	err := decoder.Decode(&groups)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.db.DeviceUpdateGroups(id, groups)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
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
			return
		}
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}
