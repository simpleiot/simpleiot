package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// enroller answers enrollment requests: a device that holds an enrollment
// token asks for a credential for its key, and gets one under its device
// node, pending approval unless the token approves automatically. The
// device node is created when it does not exist yet, so a unit fresh from
// an image shows up in the tree with its credential waiting.
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

	status, err := e.enroll(req)
	if err != nil {
		log.Printf("Enrollment refused for device %v: %v", req.DeviceID, err)
		e.reply(msg, client.EnrollReply{Error: err.Error()})
		return
	}

	log.Printf("Enrollment %v for device %v, key %v", status, req.DeviceID, req.PubKey)
	e.reply(msg, client.EnrollReply{Status: status})
}

// enroll validates a request and creates what it needs. The token is
// checked again here even though the connection was made with it, since
// the request does not say which connection it came from.
func (e *enroller) enroll(req client.EnrollRequest) (string, error) {
	autoApprove, ok := e.auth.EnrollToken(req.Token)
	if !ok {
		return "", errors.New("enrollment token refused")
	}

	if req.DeviceID == "" || !nkeys.IsValidPublicUserKey(req.PubKey) {
		return "", errors.New("device ID and public key are required")
	}

	rootID := e.auth.root()

	nodes, err := client.GetNodes(e.nc, "all", req.DeviceID, "", true)
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		return "", fmt.Errorf("error reading device node: %w", err)
	}

	switch {
	case len(nodes) == 0:
		dev := client.Device{ID: req.DeviceID, Parent: rootID, Description: req.Description}
		if err := client.SendNodeType(e.nc, dev, "enroll"); err != nil {
			return "", fmt.Errorf("error creating device node: %w", err)
		}
	case !anyLive(nodes):
		// only the upstream restores a detached device
		return "", errors.New("device is detached on this instance")
	}

	creds, err := client.GetNodesType[client.DeviceCred](e.nc, req.DeviceID, "all")
	if err != nil && !errors.Is(err, data.ErrDocumentNotFound) {
		return "", fmt.Errorf("error reading credentials: %w", err)
	}

	for _, c := range creds {
		if c.PubKey != req.PubKey {
			continue
		}
		if c.Pending || c.Disabled {
			return client.EnrollPending, nil
		}
		return client.EnrollApproved, nil
	}

	cred := client.DeviceCred{
		ID:          uuid.New().String(),
		Parent:      req.DeviceID,
		Description: "enrolled " + time.Now().Format(time.RFC3339),
		PubKey:      req.PubKey,
		Pending:     !autoApprove,
	}
	if err := client.SendNodeType(e.nc, cred, "enroll"); err != nil {
		return "", fmt.Errorf("error creating credential: %w", err)
	}

	if autoApprove {
		return client.EnrollApproved, nil
	}

	return client.EnrollPending, nil
}

func anyLive(nodes []data.NodeEdge) bool {
	for _, n := range nodes {
		if ts, _ := n.IsTombstone(); !ts {
			return true
		}
	}
	return false
}
