package client

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sort"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// GetNodes over NATS. Maps to the `p.<id>.<parent>` NATS API.
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
	nodeType := data.ToCamelCase(reflect.TypeOf(x).Name())

	nodes, err := GetNodes(nc, parent, id, nodeType, false)

	if err != nil {
		return []T{}, err
	}

	// decode from NodeEdge to custom types
	ret := make([]T, len(nodes))

	for i, n := range nodes {
		err := data.Decode(data.NodeEdgeChildren{NodeEdge: n, Children: nil}, &ret[i])
		if err != nil {
			log.Println("Error decode node in GetNodeType:", err)
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
		parents, err := GetNodes(nc, "all", un.Parent, "", false)
		if err != nil {
			return none, fmt.Errorf("Error getting root node: %v", err)
		}

		// The frontend expects the top level nodes to have Parent set
		// to root
		for i := range parents {
			parents[i].Parent = "root"
		}

		ret = append(ret, parents...)
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

	if origin != "" {
		for i := range node.Points {
			if node.Points[i].Origin == "" {
				node.Points[i].Origin = origin
			}
		}

		for i := range node.EdgePoints {
			if node.EdgePoints[i].Origin == "" {
				node.EdgePoints[i].Origin = origin
			}
		}
	}

	// we need to send the edge points first if we are creating
	// a new node, otherwise the upstream will detect an ophraned node
	// and create a new edge to the root node
	points := node.Points

	if node.ID == "" {
		return errors.New("ID must be set")
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
	nodes, err := GetNodes(nc, "all", id, "", false)
	if err != nil {
		return fmt.Errorf("GetNode error: %v", err)
	}

	if len(nodes) < 1 {
		return fmt.Errorf("No nodes returned")
	}

	node := nodes[0]

	switch node.Type {
	case data.NodeTypeUser:
		lastName, _ := node.Points.Text(data.PointTypeLastName, "0")
		lastName = lastName + " (Duplicate)"
		node.AddPoint(data.Point{Type: data.PointTypeLastName, Key: "0", Text: lastName})
	default:
		desc := node.Desc() + " (Duplicate)"
		node.AddPoint(data.Point{Type: data.PointTypeDescription, Key: "0", Text: desc})
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
				err := data.MergePoints(id, pts, &current)
				if err != nil {
					log.Println("NodeWatcher, error merging points:", err)
				}
			case pts := <-edgeUpdates:
				err := data.MergeEdgePoints(id, parent, pts, &current)
				if err != nil {
					log.Println("NodeWatcher, error merging edge points:", err)
				}
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

// SiotExport is the format used for exporting and importing data (currently YAML)
type SiotExport struct {
	Nodes []data.NodeEdgeChildren
}

// ExportNodes is used to export nodes at a particular location to YAML
// The YAML format looks like:
//
//	nodes:
//	- id: inst1
//	  type: device
//	  parent: root
//	  points:
//	  - type: versionApp
//	  children:
//	  - id: d7f5bbe9-a300-4197-93fa-b8e5e07f683a
//	    type: user
//	    parent: inst1
//	    points:
//	    - type: firstName
//	      text: admin
//	    - type: lastName
//	      text: user
//	    - type: phone
//	    - type: email
//	      text: admin@admin.com
//	    - type: pass
//	      text: admin
//
// Key="0" and Tombstone points with value set to 0 are removed from the export to make
// it easier to read.
func ExportNodes(nc *nats.Conn, id string) ([]byte, error) {
	if id == "root" || id == "" {
		root, err := GetRootNode(nc)
		if err != nil {
			return nil, fmt.Errorf("Error getting root node: %w", err)
		}
		id = root.ID
	}

	rootNodes, err := GetNodes(nc, "all", id, "", false)
	if err != nil {
		return nil, fmt.Errorf("Error getting root nodes: %w", err)
	}

	if len(rootNodes) < 1 {
		return nil, fmt.Errorf("no root nodes returned")
	}

	var necNodes []data.NodeEdgeChildren

	// we only export one node as there may be multiple mirrors of the node in the tree
	nec := data.NodeEdgeChildren{NodeEdge: rootNodes[0], Children: nil}
	err = exportNodesHelper(nc, &nec)
	if err != nil {
		return nil, err
	}

	necNodes = append(necNodes, nec)

	ne := SiotExport{Nodes: necNodes}

	return yaml.Marshal(ne)
}

func exportNodesHelper(nc *nats.Conn, node *data.NodeEdgeChildren) error {
	// sort edge and node points
	sort.Sort(data.ByTypeKey(node.Points))
	sort.Sort(data.ByTypeKey(node.EdgePoints))
	// reduce a little noise ...
	// remove tombstone "0" edge points as that does not convey much information
	// also remove and key="0" fields in points
	for i, p := range node.Points {
		if p.Key == "0" {
			node.Points[i].Key = ""
		}
	}

	for i, p := range node.EdgePoints {
		if p.Key == "0" {
			node.EdgePoints[i].Key = ""
		}
	}

	// remove tombstone 0 edge points
	i := 0
	for _, p := range node.EdgePoints {
		if p.Type == data.PointTypeTombstone && p.Value == 0 {
			continue
		}
		node.EdgePoints[i] = p
		i++
	}

	node.EdgePoints = node.EdgePoints[:i]

	children, err := GetNodes(nc, node.ID, "all", "", false)
	if err != nil {
		return fmt.Errorf("Error getting children: %w", err)
	}

	for _, c := range children {
		nec := data.NodeEdgeChildren{NodeEdge: c, Children: nil}
		err := exportNodesHelper(nc, &nec)
		if err != nil {
			return err
		}

		node.Children = append(node.Children, nec)
	}

	return nil
}

// ImportNodes is used to import nodes at a location in YAML format. New IDs
// are generated for all nodes unless preserve IDs is set to true.
// If there multiple references to the same ID,
// then an attempt is made to replace all of these with the new ID.  This also
// allows you to use "friendly" ID names in hand generated YAML files.
func ImportNodes(nc *nats.Conn, parent string, yamlData []byte, origin string, preserveIDs bool) error {
	// first make sure the parent node exists
	var rootNode data.NodeEdge
	if parent == "root" || parent == "" {
		var err error
		rootNode, err = GetRootNode(nc)
		if err != nil {
			return err
		}
	} else {
		n, err := GetNodes(nc, "all", parent, "", false)
		if err != nil {
			return err
		}
		if len(n) < 1 {
			return fmt.Errorf("Parent node \"%v\" not found", parent)
		}
	}

	var imp SiotExport

	err := yaml.Unmarshal(yamlData, &imp)
	if err != nil {
		return fmt.Errorf("Error parsing YAML data: %w", err)
	}

	var importHelper func(data.NodeEdgeChildren) error
	importHelper = func(node data.NodeEdgeChildren) error {
		err := SendNode(nc, node.NodeEdge, origin)
		if err != nil {
			return fmt.Errorf("Error sending node: %w", err)
		}

		for _, c := range node.Children {
			err := importHelper(c)
			if err != nil {
				return err
			}
		}
		return nil
	}

	if len(imp.Nodes) < 1 {
		return fmt.Errorf("Error: imported data did not have any nodes")
	}

	// set parent of first node
	imp.Nodes[0].Parent = parent

	// append (import) to top level node description
	for i, p := range imp.Nodes[0].Points {
		if p.Type == data.PointTypeDescription {
			imp.Nodes[0].Points[i].Text += " (import)"
		}
	}

	if preserveIDs {
		err := checkIDs(imp.Nodes[0], parent)
		if err != nil {
			return err
		}
	} else {
		ReplaceIDs(&imp.Nodes[0], parent)
	}

	err = importHelper(imp.Nodes[0])

	// if we imported the root node, then we have to tombstone the old root node
	if parent == "root" && rootNode.ID != imp.Nodes[0].ID {
		err := DeleteNode(nc, rootNode.ID, parent, "import")
		if err != nil {
			return fmt.Errorf("Error deleting old root node: %w", err)
		}
	}

	return err
}

func checkIDs(node data.NodeEdgeChildren, parent string) error {
	if parent == "" {
		return fmt.Errorf("parent must be specified")
	}

	if node.Parent != parent {
		return fmt.Errorf("node parent %v does not match parent %v", node.Parent, parent)
	}

	if node.ID == "" {
		return fmt.Errorf("ID cannot be blank")
	}

	for _, c := range node.Children {
		err := checkIDs(c, node.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReplaceIDs is used to replace IDs tree of nodes.
// If there multiple references to the same ID,
// then an attempt is made to replace all of these with the new ID.
// This function modifies the tree that is passed in.
// Replace IDs also updates the partent fields.
func ReplaceIDs(nodes *data.NodeEdgeChildren, parent string) {
	// idMap is used to translate old IDs to new
	idMap := make(map[string]string)

	var replaceHelper func(*data.NodeEdgeChildren, string)
	replaceHelper = func(n *data.NodeEdgeChildren, parent string) {
		n.Parent = parent
		// update node ID
		var newID string
		if n.ID == "" {
			// always assign a new ID if blank
			newID = uuid.New().String()
		} else {
			var ok bool
			newID, ok = idMap[n.ID]
			if !ok {
				newID = uuid.New().String()
				idMap[n.ID] = newID
			}
		}
		n.ID = newID

		// check for any points that might have node hashes
		for i, p := range n.Points {
			if p.Type == data.PointTypeNodeID {
				if p.Text == "" {
					continue
				}
				newID, ok := idMap[p.Text]
				if !ok {
					newID = uuid.New().String()
					idMap[p.Text] = newID
				}
				n.Points[i].Text = newID
			}
		}

		for i := range n.Children {
			replaceHelper(&n.Children[i], n.ID)
		}
	}

	replaceHelper(nodes, parent)
}
