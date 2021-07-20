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
// If parent is set to "skip", the edge details are not included
// and the hash is calculated without the edge points.
// returns data.ErrDocumentNotFound if node is not found.
func GetNode(nc *natsgo.Conn, id, parent string) (data.NodeEdge, error) {
	if parent == "" {
		parent = "none"
	}
	nodeMsg, err := nc.Request("node."+id, []byte(parent), time.Second*20)
	if err != nil {
		return data.NodeEdge{}, err
	}

	node, err := data.PbDecodeNodeRequest(nodeMsg.Data)

	if err != nil {
		return data.NodeEdge{}, err
	}

	return node, nil
}

// GetNodeChildren over NATS (immediate children only, not recursive)
// deleted nodes are skipped unless includeDel is set to true. typ
// can be used to limit nodes to a particular type, otherwise, all nodes
// are returned.
func GetNodeChildren(nc *natsgo.Conn, id, typ string, includeDel bool) ([]data.NodeEdge, error) {
	reqData, err := proto.Marshal(&pb.NatsRequest{IncludeDel: includeDel,
		Type: typ})

	if err != nil {
		return nil, err
	}

	nodeMsg, err := nc.Request("node."+id+".children", reqData, time.Second*20)
	if err != nil {
		return nil, err
	}

	nodes, err := data.PbDecodeNodes(nodeMsg.Data)

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// SendNode is used to recursively send a node and children over nats
func SendNode(src, dest *natsgo.Conn, node data.NodeEdge) error {
	points := node.Points

	points = append(points, data.Point{
		Type: data.PointTypeNodeType,
		Text: node.Type,
	})

	err := SendNodePoints(dest, node.ID, points, true)

	if err != nil {
		return fmt.Errorf("Error sending node upstream: %v", err)
	}

	if len(node.EdgePoints) < 0 {
		// edge should always have a tombstone point, set to false for root node
		node.EdgePoints = []data.Point{{Time: time.Now(), Type: data.PointTypeTombstone}}
	}

	err = SendEdgePoints(dest, node.ID, node.Parent, node.EdgePoints, true)
	if err != nil {
		return fmt.Errorf("Error sending edge points: %w", err)
	}

	// process child nodes
	childNodes, err := GetNodeChildren(src, node.ID, "", false)
	if err != nil {
		return fmt.Errorf("Error getting node children: %v", err)
	}

	for _, childNode := range childNodes {
		err := SendNode(src, dest, childNode)

		if err != nil {
			return fmt.Errorf("Error sending child node: %v", err)
		}
	}

	return nil
}
