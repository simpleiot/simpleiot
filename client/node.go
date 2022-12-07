package client

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// GetNodes over NATS. Maps to the `nodes.<parent>.<id>` NATS API.
// Returns data.ErrDocumentNotFound if node is not found.
// If parent is set to "none", the edge details are not included
// and the hash is blank.
// If parent is set to "all", then all living instances of the node are returned.
// If parent is set and id is "all", then all child nodes are returned.
// Parent can be set to "root" and id to "all" to fetch the root node(s).
func GetNodes(nc *nats.Conn, parent, id, typ string, includeDel bool) ([]data.NodeEdge, error) {
	if parent == "" {
		parent = "none"
	}

	if id == "" {
		id = "all"
	}

	var requestPoints data.Points

	if includeDel {
		requestPoints = append(requestPoints,
			data.Point{Type: data.PointTypeTombstone, Value: data.BoolToFloat(includeDel)})
	}

	if typ != "" {
		requestPoints = append(requestPoints,
			data.Point{Type: data.PointTypeNodeType, Text: typ})
	}

	reqData, err := requestPoints.ToPb()
	if err != nil {
		return nil, fmt.Errorf("Error encoding reqData: %v", err)
	}

	subject := fmt.Sprintf("nodes.%v.%v", parent, id)
	nodeMsg, err := nc.Request(subject, reqData, time.Second*20)
	if err != nil {
		return []data.NodeEdge{}, err
	}

	nodes, err := data.PbDecodeNodesRequest(nodeMsg.Data)

	if err != nil {
		return []data.NodeEdge{}, err
	}

	return nodes, nil
}

// GetNodesType gets node of a custom type.
// id and parent work the same as [GetNodes]
// Deleted nodes are not included.
func GetNodesType[T any](nc *nats.Conn, parent, id string) ([]T, error) {
	var x T
	nodeType := reflect.TypeOf(x).Name()
	nodeType = strings.ToLower(nodeType[0:1]) + nodeType[1:]

	nodes, err := GetNodes(nc, parent, id, nodeType, false)

	if err != nil {
		return []T{}, err
	}

	// decode from NodeEdge to custom types
	ret := make([]T, len(nodes))

	for i, n := range nodes {
		err := data.Decode(data.NodeEdgeChildren{NodeEdge: n, Children: nil}, &ret[i])
		if err != nil {
			log.Println("Error decode node in GetNodeType: ", err)
		}
	}

	return ret, nil
}

// GetRootNode returns the root node of the instance
func GetRootNode(nc *nats.Conn) (data.NodeEdge, error) {
	rootNodes, err := GetNodes(nc, "root", "all", "", false)

	if err != nil {
		return data.NodeEdge{}, err
	}

	if len(rootNodes) == 0 {
		return data.NodeEdge{}, data.ErrDocumentNotFound
	}

	return rootNodes[0], nil
}

// GetNodesForUser gets all nodes for a user
func GetNodesForUser(nc *nats.Conn, userID string) ([]data.NodeEdge, error) {
	var none []data.NodeEdge
	var ret []data.NodeEdge
	userNodes, err := GetNodes(nc, "all", userID, "", false)
	if err != nil {
		return none, err
	}

	var getChildren func(id string) ([]data.NodeEdge, error)

	// getNodesHelper recursively gets children of a node
	getChildren = func(id string) ([]data.NodeEdge, error) {
		var ret []data.NodeEdge

		children, err := GetNodes(nc, id, "all", "", false)
		if err != nil {
			return nil, err
		}

		for _, c := range children {
			grands, err := getChildren(c.ID)
			if err != nil {
				return nil, err
			}

			ret = append(ret, grands...)
		}

		ret = append(ret, children...)

		return ret, nil
	}

	// go through parents of root nodes and recursively get all children
	for _, un := range userNodes {
		n, err := GetNodes(nc, "all", un.Parent, "", false)
		if err != nil {
			return none, fmt.Errorf("Error getting root node: %v", err)
		}

		ret = append(ret, n...)
		c, err := getChildren(un.Parent)
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
func SendNode(nc *nats.Conn, node data.NodeEdge, origin string) error {
	// we need to send the edge points first if we are creating
	// a new node, otherwise the upstream will detect an ophraned node
	// and create a new edge to the root node
	points := node.Points

	if node.ID == "" {
		return errors.New("ID must be set to a UUID")
	}

	if node.Parent == "" || node.Parent == "none" {
		return errors.New("Parent must be set when sending a node")
	}

	err := SendNodePoints(nc, node.ID, points, true)

	if err != nil {
		return fmt.Errorf("Error sending node: %v", err)
	}

	if len(node.EdgePoints) <= 0 {
		// edge should always have a tombstone point, set to false for root node
		node.EdgePoints = []data.Point{{Time: time.Now(),
			Type: data.PointTypeTombstone, Origin: origin}}
	}

	node.EdgePoints = append(node.EdgePoints, data.Point{
		Type:   data.PointTypeNodeType,
		Text:   node.Type,
		Origin: origin,
	})

	err = SendEdgePoints(nc, node.ID, node.Parent, node.EdgePoints, true)
	if err != nil {
		return fmt.Errorf("Error sending edge points: %w", err)

	}

	return nil
}

// SendNodeType is used to send a node to a nats server. Can be
// used to create nodes.
func SendNodeType[T any](nc *nats.Conn, node T, origin string) error {
	ne, err := data.Encode(node)
	if err != nil {
		return err
	}

	if origin != "" {
		for i := range ne.Points {
			ne.Points[i].Origin = origin
		}

		for i := range ne.EdgePoints {
			ne.EdgePoints[i].Origin = origin
		}
	}

	return SendNode(nc, ne, origin)
}

func duplicateNodeHelper(nc *nats.Conn, node data.NodeEdge, newParent, origin string) error {
	children, err := GetNodes(nc, node.ID, "all", "", false)
	if err != nil {
		return fmt.Errorf("GetNodes error: %v", err)
	}

	// create new ID for duplicate node
	node.ID = uuid.New().String()
	node.Parent = newParent

	err = SendNode(nc, node, origin)
	if err != nil {
		return fmt.Errorf("SendNode error: %v", err)
	}

	for _, c := range children {
		err := duplicateNodeHelper(nc, c, node.ID, origin)
		if err != nil {
			return err
		}
	}

	return nil
}

// DuplicateNode is used to Duplicate a node and all its children
func DuplicateNode(nc *nats.Conn, id, newParent, origin string) error {
	nodes, err := GetNodes(nc, "none", id, "", false)
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

	return duplicateNodeHelper(nc, node, newParent, origin)
}

// DeleteNode removes a node from the specified parent node
func DeleteNode(nc *nats.Conn, id, parent string, origin string) error {
	err := SendEdgePoint(nc, id, parent, data.Point{
		Type:   data.PointTypeTombstone,
		Value:  1,
		Origin: origin,
	}, true)

	return err
}

// MoveNode moves a node from one parent to another
func MoveNode(nc *nats.Conn, id, oldParent, newParent, origin string) error {
	if newParent == oldParent {
		return errors.New("can't move node to itself")
	}

	// fetch the node because we need to know its type
	nodes, err := GetNodes(nc, "all", id, "", true)
	if err != nil {
		return err
	}

	if len(nodes) < 1 {
		return errors.New("Error fetching node to get type")
	}

	err = SendEdgePoints(nc, id, newParent, data.Points{
		{Type: data.PointTypeTombstone, Value: 0, Origin: origin},
		{Type: data.PointTypeNodeType, Text: nodes[0].Type, Origin: origin},
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
func MirrorNode(nc *nats.Conn, id, newParent, origin string) error {
	// fetch the node because we need to know its type
	nodes, err := GetNodes(nc, "all", id, "", true)
	if err != nil {
		return err
	}

	if len(nodes) < 1 {
		return errors.New("Error fetching node to get type")
	}

	err = SendEdgePoints(nc, id, newParent, data.Points{
		{Type: data.PointTypeTombstone, Value: 0, Origin: origin},
		{Type: data.PointTypeNodeType, Text: nodes[0].Type, Origin: origin},
	}, true)

	return err
}

// NodeWatcher creates a node watcher. update() is called any time there is an update.
// Stop can be called to stop the watcher. get() can be called to get the current value.
func NodeWatcher[T any](nc *nats.Conn, id, parent string) (get func() T, stop func(), err error) {
	stopCh := make(chan struct{})
	var current T

	pointUpdates := make(chan []data.Point)
	edgeUpdates := make(chan []data.Point)

	// create subscriptions first so that we get any updates that might happen between the
	// time we fetch node and start subscriptions

	stopPointSub, err := SubscribePoints(nc, id, func(points []data.Point) {
		pointUpdates <- points
	})
	if err != nil {
		return nil, nil, fmt.Errorf("Point subscribe failed: %v", err)
	}

	stopEdgeSub, err := SubscribeEdgePoints(nc, id, parent, func(points []data.Point) {
		edgeUpdates <- points
	})
	if err != nil {
		return nil, nil, fmt.Errorf("Edge point subscribe failed: %v", err)
	}

	nodes, err := GetNodesType[T](nc, parent, id)
	if err != nil {
		if err != data.ErrDocumentNotFound {
			return nil, nil, fmt.Errorf("Error getting node: %v", err)
		}
		// if document is not found, that is OK, points will populate it once they come in
	}

	// FIXME: we may still have a race condition where older point updates will overwrite
	// a new update when we fetch the node.
	if len(nodes) > 0 {
		current = nodes[0]
	}

	getCurrent := make(chan chan T)

	// main loop for watcher. All data access must go through the main
	// loop to avoid race conditions.
	go func() {
		for {
			select {
			case <-stopCh:
				return
			case r := <-getCurrent:
				r <- current
			case pts := <-pointUpdates:
				data.MergePoints(id, pts, &current)
			case pts := <-edgeUpdates:
				data.MergeEdgePoints(id, parent, pts, &current)
			}
		}
	}()

	return func() T {
			ret := make(chan T)
			getCurrent <- ret
			return <-ret
		}, func() {
			stopPointSub()
			stopEdgeSub()
			close(stopCh)
		}, nil
}
