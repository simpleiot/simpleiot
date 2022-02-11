package nats

import (
	"fmt"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/pb"
	"google.golang.org/protobuf/proto"
)

// GetNode over NATS. If id is "root", the root node is fetched.
// If parent is set to "none", the edge details are not included
// and the hash is calculated without the edge points.
// returns data.ErrDocumentNotFound if node is not found.
// if parent is set to "all", then all instances of the node are returned
func GetNode(nc *natsgo.Conn, id, parent string) ([]data.NodeEdge, error) {
	if parent == "" {
		parent = "none"
	}
	nodeMsg, err := nc.Request("node."+id, []byte(parent), time.Second*20)
	if err != nil {
		return []data.NodeEdge{}, err
	}

	node, err := data.PbDecodeNodesRequest(nodeMsg.Data)

	if err != nil {
		return []data.NodeEdge{}, err
	}

	return node, nil
}

// GetNodeChildren over NATS (immediate children only, not recursive)
// deleted nodes are skipped unless includeDel is set to true. typ
// can be used to limit nodes to a particular type, otherwise, all nodes
// are returned.
func GetNodeChildren(nc *natsgo.Conn, id, typ string, includeDel bool, recursive bool) ([]data.NodeEdge, error) {
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

	points := node.Points

	points = append(points, data.Point{
		Type: data.PointTypeNodeType,
		Text: node.Type,
	})

	err := SendNodePoints(nc, node.ID, points, true)

	if err != nil {
		return fmt.Errorf("Error sending node: %v", err)
	}

	return nil
}
