package data

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

// SwUpdateState represents the state of an update
type SwUpdateState struct {
	Running     bool   `json:"running"`
	Error       string `json:"error"`
	PercentDone int    `json:"percentDone"`
}

// Points converts SW update state to node points
func (sws *SwUpdateState) Points() Points {
	running := 0.0
	if sws.Running {
		running = 1
	}

	pRunning := Point{Type: PointTypeSwUpdateRunning}
	pRunning.PutFloat(running)
	pError := Point{Type: PointTypeSwUpdateError}
	pError.PutString(sws.Error)
	pPercent := Point{Type: PointTypeSwUpdatePercComplete}
	pPercent.PutFloat(float64(sws.PercentDone))
	return Points{pRunning, pError, pPercent}
}

// TODO move Node to db/store package and make it internal to that package

// Node represents the state of a device. UUID is recommended
// for ID to prevent collisions is distributed instances.
type Node struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Points Points `json:"points"`
}

func (n Node) String() string {
	ret := fmt.Sprintf("NODE: %v (%v)\n", n.ID, n.Type)

	for _, p := range n.Points {
		ret += fmt.Sprintf("  - Point: %v\n", p)
	}

	return ret
}

// Desc returns Description if set, otherwise ID
func (n *Node) Desc() string {
	desc := n.Points.Desc()

	if desc != "" {
		return desc
	}

	return n.ID
}

// FIXME all of the below functions need to be modified to go through NATS
// perhaps they should be removed

// GetState checks state of node and
// returns true if state was updated. We originally considered
// offline to be when we did not receive data from a remote device
// for X minutes. However, with points that could represent a config
// change as well. Eventually we may want to improve this to look
// at point types (perhaps Sample).
func (n *Node) GetState() (string, bool) {
	sysState := n.State()
	switch sysState {
	case PointValueSysStateUnknown, PointValueSysStateOnline:
		if time.Since(n.Points.LatestTime()) > 15*time.Minute {
			// mark device as offline
			return PointValueSysStateOffline, true
		}
	}

	return sysState, false
}

// State returns the current state of a device
func (n *Node) State() string {
	s, _ := n.Points.Text(PointTypeSysState, "")
	return s
}

// ToUser converts a node to user struct
func (n *Node) ToUser() User {
	first, _ := n.Points.Text(PointTypeFirstName, "")
	last, _ := n.Points.Text(PointTypeLastName, "")
	phone, _ := n.Points.Text(PointTypePhone, "")
	email, _ := n.Points.Text(PointTypeEmail, "")
	pass, _ := n.Points.Text(PointTypePass, "")

	return User{
		ID:        n.ID,
		FirstName: first,
		LastName:  last,
		Phone:     phone,
		Email:     email,
		Pass:      pass,
	}
}

// ToNodeEdge converts to data structure used in API
// requests
func (n *Node) ToNodeEdge(edge Edge) NodeEdge {
	return NodeEdge{
		ID:         n.ID,
		Type:       n.Type,
		Parent:     edge.Up,
		Points:     n.Points,
		EdgePoints: edge.Points,
		Hash:       edge.Hash,
	}
}

// Nodes defines a list of nodes
type Nodes []NodeEdge

// nodeFrameVersion is the first byte of an encoded node reply. A decoder
// refuses any other value rather than misreading the bytes that follow.
const nodeFrameVersion = 1

// maxNodesPerFrame bounds the node count a decoder will accept.
const maxNodesPerFrame = 10000

// EncodeNodes serializes a node reply. The frame is: a version byte, an
// error string (empty on success), a uint32 node count, then each node as
// id, type, parent, points, and edge points, using the point encoding. The
// hash is not carried.
func EncodeNodes(nodes Nodes, err error) []byte {
	buf := &bytes.Buffer{}
	buf.WriteByte(nodeFrameVersion)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	encodeString(buf, errStr)
	c := make([]byte, 4)
	binary.LittleEndian.PutUint32(c, uint32(len(nodes)))
	buf.Write(c)
	for i := range nodes {
		nodes[i].encode(buf)
	}
	return buf.Bytes()
}

// DecodeNodes deserializes a node reply made by EncodeNodes. An error the
// sender put in the frame is returned as the error; ErrDocumentNotFound is
// returned as that value so callers can compare against it. An empty payload
// decodes to no nodes.
func DecodeNodes(data []byte) ([]NodeEdge, error) {
	if len(data) == 0 {
		return []NodeEdge{}, nil
	}
	if data[0] != nodeFrameVersion {
		return nil, fmt.Errorf("DecodeNodes: unsupported frame version %d", data[0])
	}
	errStr, off, err := decodeString(data, 1)
	if err != nil {
		return nil, fmt.Errorf("DecodeNodes: %w", err)
	}
	if errStr != "" {
		if errStr == ErrDocumentNotFound.Error() {
			return []NodeEdge{}, ErrDocumentNotFound
		}
		return []NodeEdge{}, errors.New(errStr)
	}
	if off+4 > len(data) {
		return nil, fmt.Errorf("DecodeNodes: not enough data for count")
	}
	count := int(binary.LittleEndian.Uint32(data[off : off+4]))
	if count > maxNodesPerFrame {
		return nil, fmt.Errorf("DecodeNodes: count %d exceeds maximum", count)
	}
	off += 4
	ret := make([]NodeEdge, count)
	for i := 0; i < count; i++ {
		ret[i], off, err = decodeNodeEdge(data, off)
		if err != nil {
			return nil, fmt.Errorf("DecodeNodes: error at node %d: %w", i, err)
		}
	}
	return ret, nil
}

// encode writes the node to buf in the EncodeNodes format.
func (n *NodeEdge) encode(buf *bytes.Buffer) {
	encodeString(buf, n.ID)
	encodeString(buf, n.Type)
	encodeString(buf, n.Parent)
	n.Points.encode(buf)
	n.EdgePoints.encode(buf)
}

// decodeNodeEdge reads one node from data at offset and returns the offset
// just past it.
func decodeNodeEdge(data []byte, off int) (NodeEdge, int, error) {
	var n NodeEdge
	var err error
	if n.ID, off, err = decodeString(data, off); err != nil {
		return n, off, err
	}
	if n.Type, off, err = decodeString(data, off); err != nil {
		return n, off, err
	}
	if n.Parent, off, err = decodeString(data, off); err != nil {
		return n, off, err
	}
	if n.Points, off, err = decodePointsAt(data, off); err != nil {
		return n, off, err
	}
	if n.EdgePoints, off, err = decodePointsAt(data, off); err != nil {
		return n, off, err
	}
	return n, off, nil
}

// define valid commands
const (
	CmdUpdateApp = "updateApp"
	CmdPoll      = "poll"
	CmdFieldMode = "fieldMode"
)

// NodeCmd represents a command to be sent to a device
type NodeCmd struct {
	ID     string `json:"id,omitempty"`
	Cmd    string `json:"cmd"`
	Detail string `json:"detail,omitempty"`
}

// NodeVersion represents the device SW version
type NodeVersion struct {
	OS  string `json:"os"`
	App string `json:"app"`
	HW  string `json:"hw"`
}

// FIXME -- seems like we could eventually get rid of node edge if we
// do recursion in the client instead of the server. Then the client
// could keep track of the parents and edges in tree data structures
// on the client.

// NodeEdge combines node and edge data, used for APIs
type NodeEdge struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Hash       uint32 `json:"hash" yaml:"-"`
	Parent     string `json:"parent"`
	Points     Points `json:"points,omitempty"`
	EdgePoints Points `json:"edgePoints,omitempty"`
}

func (n NodeEdge) String() string {
	ret := fmt.Sprintf("NODE: %v (%v)\n", n.ID, n.Type)
	ret += fmt.Sprintf("  - Parent: %v\n", n.Parent)
	for _, p := range n.Points {
		ret += fmt.Sprintf("  - Point: %v\n", p)
	}

	for _, p := range n.EdgePoints {
		ret += fmt.Sprintf("  - Edge point: %v\n", p)
	}

	return ret
}

// IsTombstone returns Tombstone value and timestamp
func (n NodeEdge) IsTombstone() (bool, time.Time) {
	p, _ := n.EdgePoints.Find(PointTypeTombstone, "")
	return p.Bool(), p.Time
}

// Desc returns Description if set, otherwise ID
func (n NodeEdge) Desc() string {
	desc := n.Points.Desc()

	if desc != "" {
		return desc
	}

	return n.ID
}

// FIXME -- should ToNode really be used as it is lossy?

// ToNode converts to structure stored in db
func (n *NodeEdge) ToNode() Node {
	return Node{
		ID:     n.ID,
		Type:   n.Type,
		Points: n.Points,
	}
}

// AddPoint takes a point for a device and adds/updates its array of points
func (n *NodeEdge) AddPoint(pIn Point) {
	n.Points.Add(pIn)
}

// RemoveDuplicateNodesIDParent removes duplicate nodes in list with the
// same ID and parent
func RemoveDuplicateNodesIDParent(nodes []NodeEdge) []NodeEdge {
	keys := make(map[string]bool)
	ret := []NodeEdge{}

	for _, n := range nodes {
		key := n.ID + n.Parent
		if _, ok := keys[key]; !ok {
			keys[key] = true
			ret = append(ret, n)
		}
	}

	return ret
}

// RemoveDuplicateNodesID removes duplicate nodes in list with the
// same ID (can have different parents)
func RemoveDuplicateNodesID(nodes []NodeEdge) []NodeEdge {
	keys := make(map[string]bool)
	ret := []NodeEdge{}

	for _, n := range nodes {
		key := n.ID
		if _, ok := keys[key]; !ok {
			keys[key] = true
			ret = append(ret, n)
		}
	}

	return ret
}
