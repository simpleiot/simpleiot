package data

import (
	"time"

	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
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

	return Points{
		Point{
			Type:  PointTypeSwUpdateRunning,
			Value: running,
		},
		Point{
			Type: PointTypeSwUpdateError,
			Text: sws.Error,
		},
		Point{
			Type:  PointTypeSwUpdatePercComplete,
			Value: float64(sws.PercentDone),
		}}
}

// Node represents the state of a device. UUID is recommended
// for ID to prevent collisions is distributed instances.
type Node struct {
	ID     string `json:"id" boltholdKey:"ID"`
	Type   string `json:"type"`
	Hash   []byte `json:"hash"`
	Points Points `json:"points"`
}

// Desc returns Description if set, otherwise ID
func (n *Node) Desc() string {
	firstName, _ := n.Points.Text("", PointTypeFirstName, 0)
	if firstName != "" {
		lastName, _ := n.Points.Text("", PointTypeLastName, 0)
		if lastName == "" {
			return firstName
		}

		return firstName + " " + lastName
	}

	desc, _ := n.Points.Text("", PointTypeDescription, 0)
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
	s, _ := n.Points.Text("", PointTypeSysState, 0)
	return s
}

// ToUser converts a node to user struct
func (n *Node) ToUser() User {
	first, _ := n.Points.Text("", PointTypeFirstName, 0)
	last, _ := n.Points.Text("", PointTypeLastName, 0)
	phone, _ := n.Points.Text("", PointTypePhone, 0)
	email, _ := n.Points.Text("", PointTypeEmail, 0)
	pass, _ := n.Points.Text("", PointTypePass, 0)

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
func (n *Node) ToNodeEdge(parent string) NodeEdge {
	return NodeEdge{
		ID:     n.ID,
		Type:   n.Type,
		Parent: parent,
		Points: n.Points,
	}
}

// define valid commands
const (
	CmdUpdateApp string = "updateApp"
	CmdPoll             = "poll"
	CmdFieldMode        = "fieldMode"
)

// NodeCmd represents a command to be sent to a device
type NodeCmd struct {
	ID     string `json:"id,omitempty" boltholdKey:"ID"`
	Cmd    string `json:"cmd"`
	Detail string `json:"detail,omitempty"`
}

// NodeVersion represents the device SW version
type NodeVersion struct {
	OS  string `json:"os"`
	App string `json:"app"`
	HW  string `json:"hw"`
}

// NodeEdge combines node and edge data, used for APIs
type NodeEdge struct {
	ID     string `json:"id" boltholdKey:"ID"`
	Type   string `json:"type"`
	Parent string `json:"parent"`
	Points Points `json:"points"`
}

// ToNode converts to structure stored in db
func (n *NodeEdge) ToNode() Node {
	return Node{
		ID:     n.ID,
		Type:   n.Type,
		Points: n.Points,
	}
}

// ProcessPoint takes a point for a device and adds/updates its array of points
func (n *NodeEdge) ProcessPoint(pIn Point) {
	pFound := false
	for i, p := range n.Points {
		if p.ID == pIn.ID && p.Type == pIn.Type && p.Index == pIn.Index {
			pFound = true
			n.Points[i] = pIn
		}
	}

	if !pFound {
		n.Points = append(n.Points, pIn)
	}
}

// PbDecodeNode converts a protobuf to node data structure
func PbDecodeNode(data []byte) (Node, error) {
	pbNode := &pb.Node{}

	err := proto.Unmarshal(data, pbNode)
	if err != nil {
		return Node{}, err
	}

	points := make([]Point, len(pbNode.Points))

	for i, pPb := range pbNode.Points {
		s, err := PbToPoint(pPb)
		if err != nil {
			return Node{}, err
		}
		points[i] = s
	}

	ret := Node{
		ID:     pbNode.Id,
		Type:   pbNode.Type,
		Points: points,
	}

	return ret, nil
}

// ToPb encodes a node to a protobuf
func (n *Node) ToPb() ([]byte, error) {
	points := make([]*pb.Point, len(n.Points))

	for i, p := range n.Points {
		pPb, err := p.ToPb()
		if err != nil {
			return []byte{}, err
		}

		points[i] = &pPb
	}

	pbNode := pb.Node{
		Id:     n.ID,
		Type:   n.Type,
		Points: points,
	}

	return proto.Marshal(&pbNode)
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
