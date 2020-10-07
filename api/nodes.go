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

// Nodes handles node requests
type Nodes struct {
	db        *db.Db
	check     RequestValidator
	nh        *NatsHandler
	authToken string
}

// NewNodesHandler returns a new node handler
func NewNodesHandler(db *db.Db, v RequestValidator, authToken string,
	nh *NatsHandler) http.Handler {
	return &Nodes{db, v, nh, authToken}
}

// Top level handler for http requests in the coap-server process
// TODO need to add node auth
func (h *Nodes) ServeHTTP(res http.ResponseWriter, req *http.Request) {

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

			nodes, err := h.db.NodesForUser(userUUID)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}
			if len(nodes) > 0 {
				en := json.NewEncoder(res)
				en.Encode(nodes)
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
			node, err := h.db.Node(id)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
			} else {
				en := json.NewEncoder(res)
				en.Encode(node)
			}
		case http.MethodDelete:
			err := h.db.NodeDelete(id)
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
			h.updateNodeGroups(res, req, id)
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
			cmd, err := h.db.NodeGetCmd(id)
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

func (h *Nodes) processCmd(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var cmd data.NodeCmd
	err := decoder.Decode(&cmd)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// set ID in case it is not set in API call
	cmd.ID = id

	// set cmd in DB for legacy nodes that still fetch over http
	err = h.db.NodeSetCmd(cmd)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO how to support old nodes still fetching commands via http
	// perhaps check if node is connected via NATs
	err = nats.SendCmd(h.nh.Nc, cmd, time.Second*10)
	if err != nil {
		log.Printf("Error sending command (%v) to node: ", err)
		// don't return HTTP error for now as some units still fetch over http
		//http.Error(res, "Error sending command to node", http.StatusInternalServerError)
		//return
	} else {
		err = h.db.NodeDeleteCmd(cmd.ID)
		if err != nil {
			log.Printf("Error deleting command for node %v: %v", id, err)
		}
	}

	// process updates that are now pushed
	if cmd.Cmd == data.CmdUpdateApp {
		log.Printf("Sending %v to node %v\n", cmd.Detail, cmd.ID)
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

func (h *Nodes) updateNodeGroups(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var groups []uuid.UUID
	err := decoder.Decode(&groups)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.db.NodeUpdateGroups(id, groups)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}

func (h *Nodes) processPoints(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var points data.Points
	err := decoder.Decode(&points)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	for _, p := range points {
		err = h.db.NodePoint(id, p)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}
