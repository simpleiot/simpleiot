package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
	"github.com/simpleiot/simpleiot/msg"
	"github.com/simpleiot/simpleiot/nats"
)

// NodeMove is a data structure used in the /node/parent api calls
type NodeMove struct {
	ID        string
	OldParent string
	NewParent string
}

// Nodes handles node requests
type Nodes struct {
	db        *genji.Db
	check     RequestValidator
	nh        *NatsHandler
	authToken string
	messenger *msg.Messenger
}

// NewNodesHandler returns a new node handler
func NewNodesHandler(db *genji.Db, v RequestValidator, authToken string,
	nh *NatsHandler, messenger *msg.Messenger) http.Handler {
	return &Nodes{db, v, nh, authToken, messenger}
}

// Top level handler for http requests in the coap-server process
// TODO need to add node auth
func (h *Nodes) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	var validUser bool
	var userID string

	if req.Header.Get("Authorization") != h.authToken {
		// all requests require valid JWT or authToken validation
		validUser, userID = h.check.Valid(req)

		if !validUser {
			http.Error(res, "Unauthorized", http.StatusUnauthorized)
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

			nodes, err := h.db.NodesForUser(userID)
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
		case http.MethodPost:
			// create node
			h.insertNode(res, req)
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

	case "parents":
		switch req.Method {
		case http.MethodPost:
			var nodeMove NodeMove
			if err := decode(req.Body, &nodeMove); err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
			err := h.db.EdgeMove(nodeMove.ID, nodeMove.OldParent,
				nodeMove.NewParent)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
			} else {
				en := json.NewEncoder(res)
				en.Encode(data.StandardResponse{Success: true, ID: id})
			}
			return
		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
		}

	case "msg":
		switch req.Method {
		case http.MethodPost:
			var point data.Point
			if err := decode(req.Body, &point); err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			// FIXME report errors to user
			var err error
			if point.Text != "" {
				// send message to all users
				// TODO do we want to notify children or all descendents here?
				nodes, err := h.db.NodeDescendents(id, data.NodeTypeUser, false)
				if err != nil {
					http.Error(res, err.Error(), http.StatusInternalServerError)
					return
				}

				for _, ne := range nodes {
					n := ne.ToNode()
					u := n.ToUser()
					if u.Phone != "" {
						err := h.messenger.SendSMS(u.Phone, point.Text)
						if err != nil {
							log.Println("Error sending message: ", err)
						}
					}
				}
			}

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

// RequestValidator validates an HTTP request.
type RequestValidator interface {
	Valid(req *http.Request) (bool, string)
}

func (h *Nodes) insertNode(res http.ResponseWriter, req *http.Request) {
	var node data.NodeEdge
	if err := decode(req.Body, &node); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.db.NodeInsertEdge(node)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: id})
}

func (h *Nodes) processPoints(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var points data.Points
	err := decoder.Decode(&points)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = nats.SendPoints(h.nh.Nc, id, points, true)

	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}
