package api

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

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

// NodeCopy is a data structured used in the /node/:id/parents api call.
// OldParent names the edge the copy was started from, which decides whether a
// mirror of it is marked a mirror.
type NodeCopy struct {
	ID        string
	OldParent string
	NewParent string
	Duplicate bool
}

// NodeDelete is a data structure used with /node/:id DELETE call
type NodeDelete struct {
	Parent string
}

// DeviceAuthorizer answers whether a device public key is enrolled, and for
// which device. The NATS authorizer implements it.
type DeviceAuthorizer interface {
	Device(pubKey string) (deviceID string, ok bool)
}

// Nodes handles node requests
type Nodes struct {
	check     RequestValidator
	nc        *nats.Conn
	authToken string
	// deviceAuth resolves device tokens; nil accepts none.
	deviceAuth DeviceAuthorizer
	// deviceAuthRequired limits the shared token to loopback, the same as
	// the NATS side.
	deviceAuthRequired bool
}

// NewNodesHandler returns a new node handler
func NewNodesHandler(v RequestValidator, authToken string,
	nc *nats.Conn, deviceAuth DeviceAuthorizer, deviceAuthRequired bool) http.Handler {
	return &Nodes{check: v, nc: nc, authToken: authToken,
		deviceAuth: deviceAuth, deviceAuthRequired: deviceAuthRequired}
}

// authenticate works out who a request is from: the shared token (full
// access), a user (JWT from login), or a device (JWT signed with its key,
// limited to its own subtree).
func (h *Nodes) authenticate(req *http.Request) (validUser bool, userID, deviceID string, ok bool) {
	header := req.Header.Get("Authorization")

	if header == h.authToken {
		if h.deviceAuthRequired && !remoteIsLoopback(req.RemoteAddr) {
			return false, "", "", false
		}
		return false, "", "", true
	}

	if valid, id := h.check.Valid(req); valid {
		return true, id, "", true
	}

	if h.deviceAuth != nil {
		if tok, found := strings.CutPrefix(header, "Bearer "); found {
			pubKey, err := client.VerifyDeviceJWT(tok)
			if err == nil {
				if id, enrolled := h.deviceAuth.Device(pubKey); enrolled {
					return false, "", id, true
				}
			}
		}
	}

	return false, "", "", false
}

func remoteIsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// underDevice reports whether node id is the device or below it, following
// parents up the tree.
func (h *Nodes) underDevice(id, deviceID string) bool {
	seen := map[string]bool{}
	frontier := []string{id}

	for len(frontier) > 0 {
		cur := frontier[0]
		frontier = frontier[1:]
		if cur == deviceID {
			return true
		}
		if seen[cur] {
			continue
		}
		seen[cur] = true

		parents, err := client.GetNodes(h.nc, "all", cur, "", false)
		if err != nil {
			return false
		}
		for _, p := range parents {
			frontier = append(frontier, p.Parent)
		}
	}

	return false
}

// Top level handler for http requests in the coap-server process
func (h *Nodes) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	validUser, userID, deviceID, ok := h.authenticate(req)
	if !ok {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if deviceID != "" {
		// a device reads and writes its own subtree and nothing else
		allowed := (head == "" && req.Method == http.MethodGet) ||
			((head == "points" || head == "samples" || head == "not") && req.Method == http.MethodPost)
		if id == "" || !allowed {
			http.Error(res, "Forbidden", http.StatusForbidden)
			return
		}
		if !h.underDevice(id, deviceID) {
			http.Error(res, "Forbidden", http.StatusForbidden)
			return
		}
		userID = deviceID
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
				log.Println("Error getting nodes for user:", err)
			}

			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}
			if len(nodes) > 0 {
				en := json.NewEncoder(res)
				err := en.Encode(nodes)
				if err != nil {
					http.Error(res, "encoding error", http.StatusMethodNotAllowed)
				}
				return
			}
			_, _ = res.Write([]byte("[]"))
		case http.MethodPost:
			// create node
			h.insertNode(res, req, userID)
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
			body, err := io.ReadAll(req.Body)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			parent := string(body)

			node, err := client.GetNodes(h.nc, parent, id, "", false)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
			} else {
				en := json.NewEncoder(res)
				err := en.Encode(node)
				if err != nil {
					http.Error(res, "encoding error", http.StatusMethodNotAllowed)
					return
				}
			}
		case http.MethodDelete:
			var nodeDelete NodeDelete
			if err := decode(req.Body, &nodeDelete); err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			err := client.DeleteNode(h.nc, id, nodeDelete.Parent, userID)

			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			en := json.NewEncoder(res)
			err = en.Encode(data.StandardResponse{Success: true, ID: id})
			if err != nil {
				http.Error(res, "encoding error", http.StatusMethodNotAllowed)

			}
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
				nodeMove.NewParent, userID)

			if err != nil {
				log.Println("Error moving node:", err)
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			en := json.NewEncoder(res)
			err = en.Encode(data.StandardResponse{Success: true, ID: id})
			if err != nil {
				http.Error(res, "encoding error", http.StatusMethodNotAllowed)
			}

		case http.MethodPut:
			var nodeCopy NodeCopy
			if err := decode(req.Body, &nodeCopy); err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			if !nodeCopy.Duplicate {
				err := client.MirrorNode(h.nc, id, nodeCopy.OldParent,
					nodeCopy.NewParent, userID)

				if err != nil {
					log.Println("Error mirroring node:", err)
					http.Error(res, err.Error(), http.StatusNotFound)
					return
				}
			} else {
				err := client.DuplicateNode(h.nc, id, nodeCopy.NewParent, userID)

				if err != nil {
					log.Println("Error duplicating node:", err)
					http.Error(res, err.Error(), http.StatusNotFound)
					return
				}
			}

			en := json.NewEncoder(res)
			err := en.Encode(data.StandardResponse{Success: true, ID: id})
			if err != nil {
				http.Error(res, "encoding error", http.StatusMethodNotAllowed)
			}

			return

		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
		}

	case "key":
		if req.Method == http.MethodPost {
			h.generateKey(res, id, userID)
			return
		}

		http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		return

	case "not":
		switch req.Method {
		case http.MethodPost:
			var not data.Notification
			if err := decode(req.Body, &not); err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			not.ID = uuid.New().String()
			if not.SourceNode == "" {
				not.SourceNode = id
			}

			p, err := not.Point()

			if err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			p.Origin = userID

			err = client.SendNodePoint(h.nc, id, p, true)

			if err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			en := json.NewEncoder(res)
			err = en.Encode(data.StandardResponse{Success: true, ID: id})
			if err != nil {
				http.Error(res, "encoding error", http.StatusMethodNotAllowed)
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

func (h *Nodes) insertNode(res http.ResponseWriter, req *http.Request, userID string) {
	var node data.NodeEdge
	if err := decode(req.Body, &node); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if node.ID == "" {
		node.ID = uuid.New().String()
	}

	// populate origin for all points
	for i := range node.Points {
		node.Points[i].Origin = userID
	}

	err := client.SendNode(h.nc, node, userID)

	if err != nil {
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	err = encode(res, data.StandardResponse{Success: true, ID: node.ID})
	if err != nil {
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}
}

// generateKey makes a key pair for a deviceCred node: the public key is
// written on the node and the seed is returned once, for the operator to
// carry to the device. Nothing stores the seed.
func (h *Nodes) generateKey(res http.ResponseWriter, id, userID string) {
	nodes, err := client.GetNodes(h.nc, "all", id, "", false)
	if err != nil || len(nodes) == 0 {
		http.Error(res, "node not found", http.StatusNotFound)
		return
	}

	if nodes[0].Type != data.NodeTypeDeviceCred {
		http.Error(res, "not a device credential node", http.StatusBadRequest)
		return
	}

	seed, pubKey, err := client.GenerateDeviceKey()
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	p := data.NewPointString(data.PointTypePubKey, "", pubKey)
	p.Origin = userID
	if err := client.SendNodePoint(h.nc, id, p, true); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := encode(res, DeviceKeyResponse{PubKey: pubKey, Seed: seed}); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Nodes) processPoints(res http.ResponseWriter, req *http.Request, id, userID string) {
	decoder := json.NewDecoder(req.Body)
	var points data.Points
	err := decoder.Decode(&points)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// populate origin for all points
	for i := range points {
		points[i].Origin = userID
		//points[i].Time = time.Now()
	}

	err = client.SendNodePoints(h.nc, id, points, true)

	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	en := json.NewEncoder(res)
	err = en.Encode(data.StandardResponse{Success: true, ID: id})
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
}
