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
	db        *db.Db
	check     RequestValidator
	nh        *NatsHandler
	authToken string
}

// NewDevicesHandler returns a new device handler
func NewDevicesHandler(db *db.Db, v RequestValidator, authToken string,
	nh *NatsHandler) http.Handler {
	return &Devices{db, v, nh, authToken}
}

// Top level handler for http requests in the coap-server process
// TODO need to add device auth
func (h *Devices) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	var validUser bool
	var userUUID uuid.UUID

	if req.Header.Get("Authorization") != h.authToken {
		// all requests require valid JWT or authToken validation
		var userID string
		validUser, userID = h.check.Valid(req)

		if !validUser {
			http.Error(res, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var err error
		userUUID, err = uuid.Parse(userID)

		if err != nil {
			http.Error(res, "User UUID invalid", http.StatusUnauthorized)
			return
		}
	}

	if id == "" {
		switch req.Method {
		case http.MethodGet:
			if !validUser {
				http.Error(res, "invalid user", http.StatusMethodNotAllowed)
				return
			}

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
			return
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
			return
		}

	case "samples", "points":
		if req.Method == http.MethodPost {
			h.processPoints(res, req, id)
			return
		}

		http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		return

	case "groups":
		if req.Method == http.MethodPost {
			h.updateDeviceGroups(res, req, id)
			return
		}
		http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		return

	case "cmd":
		if req.Method == http.MethodPost {
			// Post is typically done by UI
			if !validUser {
				http.Error(res, "Unauthorized", http.StatusUnauthorized)
				return
			}

			h.processCmd(res, req, id)
			return
		} else if req.Method == http.MethodGet {
			cmd, err := h.db.DeviceGetCmd(id)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}

			// id is not required
			cmd.ID = ""

			en := json.NewEncoder(res)
			en.Encode(cmd)
			return
		} else {
			http.Error(res, "only POST/GET allowed", http.StatusMethodNotAllowed)
			return
		}
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
	err = nats.SendCmd(h.nh.Nc, cmd, time.Second*10)
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

func (h *Devices) processPoints(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var points data.Points
	err := decoder.Decode(&points)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	for _, p := range points {
		err = h.db.DevicePoint(id, p)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}
