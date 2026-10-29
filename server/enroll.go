package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// enroller answers enrollment requests: a device that holds an enrollment
// token asks for a credential for its key, and gets one under its device
// node, pending approval unless the token approves automatically. The
// device node is created when it does not exist yet, under the node the
// token sits in, so a unit fresh from an image shows up in its group with
// its credential waiting.
type enroller struct {
	nc   *nats.Conn
	auth *authorizer
	sub  *nats.Subscription
}

func (e *enroller) start(nc *nats.Conn, auth *authorizer) error {
	e.nc = nc
	e.auth = auth

	var err error
	e.sub, err = nc.Subscribe(client.SubjectEnrollRequest, e.handle)
	if err != nil {
		return fmt.Errorf("error subscribing to enrollment requests: %w", err)
	}

	return nil
}

func (e *enroller) stop() {
	if e.sub != nil {
		_ = e.sub.Unsubscribe()
	}
}

func (e *enroller) reply(msg *nats.Msg, r client.EnrollReply) {
	b, _ := json.Marshal(r)
	if err := e.nc.Publish(msg.Reply, b); err != nil {
		log.Println("Error replying to enrollment request:", err)
	}
}

func (e *enroller) handle(msg *nats.Msg) {
	var req client.EnrollRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		e.reply(msg, client.EnrollReply{Error: "bad request"})
		return
	}

	status, err := e.enroll(req, msg.Reply)
	if err != nil {
		log.Printf("Enrollment refused for device %v: %v", req.DeviceID, err)
		e.reply(msg, client.EnrollReply{Error: err.Error()})
		return
	}

	log.Printf("Enrollment %v for device %v, key %v", status, req.DeviceID, req.PubKey)
	e.reply(msg, client.EnrollReply{Status: status})
}

// maxPendingEnrollments bounds how many devices may wait for approval at
// once, so a token that has leaked cannot fill the tree with device nodes.
const maxPendingEnrollments = 100

// enroll validates a request and creates what it needs. The token is
// checked again here even though the connection was made with it, since
// the request does not say which connection it came from; the reply
// subject does, since only the connection holding the key can subscribe
// to that key's inbox.
func (e *enroller) enroll(req client.EnrollRequest, reply string) (string, error) {
	tok, ok := e.auth.EnrollToken(req.Token)
	if !ok {
		return "", errors.New("enrollment token refused")
	}

	if err := data.CheckSubjectToken("device ID", req.DeviceID); err != nil {
		return "", err
	}
	if !nkeys.IsValidPublicUserKey(req.PubKey) {
		return "", errors.New("public key is not a user key")
	}
	if !strings.HasPrefix(reply, client.InboxPrefix(req.PubKey)+".") {
		return "", errors.New("key does not match the connection")
	}

	parent, err := e.destination(tok)
	if err != nil {
		return "", err
	}

	nodes, err := client.GetNodes(e.nc, "all", req.DeviceID, "", true)
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		return "", fmt.Errorf("error reading device node: %w", err)
	}

	switch {
	case len(nodes) == 0:
		n, err := e.pendingDevices()
		if err != nil {
			return "", err
		}
		if n >= maxPendingEnrollments {
			return "", errors.New("too many devices are waiting for approval")
		}
		dev := client.Device{ID: req.DeviceID, Parent: parent, Description: req.Description}
		if err := client.SendNodeType(e.nc, dev, "enroll"); err != nil {
			return "", fmt.Errorf("error creating device node: %w", err)
		}
	case !anyLive(nodes):
		// only the upstream restores a detached device
		return "", errors.New("device is detached on this instance")
	}

	// a device that already exists is known, wherever it sits now; a key
	// for it waits for an operator whatever the token says, so a token
	// holder cannot take over a device by naming its ID, nor undo a
	// credential that was disabled to revoke it
	created := len(nodes) == 0

	creds, err := client.GetNodesType[client.DeviceCred](e.nc, req.DeviceID, "all")
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		return "", fmt.Errorf("error reading credentials: %w", err)
	}

	for _, c := range creds {
		if c.PubKey == req.PubKey {
			if c.Pending || c.Disabled {
				return client.EnrollPending, nil
			}
			return client.EnrollApproved, nil
		}
	}

	autoApprove := tok.autoApprove && created

	cred := client.DeviceCred{
		ID:          data.NewID(),
		Parent:      req.DeviceID,
		Description: "enrolled " + time.Now().Format(time.RFC3339),
		PubKey:      req.PubKey,
		Pending:     !autoApprove,
		EnrolledBy:  tok.nodeID,
	}
	if err := client.SendNodeType(e.nc, cred, "enroll"); err != nil {
		return "", fmt.Errorf("error creating credential: %w", err)
	}

	if autoApprove {
		return client.EnrollApproved, nil
	}

	return client.EnrollPending, nil
}

// errTokenPlacement is what a device is told when its token sits where it
// cannot enroll. The reason, which names nodes, goes to the log only.
var errTokenPlacement = errors.New("enrollment token is not in the root or a group")

// destination returns the node a token creates new devices under: its one
// parent, when that is the root or a group still in the tree. A token in
// two places is ambiguous, and one under a device or a deleted group would
// put devices where nobody looks.
func (e *enroller) destination(tok enrollTokenInfo) (string, error) {
	if len(tok.parents) != 1 {
		log.Printf("Enrollment token %v is in %v places, refusing it; use one token per group",
			tok.nodeID, len(tok.parents))
		return "", errTokenPlacement
	}

	parent := tok.parents[0]
	rootID := e.auth.root()
	if parent == rootID {
		return parent, nil
	}

	edges, err := client.GetNodes(e.nc, "all", parent, "", false)
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		return "", fmt.Errorf("error reading enrollment token parent: %w", err)
	}
	if len(edges) == 0 || edges[0].Type != data.NodeTypeGroup {
		typ := "deleted"
		if len(edges) > 0 {
			typ = edges[0].Type
		}
		log.Printf("Enrollment token %v is under a %v node, not the root or a group, refusing it",
			tok.nodeID, typ)
		return "", errTokenPlacement
	}

	ok, err := e.inTree(edges, rootID)
	if err != nil {
		return "", err
	}
	if !ok {
		log.Printf("Enrollment token %v is under group %v, which is no longer in the tree, refusing it",
			tok.nodeID, parent)
		return "", errTokenPlacement
	}

	return parent, nil
}

// inTree reports whether a path of live edges leads from a node, given its
// own live edges, up to the root. Deleting a group tombstones only the
// group's own edges, so a group under a deleted group still has a live edge
// of its own.
func (e *enroller) inTree(edges []data.NodeEdge, rootID string) (bool, error) {
	seen := map[string]bool{}
	var queue []string
	for _, ed := range edges {
		queue = append(queue, ed.Parent)
	}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if id == rootID {
			return true, nil
		}
		if seen[id] {
			continue
		}
		seen[id] = true

		up, err := client.GetNodes(e.nc, "all", id, "", false)
		if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
			return false, fmt.Errorf("error reading node %v: %w", id, err)
		}
		for _, ed := range up {
			queue = append(queue, ed.Parent)
		}
	}

	return false, nil
}

// pendingDevices counts the devices with a credential waiting for
// approval.
func (e *enroller) pendingDevices() (int, error) {
	creds, err := client.GetNodesType[client.DeviceCred](e.nc, "all", "all")
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		return 0, fmt.Errorf("error reading credentials: %w", err)
	}
	devices := map[string]bool{}
	for _, c := range creds {
		if c.Pending && !c.Disabled {
			devices[c.Parent] = true
		}
	}
	return len(devices), nil
}

func anyLive(nodes []data.NodeEdge) bool {
	for _, n := range nodes {
		if ts, _ := n.IsTombstone(); !ts {
			return true
		}
	}
	return false
}
