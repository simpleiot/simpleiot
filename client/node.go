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
			data.NewPointFloat(data.PointTypeTombstone, "", data.BoolToFloat(includeDel)))
	}

	if typ != "" {
		requestPoints = append(requestPoints,
			data.NewPointString(data.PointTypeNodeType, "", typ))
	}

	reqData := requestPoints.Encode()

	subject := fmt.Sprintf("nodes.%v.%v", parent, id)
	nodeMsg, err := nc.Request(subject, reqData, time.Second*20)
	if err != nil {
		return []data.NodeEdge{}, err
	}

	nodes, err := data.DecodeNodes(nodeMsg.Data)

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
			return none, fmt.Errorf("error getting root node: %v", err)
		}

		// The frontend expects the top level nodes to have Parent set
		// to root
		for i := range parents {
			parents[i].Parent = "root"
		}

		ret = append(ret, parents...)
		c, err := getChildren(un.Parent)
		if err != nil {
			return none, fmt.Errorf("error getting children: %v", err)
		}
		ret = append(ret, c...)
	}

	ret = data.RemoveDuplicateNodesIDParent(ret)

	return ret, nil
}

// shouldMarkPrimary reports whether SendNode should mark this edge primary.
//
// A node that owns something outside the tree runs its client on one edge
// only, so the edge it is created with is marked primary. SendNode is the one
// function every creation path reaches -- the add-node API, an import, and the
// clients that discover hardware -- which is why the mark is applied there.
//
// Two cases are left alone. A caller that supplies its own role keeps it, so
// MirrorNode can mark a mirror and an import can restore whatever the file
// carries. An edge that already exists keeps whatever it has, because SendNode
// is also the update path: an import updating a node that was mirrored into a
// group would otherwise mark that mirror primary and start a second client on
// it, which is the failure this whole mechanism exists to prevent. Leaving
// existing edges alone is also what makes an upgrade quiet -- edges from
// before this point stay unmarked rather than being guessed at.
func shouldMarkPrimary(nc *nats.Conn, node data.NodeEdge) (bool, error) {
	if !data.NodeTypeIsPrimary(node.Type) {
		return false, nil
	}

	if node.EdgeRole() != data.EdgeRoleNone {
		return false, nil
	}

	existing, err := GetNodes(nc, node.Parent, node.ID, "", true)
	if err != nil && err != data.ErrDocumentNotFound {
		return false, fmt.Errorf("error checking for existing edge: %w", err)
	}

	return len(existing) == 0, nil
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
		return errors.New("parent must be set when sending a node")
	}

	markPrimary, err := shouldMarkPrimary(nc, node)
	if err != nil {
		return err
	}

	err = SendNodePoints(nc, node.ID, points, true)

	if err != nil {
		return fmt.Errorf("error sending node: %v", err)
	}

	if len(node.EdgePoints) <= 0 {
		// edge should always have a tombstone point, set to false for root node
		node.EdgePoints = []data.Point{{Time: time.Now(),
			Type: data.PointTypeTombstone, Origin: origin}}
	}

	// a caller can supply the node type itself, as an import does when the
	// file it applies carries one, so only add it when it is missing
	hasNodeType := false

	for _, p := range node.EdgePoints {
		if p.Type == data.PointTypeNodeType {
			hasNodeType = true
			break
		}
	}

	if !hasNodeType {
		ntPt := data.NewPointString(data.PointTypeNodeType, "", node.Type)
		ntPt.Origin = origin
		node.EdgePoints = append(node.EdgePoints, ntPt)
	}

	if markPrimary {
		pPt := data.NewPointFloat(data.PointTypePrimary, "", 1)
		pPt.Origin = origin
		node.EdgePoints = append(node.EdgePoints, pPt)
	}

	err = SendEdgePoints(nc, node.ID, node.Parent, node.EdgePoints, true)
	if err != nil {
		return fmt.Errorf("error sending edge points: %w", err)

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

	// a duplicate is a new node rather than another view of the one it was
	// copied from, so it carries no role over. Duplicating a mirror would
	// otherwise produce a node whose only edge is a mirror, leaving it with
	// no primary anywhere and no client running it. SendNode marks the new
	// edge primary when the type calls for it.
	node.EdgePoints = clearEdgeRole(node.EdgePoints)

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
		return fmt.Errorf("no nodes returned")
	}

	node := nodes[0]

	switch node.Type {
	case data.NodeTypeUser:
		lastName, _ := node.Points.Text(data.PointTypeLastName, "0")
		lastName = lastName + " (Duplicate)"
		node.AddPoint(data.NewPointString(data.PointTypeLastName, "0", lastName))
	default:
		desc := node.Desc() + " (Duplicate)"
		node.AddPoint(data.NewPointString(data.PointTypeDescription, "0", desc))
	}

	return duplicateNodeHelper(nc, node, newParent, origin)
}

// clearEdgeRole removes any primary or mirror point from a set of edge points.
func clearEdgeRole(points data.Points) data.Points {
	var ret data.Points

	for _, p := range points {
		if p.Type == data.PointTypePrimary || p.Type == data.PointTypeMirror {
			continue
		}

		ret = append(ret, p)
	}

	return ret
}

// DeleteNode removes a node from the specified parent node.
//
// Deleting the primary edge of a node also deletes its mirrors. A mirror of a
// deleted sensor is an entry in someone's group with no client behind it and
// no points arriving, and nothing in the UI to say the original is gone.
// Deleting a mirror, or an edge with no role, removes only that edge.
func DeleteNode(nc *nats.Conn, id, parent string, origin string) error {
	tombstone := func() data.Point {
		p := data.NewPointFloat(data.PointTypeTombstone, "", 1)
		p.Origin = origin
		return p
	}

	for _, m := range mirrorsOf(nc, id, parent) {
		if err := SendEdgePoint(nc, id, m, tombstone(), true); err != nil {
			return fmt.Errorf("error deleting mirror under %v: %w", m, err)
		}
	}

	return SendEdgePoint(nc, id, parent, tombstone(), true)
}

// mirrorsOf returns the parents of the mirror edges that go with a primary
// edge, and nothing when the edge named is not the primary. A failure to read
// the tree returns nothing rather than an error, so that a delete the user
// asked for still happens; the cost is a mirror left behind, which they can
// remove themselves.
func mirrorsOf(nc *nats.Conn, id, parent string) []string {
	nodes, err := GetNodes(nc, "all", id, "", false)
	if err != nil {
		log.Println("Error looking up mirrors to delete:", err)
		return nil
	}

	primary := false

	for _, n := range nodes {
		if n.Parent == parent && n.EdgeRole() == data.EdgeRolePrimary {
			primary = true
			break
		}
	}

	if !primary {
		return nil
	}

	var ret []string

	for _, n := range nodes {
		if n.Parent != parent && n.EdgeRole() == data.EdgeRoleMirror {
			ret = append(ret, n.Parent)
		}
	}

	return ret
}

// roleFor returns the edge point that reproduces the role of the edge under
// oldParent, and whether there is one to write.
func roleFor(nodes []data.NodeEdge, oldParent string) (data.Point, bool) {
	for _, n := range nodes {
		if n.Parent != oldParent {
			continue
		}

		switch n.EdgeRole() {
		case data.EdgeRolePrimary:
			return data.NewPointFloat(data.PointTypePrimary, "", 1), true
		case data.EdgeRoleMirror:
			return data.NewPointFloat(data.PointTypeMirror, "", 1), true
		}
	}

	return data.Point{}, false
}

// checkMoveParent rejects a move that would take a node out from under the
// parent that owns it. A modbusIo is found by walking down from its modbus
// bus rather than from the tree root, so moving one into a group leaves it
// where nothing looks for it and it quietly stops working. Mirroring is the
// operation that was wanted in that case, and it stays available.
func checkMoveParent(nc *nats.Conn, typ, newParent string) error {
	owner := data.NodeTypeOwner(typ)
	if owner == "" {
		return nil
	}

	parents, err := GetNodes(nc, "all", newParent, "", false)
	if err != nil {
		return fmt.Errorf("error fetching destination node: %w", err)
	}

	if len(parents) < 1 {
		return errors.New("error fetching destination node to get type")
	}

	if parents[0].Type == owner {
		return nil
	}

	return fmt.Errorf("a %v node belongs under a %v node and cannot be moved under a %v node; mirror it instead",
		typ, owner, parents[0].Type)
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
		return errors.New("error fetching node to get type")
	}

	if err := checkMoveParent(nc, nodes[0].Type, newParent); err != nil {
		return err
	}

	points := data.Points{
		func() data.Point {
			p := data.NewPointFloat(data.PointTypeTombstone, "", 0)
			p.Origin = origin
			return p
		}(),
		func() data.Point {
			p := data.NewPointString(data.PointTypeNodeType, "", nodes[0].Type)
			p.Origin = origin
			return p
		}(),
	}

	// a move writes a new edge rather than editing the old one, so the role
	// has to be carried across. Without this a moved node would come out
	// unmarked, and a moved mirror would start running a client.
	if p, ok := roleFor(nodes, oldParent); ok {
		p.Origin = origin
		points = append(points, p)
	}

	err = SendEdgePoints(nc, id, newParent, points, true)

	if err != nil {
		return err
	}

	err = SendEdgePoint(nc, id, oldParent, data.NewPointFloat(data.PointTypeTombstone, "", 1), true)

	if err != nil {
		return err
	}

	return nil
}

// MirrorNode adds an existing node to a new parent. A node can have
// multiple parents.
//
// When the node being mirrored has a primary edge -- it owns a bus, a line, a
// socket -- the new edge is marked a mirror, so that it displays the node
// without running a second client on it. oldParent names the edge being
// mirrored from, which may itself be a mirror. Mirroring a node with no
// primary location, such as a user into a second group, marks nothing.
//
// A node of an owning type whose edges carry no role -- one created before
// edge roles existed -- has its source edge marked primary here, so that the
// mirror is a mirror of something.
func MirrorNode(nc *nats.Conn, id, oldParent, newParent, origin string) error {
	if newParent == oldParent {
		return errors.New("can't mirror node to the parent it is already under")
	}

	// fetch the node because we need to know its type
	nodes, err := GetNodes(nc, "all", id, "", true)
	if err != nil {
		return err
	}

	if len(nodes) < 1 {
		return errors.New("error fetching node to get type")
	}

	points := data.Points{
		func() data.Point {
			p := data.NewPointFloat(data.PointTypeTombstone, "", 0)
			p.Origin = origin
			return p
		}(),
		func() data.Point {
			p := data.NewPointString(data.PointTypeNodeType, "", nodes[0].Type)
			p.Origin = origin
			return p
		}(),
	}

	mirror, markSource := mirrorRoleFor(nodes, nodes[0].Type, oldParent)

	if mirror {
		p := data.NewPointFloat(data.PointTypeMirror, "", 1)
		p.Origin = origin
		points = append(points, p)
	}

	// the source edge is marked first, so that a failure part way through
	// leaves the node with a primary edge and no mirror rather than a
	// mirror and nothing that owns the hardware
	if markSource {
		p := data.NewPointFloat(data.PointTypePrimary, "", 1)
		p.Origin = origin

		if err := SendEdgePoint(nc, id, oldParent, p, true); err != nil {
			return fmt.Errorf("error marking source edge primary: %w", err)
		}
	}

	return SendEdgePoints(nc, id, newParent, points, true)
}

// mirrorRoleFor reports whether a new edge to this node should be marked a
// mirror, and whether the source edge has to be marked primary to go with it.
//
// The source edge decides the first, so that mirroring a mirror produces
// another mirror. When the source edge cannot be found -- the UI copied from a
// parent that has since gone -- any role on the node is enough to say the node
// has a primary location.
//
// A node whose edges carry no role at all is one created before edge roles
// existed. Nothing can be guessed about which of several existing edges was
// meant to be the primary, but a mirror being made now is a new edge, and the
// edge it is made from is the place the node already lived. So for a node type
// that owns something outside the tree, this marks that source edge primary
// and the new edge a mirror. Without it, mirroring a hardware node onto an
// upstream instance would start a second client there driving a line that
// exists on the device.
func mirrorRoleFor(nodes []data.NodeEdge, typ, oldParent string) (mirror, markSource bool) {
	for _, n := range nodes {
		if n.Parent == oldParent {
			if n.EdgeRole() != data.EdgeRoleNone {
				return true, false
			}

			break
		}
	}

	for _, n := range nodes {
		if n.EdgeRole() != data.EdgeRoleNone {
			return true, false
		}
	}

	if !data.NodeTypeIsPrimary(typ) {
		return false, false
	}

	// only mark the source when it is there to mark; a mirror made from an
	// edge that has since gone leaves the roles for the next one to set
	for _, n := range nodes {
		if n.Parent == oldParent {
			return true, true
		}
	}

	return false, false
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
		return nil, nil, fmt.Errorf("point subscribe failed: %v", err)
	}

	stopEdgeSub, err := SubscribeEdgePoints(nc, id, parent, func(points []data.Point) {
		edgeUpdates <- points
	})
	if err != nil {
		return nil, nil, fmt.Errorf("edge point subscribe failed: %v", err)
	}

	nodes, err := GetNodesType[T](nc, parent, id)
	if err != nil {
		if err != data.ErrDocumentNotFound {
			return nil, nil, fmt.Errorf("error getting node: %v", err)
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

// ExportNodes exports the nodes below a location as YAML. The node type is the
// key, and each point type is a key of its own:
//
//	apiVersion: 1
//	nodes:
//	  - group:
//	      description: Sensors
//	      children:
//	        - modbus:
//	            description: Modbus sensors
//	            port: /dev/ttyS1
//	            baud: 9600
//
// A text point is written quoted and a numeric one bare, so a value keeps its
// kind: `port: "502"` is text and `port: 502` is numeric.
//
// The file describes configuration and nothing else, which is what makes an
// export usable as a provisioning file. It carries no node IDs, since nodes are
// matched by description when a file is applied; a nodeID point is written as
// the description of the node it points at. Points that carry no value, points
// carrying raw bytes, tombstoned points, and point origins are all left out.
//
// Exporting the root node exports what is under it rather than the node
// itself: the root is the instance rather than configuration, and a file
// describing it would match nothing anywhere else.
func ExportNodes(nc *nats.Conn, id string) ([]byte, error) {
	root, err := GetRootNode(nc)
	if err != nil {
		return nil, fmt.Errorf("error getting root node: %w", err)
	}

	if id == "root" || id == "" {
		id = root.ID
	}

	var necs []data.NodeEdgeChildren

	if id == root.ID {
		children, err := GetNodes(nc, id, "all", "", false)
		if err != nil {
			return nil, fmt.Errorf("error getting nodes: %w", err)
		}

		for _, c := range children {
			nec := data.NodeEdgeChildren{NodeEdge: c, Children: nil}
			if err := exportNodesHelper(nc, &nec); err != nil {
				return nil, err
			}

			necs = append(necs, nec)
		}
	} else {
		nodes, err := GetNodes(nc, "all", id, "", false)
		if err != nil {
			return nil, fmt.Errorf("error getting nodes: %w", err)
		}

		if len(nodes) < 1 {
			return nil, fmt.Errorf("no nodes returned")
		}

		// we only export one node as there may be multiple mirrors of the node in the tree
		nec := data.NodeEdgeChildren{NodeEdge: nodes[0], Children: nil}
		if err := exportNodesHelper(nc, &nec); err != nil {
			return nil, err
		}

		necs = append(necs, nec)
	}

	if err := checkExportKeys(necs); err != nil {
		return nil, err
	}

	// a nodeID point holds the ID of the node it refers to, and a file names
	// that node by description instead, so we need every node in the tree
	// rather than only the ones being exported
	tree, err := getTree(nc, root.ID)
	if err != nil {
		return nil, err
	}

	descriptions := map[string]string{}
	for _, n := range tree {
		descriptions[n.ID] = n.Points.MatchKey()
	}

	nodes := make([]data.NodeYAML, len(necs))
	for i, nec := range necs {
		nodes[i] = exportNodeYAML(nec, descriptions)
	}

	f := data.NodeFile{
		APIVersion: data.NodeFileAPIVersion,
		Nodes:      nodes,
	}

	// indent sequences so that the nesting a person reads matches the nesting
	// of the tree
	return yaml.MarshalWithOptions(f, yaml.IndentSequence(true))
}

// checkExportKeys makes sure the tree can be described by a file. A file finds
// each node by description among its siblings, so two siblings sharing one is a
// tree no file can express. Rather than write a file that does the wrong thing
// when it is applied, say so here.
func checkExportKeys(nodes []data.NodeEdgeChildren) error {
	if err := checkSiblingKeys("the top of the tree", nodes); err != nil {
		return err
	}

	var check func(data.NodeEdgeChildren) error
	check = func(n data.NodeEdgeChildren) error {
		if err := checkSiblingKeys(describeNode(n.NodeEdge), n.Children); err != nil {
			return err
		}

		for _, c := range n.Children {
			if err := check(c); err != nil {
				return err
			}
		}

		return nil
	}

	for _, n := range nodes {
		if err := check(n); err != nil {
			return err
		}
	}

	return nil
}

// checkSiblingKeys reports nodes that an applied file could not tell apart.
func checkSiblingKeys(where string, nodes []data.NodeEdgeChildren) error {
	keys := map[string]bool{}
	types := map[string]bool{}

	for _, n := range nodes {
		key := n.Points.MatchKey()

		if key == "" {
			// no description: this node is found by its type instead, so there
			// can only be one of that type here
			if types[n.Type] {
				return fmt.Errorf("%v has more than one %v node with no description, so a file could "+
					"not tell them apart. Give them descriptions to export this tree", where, n.Type)
			}

			types[n.Type] = true

			continue
		}

		if keys[key] {
			return fmt.Errorf("%v has more than one node described %q, so a file could not tell them "+
				"apart. Give them unique descriptions to export this tree", where, key)
		}

		keys[key] = true
	}

	return nil
}

// describeNode names a node for an error message.
func describeNode(n data.NodeEdge) string {
	if key := n.Points.MatchKey(); key != "" {
		return key
	}

	return "the " + n.Type + " node"
}

// exportNodeYAML converts a fetched subtree into the file format.
func exportNodeYAML(nec data.NodeEdgeChildren, descriptions map[string]string) data.NodeYAML {
	out := data.NodeYAML{Type: nec.Type}

	for _, p := range nec.Points {
		if p.DataType == data.PointDataTypeUnknown && len(p.Data) == 0 {
			// a point with no value says nothing in a file
			continue
		}

		if p.Type == data.PointTypeNodeID {
			if desc, ok := descriptions[p.Txt()]; ok && desc != "" {
				p.PutString(desc)
			}
		}

		out.Points = append(out.Points, p)
	}

	out.EdgePoints = append(out.EdgePoints, nec.EdgePoints...)

	for _, c := range nec.Children {
		out.Children = append(out.Children, exportNodeYAML(c, descriptions))
	}

	return out
}

func exportNodesHelper(nc *nats.Conn, node *data.NodeEdgeChildren) error {
	// sort edge and node points
	sort.Sort(data.ByTypeKey(node.Points))
	sort.Sort(data.ByTypeKey(node.EdgePoints))
	// reduce a little noise ...
	// remove tombstone "0" edge points as that does not convey much information
	// also remove nodeType edge points, as the node type is the key a node is
	// written under, and remove key="0" fields in points
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

	// remove tombstone 0 and nodeType edge points
	i := 0
	for _, p := range node.EdgePoints {
		if p.Type == data.PointTypeTombstone && p.Val() == 0 {
			continue
		}

		if p.Type == data.PointTypeNodeType {
			continue
		}
		node.EdgePoints[i] = p
		i++
	}

	node.EdgePoints = node.EdgePoints[:i]

	children, err := GetNodes(nc, node.ID, "all", "", false)
	if err != nil {
		return fmt.Errorf("error getting children: %w", err)
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

// ImportNodes applies a node file to the tree. Nodes are matched by
// description, so importing a file creates what is missing, updates what has
// drifted, and does nothing when the tree already agrees. Nodes the file does
// not mention are left alone; a delete: list removes nodes.
//
// This is the same operation provisioning performs, so a file works either way.
func ImportNodes(nc *nats.Conn, yamlData []byte, origin string, dryRun bool) (ApplyPlan, error) {
	var f data.NodeFile

	if err := yaml.Unmarshal(yamlData, &f); err != nil {
		return ApplyPlan{}, fmt.Errorf("error parsing YAML data: %w", err)
	}

	if len(f.Nodes) < 1 && len(f.Delete) < 1 {
		return ApplyPlan{}, fmt.Errorf("error: imported data did not have any nodes")
	}

	return Apply(nc, f, ApplyOptions{Origin: origin, DryRun: dryRun})
}
