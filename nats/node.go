package nats

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// GetNode over NATS. If id is "root", the root node is fetched.
// If parent is set to "none", the edge details are not included
// and the hash is calculated without the edge points.
// returns data.ErrDocumentNotFound if node is not found.
// If parent is set to "all", then all living instances of the node are returned.
func GetNode(nc *natsgo.Conn, id, parent string) ([]data.NodeEdge, error) {
	if parent == "" {
		parent = "none"
	}
	nodeMsg, err := nc.Request("node."+id, []byte(parent), time.Second*20)
	if err != nil {
		return []data.NodeEdge{}, err
	}

	nodes, err := data.PbDecodeNodesRequest(nodeMsg.Data)

	if err != nil {
		return []data.NodeEdge{}, err
	}

	return nodes, nil
}

// GetNodeChildren over NATS (immediate children only, not recursive)
// deleted nodes are skipped unless includeDel is set to true. typ
// can be used to limit nodes to a particular type, otherwise, all nodes
// are returned.
func GetNodeChildren(nc *natsgo.Conn, id, typ string, includeDel bool, recursive bool) ([]data.NodeEdge, error) {
	fmt.Println("CLIFF: GetNodeChildren: ", id, typ)
	reqData, err := proto.Marshal(&pb.NatsRequest{IncludeDel: includeDel,
		Type: typ})

	if err != nil {
		return nil, err
	}

	nodeMsg, err := nc.Request("node."+id+".children", reqData, time.Second*20)
	if err != nil {
		return nil, err
	}

	nodes, err := data.PbDecodeNodesRequest(nodeMsg.Data)
	if err != nil {
		return nil, err
	}

	if recursive {
		recNodes := []data.NodeEdge{}
		for _, n := range nodes {
			c, err := GetNodeChildren(nc, n.ID, typ, includeDel, true)
			if err != nil {
				return nil, fmt.Errorf("GetNodeChildren, error getting children: %v", err)
			}
			recNodes = append(recNodes, c...)
		}

		nodes = append(nodes, recNodes...)
	}

	return nodes, nil
}

// GetNodesForUser gets all nodes for a user
func GetNodesForUser(nc *natsgo.Conn, userID string) ([]data.NodeEdge, error) {
	var none []data.NodeEdge
	var ret []data.NodeEdge
	rootNodes, err := GetNode(nc, userID, "all")
	if err != nil {
		return none, err
	}

	// go through parents of root nodes and recursively get all children
	for _, rn := range rootNodes {
		n, err := GetNode(nc, rn.Parent, "none")
		if err != nil {
			return none, fmt.Errorf("Error getting root node: %v", err)
		}
		ret = append(ret, n...)
		c, err := GetNodeChildren(nc, rn.Parent, "", false, true)
		if err != nil {
			return none, fmt.Errorf("Error getting children: %v", err)
		}

		ret = append(ret, c...)
	}

	ret = data.RemoveDuplicateNodesIDParent(ret)

	return ret, nil
}

// SendNode is used to send a node to a nats server. Can be
// used to create nodes.
func SendNode(nc *natsgo.Conn, node data.NodeEdge) error {
	// we need to send the edge points first if we are creating
	// a new node, otherwise the upstream will detect an ophraned node
	// and create a new edge to the root node

	points := node.Points

	if node.ID == "" {
		node.ID = uuid.New().String()
	}

	points = append(points, data.Point{
		Type: data.PointTypeNodeType,
		Text: node.Type,
	})

	err := SendNodePoints(nc, node.ID, points, true)

	if err != nil {
		return fmt.Errorf("Error sending node: %v", err)
	}

	if node.Parent != "" && node.Parent != "none" {
		if len(node.EdgePoints) < 0 {
			// edge should always have a tombstone point, set to false for root node
			node.EdgePoints = []data.Point{{Time: time.Now(), Type: data.PointTypeTombstone}}
		}

		err := SendEdgePoints(nc, node.ID, node.Parent, node.EdgePoints, true)
		if err != nil {
			return fmt.Errorf("Error sending edge points: %w", err)

		}
	}

	return nil
}

func duplicateNodeHelper(nc *natsgo.Conn, node data.NodeEdge, newParent string) error {
	children, err := GetNodeChildren(nc, node.ID, "", false, false)
	if err != nil {
		return fmt.Errorf("GetNodeChildren error: %v", err)
	}

	// create new ID for duplicate node
	node.ID = uuid.New().String()
	node.Parent = newParent

	err = SendNode(nc, node)
	if err != nil {
		return fmt.Errorf("SendNode error: %v", err)
	}

	for _, c := range children {
		err := duplicateNodeHelper(nc, c, node.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// DuplicateNode is used to Duplicate a node and all its children
func DuplicateNode(nc *natsgo.Conn, id, newParent string) error {
	nodes, err := GetNode(nc, id, "none")
	if err != nil {
		return fmt.Errorf("GetNode error: %v", err)
	}

	if len(nodes) < 1 {
		return fmt.Errorf("No nodes returned")
	}

	node := nodes[0]

	switch node.Type {
	case data.NodeTypeUser:
		lastName, _ := node.Points.Text(data.PointTypeLastName, "")
		lastName = lastName + " (Duplicate)"
		node.AddPoint(data.Point{Type: data.PointTypeLastName, Text: lastName})
	default:
		desc := node.Desc() + " (Duplicate)"
		node.AddPoint(data.Point{Type: data.PointTypeDescription, Text: desc})
	}

	return duplicateNodeHelper(nc, node, newParent)
}

// DeleteNode removes a node from the specified parent node
func DeleteNode(nc *natsgo.Conn, id, parent string) error {
	err := SendEdgePoint(nc, id, parent, data.Point{
		Type:  data.PointTypeTombstone,
		Value: 1,
	}, true)

	return err
}

// MoveNode moves a node from one parent to another
func MoveNode(nc *natsgo.Conn, id, oldParent, newParent string) error {
	if newParent == oldParent {
		return errors.New("can't move node to itself")
	}

	err := SendEdgePoint(nc, id, newParent, data.Point{
		Type:  data.PointTypeTombstone,
		Value: 0,
	}, true)

	if err != nil {
		return err
	}

	err = SendEdgePoint(nc, id, oldParent, data.Point{
		Type:  data.PointTypeTombstone,
		Value: 1,
	}, true)

	if err != nil {
		return err
	}

	return nil
}

// MirrorNode adds a an existing node to a new parent. A node can have
// multiple parents.
func MirrorNode(nc *natsgo.Conn, id, newParent string) error {
	err := SendEdgePoint(nc, id, newParent, data.Point{
		Type:  data.PointTypeTombstone,
		Value: 0,
	}, true)

	return err
}

// UserCheck sends a nats message to check auth of user
// This function returns user nodes and a JWT node which includes a token
func UserCheck(nc *natsgo.Conn, email, pass string) ([]data.NodeEdge, error) {
	points := data.Points{
		{Type: data.PointTypeEmail, Text: email},
		{Type: data.PointTypePass, Text: pass},
	}

	pointsData, err := points.ToPb()
	if err != nil {
		return []data.NodeEdge{}, err
	}

	nodeMsg, err := nc.Request("auth.user", pointsData, time.Second*20)
	if err != nil {
		return []data.NodeEdge{}, err
	}

	nodes, err := data.PbDecodeNodesRequest(nodeMsg.Data)

	if err != nil {
		return []data.NodeEdge{}, err
	}

	return nodes, nil
}
