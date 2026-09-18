package api

import (
	"crypto/subtle"
	"errors"
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
	users     UserAuthority
	nc        *nats.Conn
	authToken string
	// deviceAuth resolves device tokens; nil accepts none.
	deviceAuth DeviceAuthorizer
	// deviceAuthRequired limits the shared token to loopback, the same as
	// the NATS side.
	deviceAuthRequired bool
}

// NewNodesHandler returns a new node handler
func NewNodesHandler(users UserAuthority, authToken string,
	nc *nats.Conn, deviceAuth DeviceAuthorizer, deviceAuthRequired bool) http.Handler {
	return &Nodes{users: users, nc: nc, authToken: authToken,
		deviceAuth: deviceAuth, deviceAuthRequired: deviceAuthRequired}
}

// principal is who a request is from, and what it may reach. Exactly one
// of full, userID, and deviceID is set.
type principal struct {
	// full is the shared token: every node.
	full bool
	// userID is a signed-in user, limited to the subtrees under anchors.
	userID  string
	anchors []string
	// deviceID is a device with a signed JWT, limited to its own subtree.
	deviceID string
}

// origin is what the principal's writes are stamped with.
func (p principal) origin() string {
	switch {
	case p.userID != "":
		return p.userID
	case p.deviceID != "":
		return p.deviceID
	}
	return ""
}

// authenticate works out who a request is from: the shared token (full
// access), a user (JWT from login), or a device (JWT signed with its key).
// A request that is none of these is refused; in particular an instance
// with no token configured does not accept a request with no credentials.
func (h *Nodes) authenticate(req *http.Request) (principal, bool) {
	header := req.Header.Get("Authorization")

	if h.authToken != "" && len(header) == len(h.authToken) &&
		subtle.ConstantTimeCompare([]byte(header), []byte(h.authToken)) == 1 {
		if h.deviceAuthRequired && !remoteIsLoopback(req.RemoteAddr) {
			log.Printf("HTTP auth: refusing shared token from %v, device auth is required",
				req.RemoteAddr)
			return principal{}, false
		}
		return principal{full: true}, true
	}

	tok, found := strings.CutPrefix(header, "Bearer ")
	if !found || tok == "" {
		return principal{}, false
	}

	if h.users != nil {
		if userID, _, ok := h.users.UserFromToken(tok); ok {
			// the token is good; the user also has to still be in the
			// tree, or a deleted user would keep access until it
			// expired
			anchors := h.users.UserAnchors(userID)
			if len(anchors) == 0 {
				log.Printf("HTTP auth: refusing user %v from %v, user is not in the tree",
					userID, req.RemoteAddr)
				return principal{}, false
			}
			return principal{userID: userID, anchors: anchors}, true
		}
	}

	if h.deviceAuth != nil {
		pubKey, err := client.VerifyDeviceJWT(tok)
		if err == nil {
			if id, enrolled := h.deviceAuth.Device(pubKey); enrolled {
				return principal{deviceID: id}, true
			}
		}
	}

	return principal{}, false
}

func remoteIsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// allows reports whether a principal may act on a node: every node for the
// shared token, a node under one of the user's anchors, or a node under
// the device.
func (h *Nodes) allows(p principal, id string) bool {
	switch {
	case p.full:
		return true
	case id == "":
		return false
	case p.userID != "":
		for _, a := range p.anchors {
			if h.users.IsUnder(id, a) {
				return true
			}
		}
		return false
	case p.deviceID != "":
		return h.underDevice(id, p.deviceID)
	}
	return false
}

// allowsAll is allows for every ID given; an empty ID is skipped, since
// the routes treat a missing parent as "any".
func (h *Nodes) allowsAll(p principal, ids ...string) bool {
	for _, id := range ids {
		if id == "" || id == "all" {
			continue
		}
		if !h.allows(p, id) {
			return false
		}
	}
	return true
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

func forbidden(res http.ResponseWriter) {
	http.Error(res, "Forbidden", http.StatusForbidden)
}

// Top level handler for http requests in the coap-server process
func (h *Nodes) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	var id string
	id, req.URL.Path = ShiftPath(req.URL.Path)

	var head string
	head, req.URL.Path = ShiftPath(req.URL.Path)

	p, ok := h.authenticate(req)
	if !ok {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if p.deviceID != "" {
		// a device reads and writes its own subtree and nothing else
		allowed := (head == "" && req.Method == http.MethodGet) ||
			((head == "points" || head == "samples" || head == "not") && req.Method == http.MethodPost)
		if id == "" || !allowed {
			forbidden(res)
			return
		}
	}

	if id == "" {
		switch req.Method {
		case http.MethodPost:
			// create node
			h.insertNode(res, req, p)
		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
			return
		}
		return
	}

	// every route with an ID acts on that node
	if !h.allows(p, id) {
		forbidden(res)
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
			if !h.allowsAll(p, parent) {
				forbidden(res)
				return
			}

			nodes, err := client.GetNodes(h.nc, parent, id, "", false)
			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			if !p.full {
				// the reply lists the node under each parent; keep
				// the parents the principal may see, and leave the
				// values of secrets out
				kept := nodes[:0]
				for _, n := range nodes {
					if h.allows(p, n.Parent) || h.isAnchor(p, n.ID) {
						kept = append(kept, n)
					}
				}
				nodes = data.RedactNodes(kept)
			}

			if err := encode(res, nodes); err != nil {
				http.Error(res, "encoding error", http.StatusInternalServerError)
			}
		case http.MethodDelete:
			var nodeDelete NodeDelete
			if err := decode(req.Body, &nodeDelete); err != nil {
				decodeError(res, err)
				return
			}

			if !h.allowsAll(p, nodeDelete.Parent) {
				forbidden(res)
				return
			}

			err := client.DeleteNode(h.nc, id, nodeDelete.Parent, p.origin())

			if err != nil {
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			if err := encode(res, data.StandardResponse{Success: true, ID: id}); err != nil {
				http.Error(res, "encoding error", http.StatusInternalServerError)
			}
		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
			return
		}

	case "samples", "points":
		if req.Method == http.MethodPost {
			h.processPoints(res, req, id, p)
			return
		}

		http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		return

	case "parents":
		switch req.Method {
		case http.MethodPost:
			var nodeMove NodeMove
			if err := decode(req.Body, &nodeMove); err != nil {
				decodeError(res, err)
				return
			}

			// both parents have to be in scope, or a node could be
			// moved into or out of the principal's reach
			if nodeMove.NewParent == "" || !h.allowsAll(p, nodeMove.OldParent, nodeMove.NewParent) {
				forbidden(res)
				return
			}

			err := client.MoveNode(h.nc, id, nodeMove.OldParent,
				nodeMove.NewParent, p.origin())

			if err != nil {
				log.Println("Error moving node:", err)
				http.Error(res, err.Error(), http.StatusNotFound)
				return
			}

			if err := encode(res, data.StandardResponse{Success: true, ID: id}); err != nil {
				http.Error(res, "encoding error", http.StatusInternalServerError)
			}

		case http.MethodPut:
			var nodeCopy NodeCopy
			if err := decode(req.Body, &nodeCopy); err != nil {
				decodeError(res, err)
				return
			}

			if nodeCopy.NewParent == "" || !h.allowsAll(p, nodeCopy.OldParent, nodeCopy.NewParent) {
				forbidden(res)
				return
			}

			if !nodeCopy.Duplicate {
				err := client.MirrorNode(h.nc, id, nodeCopy.OldParent,
					nodeCopy.NewParent, p.origin())

				if err != nil {
					log.Println("Error mirroring node:", err)
					http.Error(res, err.Error(), http.StatusNotFound)
					return
				}
			} else {
				err := client.DuplicateNode(h.nc, id, nodeCopy.NewParent, p.origin())

				if err != nil {
					log.Println("Error duplicating node:", err)
					http.Error(res, err.Error(), http.StatusNotFound)
					return
				}
			}

			if err := encode(res, data.StandardResponse{Success: true, ID: id}); err != nil {
				http.Error(res, "encoding error", http.StatusInternalServerError)
			}

			return

		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
		}

	case "key":
		if req.Method == http.MethodPost {
			h.generateKey(res, id, p)
			return
		}

		http.Error(res, "only POST allowed", http.StatusMethodNotAllowed)
		return

	case "not":
		switch req.Method {
		case http.MethodPost:
			var not data.Notification
			if err := decode(req.Body, &not); err != nil {
				decodeError(res, err)
				return
			}

			not.ID = uuid.New().String()
			if not.SourceNode == "" {
				not.SourceNode = id
			}

			pt, err := not.Point()

			if err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			pt.Origin = p.origin()

			err = client.SendNodePoint(h.nc, id, pt, true)

			if err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}

			if err := encode(res, data.StandardResponse{Success: true, ID: id}); err != nil {
				http.Error(res, "encoding error", http.StatusInternalServerError)
			}
		default:
			http.Error(res, "invalid method", http.StatusMethodNotAllowed)
		}

	default:
		http.Error(res, "Not Found", http.StatusNotFound)
	}
}

// isAnchor reports whether a node is the top of what the principal may
// see: one of a user's anchors, or the device itself. Its edge is listed
// even though the parent is out of scope, since the UI needs it to show
// the top of the tree.
func (h *Nodes) isAnchor(p principal, id string) bool {
	if p.deviceID != "" && id == p.deviceID {
		return true
	}
	for _, a := range p.anchors {
		if a == id {
			return true
		}
	}
	return false
}

func (h *Nodes) insertNode(res http.ResponseWriter, req *http.Request, p principal) {
	var node data.NodeEdge
	if err := decode(req.Body, &node); err != nil {
		decodeError(res, err)
		return
	}

	if !h.allows(p, node.Parent) {
		forbidden(res)
		return
	}

	if node.ID == "" {
		node.ID = uuid.New().String()
	} else if !p.full {
		// an ID the caller chose may name a node that already exists,
		// which would attach it under the parent: only allowed when
		// that node is already in scope, or is new to the tree
		existing, err := client.GetNodes(h.nc, "all", node.ID, "", true)
		if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(existing) > 0 && !h.allows(p, node.ID) {
			forbidden(res)
			return
		}
	}

	// populate origin for all points
	for i := range node.Points {
		node.Points[i].Origin = p.origin()
	}

	err := client.SendNode(h.nc, node, p.origin())

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

// KeyResponse is the reply to a key request. The token is returned once and
// never stored.
type KeyResponse struct {
	Token string `json:"token"`
}

// generateKey makes a token for an enrollToken node: the hash is written on
// the node and the token is returned once.
func (h *Nodes) generateKey(res http.ResponseWriter, id string, p principal) {
	nodes, err := client.GetNodes(h.nc, "all", id, "", false)
	if err != nil || len(nodes) == 0 {
		http.Error(res, "node not found", http.StatusNotFound)
		return
	}

	if nodes[0].Type != data.NodeTypeEnrollToken {
		http.Error(res, "not an enrollment token node", http.StatusBadRequest)
		return
	}

	token, hash, err := client.GenerateEnrollToken()
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	pt := data.NewPointString(data.PointTypeTokenHash, "", hash)
	pt.Origin = p.origin()
	if err := client.SendNodePoint(h.nc, id, pt, true); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := encode(res, KeyResponse{Token: token}); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Nodes) processPoints(res http.ResponseWriter, req *http.Request, id string, p principal) {
	var points data.Points
	err := decode(req.Body, &points)
	if err != nil {
		decodeError(res, err)
		return
	}

	// populate origin for all points
	for i := range points {
		points[i].Origin = p.origin()
	}

	err = client.SendNodePoints(h.nc, id, points, true)

	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := encode(res, data.StandardResponse{Success: true, ID: id}); err != nil {
		http.Error(res, "encoding error", http.StatusInternalServerError)
	}
}
