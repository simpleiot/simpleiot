package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// NodeMove is a data structure used in the /node/:id/parents api call
type NodeMove struct {
	ID        string
	OldParent string
	NewParent string
}

// NodeCopy is a data structured used in the /node/:id/parents api call
type NodeCopy struct {
	ID        string
	NewParent string
	Duplicate bool
}

// NodeDelete is a data structure used with /node/:id DELETE call
type NodeDelete struct {
	Parent string
}

// Nodes handles node requests
type Nodes struct {
	check     RequestValidator
	nc        *nats.Conn
	authToken string
}

// NewNodesHandler returns a new node handler
func NewNodesHandler(v RequestValidator, authToken string,
	nc *nats.Conn) http.Handler {
	return &Nodes{v, nc, authToken}
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

			nodes, err := client.GetNodesForUser(h.nc, userID)
			if err != nil {
				log.Println("Error getting nodes for user: ", err)
			}

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
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			node, err := client.GetNode(h.nc, id, string(body))
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
			} else {
				en := json.NewEncoder(res)
				en.Encode(node)
			}
		case http.MethodDelete:
			var nodeDelete NodeDelete
			if err := decode(req.Body, &nodeDelete); err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			err := client.DeleteNode(h.nc, id, nodeDelete.Parent)

			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			en := json.NewEncoder(res)
			en.Encode(data.StandardResponse{Success: true, ID: id})
		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
			return
		}

	case "samples", "points":
		if req.Method == http.MethodPost {
			h.processPoints(res, req, id, userID)
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

			err := client.MoveNode(h.nc, id, nodeMove.OldParent,
				nodeMove.NewParent)

			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			en := json.NewEncoder(res)
			en.Encode(data.StandardResponse{Success: true, ID: id})
			return

		case http.MethodPut:
			var nodeCopy NodeCopy
			if err := decode(req.Body, &nodeCopy); err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			if !nodeCopy.Duplicate {
				err := client.MirrorNode(h.nc, id, nodeCopy.NewParent)

				if err != nil {
					http.Error(res, err.Error(), http.StatusNotFound)
					return
				}
			} else {
				err := client.DuplicateNode(h.nc, id, nodeCopy.NewParent)

				if err != nil {
					http.Error(res, err.Error(), http.StatusNotFound)
					return
				}
			}

			en := json.NewEncoder(res)
			en.Encode(data.StandardResponse{Success: true, ID: id})

			return

		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
		}

	case "not":
		switch req.Method {
		case http.MethodPost:
			var not data.Notification
			if err := decode(req.Body, &not); err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			not.ID = uuid.New().String()

			d, err := not.ToPb()

			if err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			err = h.nc.Publish("node."+id+".not", d)

			if err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			en := json.NewEncoder(res)
			en.Encode(data.StandardResponse{Success: true, ID: id})

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

	if node.ID == "" {
		node.ID = uuid.New().String()
	}

	err := client.SendNode(h.nc, node)

	if err != nil {
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: node.ID})
}

func (h *Nodes) processPoints(res http.ResponseWriter, req *http.Request, id, userID string) {
	decoder := json.NewDecoder(req.Body)
	var points data.Points
	err := decoder.Decode(&points)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// populate orgin for all points
	for i := range points {
		points[i].Origin = userID
	}

	err = client.SendNodePointsCreate(h.nc, id, points, true)

	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}
