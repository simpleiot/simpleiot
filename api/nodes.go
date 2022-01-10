package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
	"github.com/simpleiot/simpleiot/store"

	natsgo "github.com/nats-io/nats.go"
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
}

// NodeDelete is a data structure used with /node/:id DELETE call
type NodeDelete struct {
	Parent string
}

// Nodes handles node requests
type Nodes struct {
	db        *store.Db
	check     RequestValidator
	nc        *natsgo.Conn
	authToken string
}

// NewNodesHandler returns a new node handler
func NewNodesHandler(db *store.Db, v RequestValidator, authToken string,
	nc *natsgo.Conn) http.Handler {
	return &Nodes{db, v, nc, authToken}
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

			{
				nodes, err := nats.GetNodesForUser(h.nc, userID)
				if err != nil {
					log.Println("Error getting nodes for user: ", err)
				}
				_ = nodes
			}

			// FIXME, replace this with a NATS call so we can remove db from this
			// module
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
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			node, err := nats.GetNode(h.nc, id, string(body))
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

			err := nats.SendEdgePoint(h.nc, id, nodeDelete.Parent, data.Point{
				Type:  data.PointTypeTombstone,
				Value: 1,
			}, true)

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

			if nodeMove.NewParent == nodeMove.OldParent {
				http.Error(res, "can't move node to itself", http.StatusNotFound)
				return
			}

			err := nats.SendEdgePoint(h.nc, id, nodeMove.NewParent, data.Point{
				Type:  data.PointTypeTombstone,
				Value: 0,
			}, true)

			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			err = nats.SendEdgePoint(h.nc, id, nodeMove.OldParent, data.Point{
				Type:  data.PointTypeTombstone,
				Value: 1,
			}, true)

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

			err := nats.SendEdgePoint(h.nc, id, nodeCopy.NewParent, data.Point{
				Type:  data.PointTypeTombstone,
				Value: 0,
			}, true)

			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
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

	node.Points = append(node.Points, data.Point{
		Type: data.PointTypeNodeType,
		Text: node.Type,
	})

	err := nats.SendNodePoints(h.nc, node.ID, node.Points, true)

	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = nats.SendEdgePoint(h.nc, node.ID, node.Parent, data.Point{
		Type:  data.PointTypeTombstone,
		Value: 0,
	}, true)

	if err != nil {
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	encode(res, data.StandardResponse{Success: true, ID: node.ID})
}

func (h *Nodes) processPoints(res http.ResponseWriter, req *http.Request, id string) {
	decoder := json.NewDecoder(req.Body)
	var points data.Points
	err := decoder.Decode(&points)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = nats.SendNodePointsCreate(h.nc, id, points, true)

	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	en := json.NewEncoder(res)
	en.Encode(data.StandardResponse{Success: true, ID: id})
}
