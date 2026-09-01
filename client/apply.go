package client

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// ApplyOptions controls how a node file is applied.
type ApplyOptions struct {
	// Origin is recorded on every point applied, and is "import" for
	// siot import and "provision:<name>" for a provisioning pass.
	Origin string
	// DryRun plans the work without sending anything.
	DryRun bool
}

// ApplySend is one node the file wants created or updated. For an update,
// Node carries only the points that differ from what the tree already holds.
type ApplySend struct {
	Node    data.NodeEdge
	Key     string
	Created bool
}

// ApplyDelete is one node a delete: entry matched.
type ApplyDelete struct {
	ID     string
	Parent string
	Key    string
	Type   string
}

// ApplyPlan is what applying a file will do. A file that has already been
// applied plans nothing, which is what makes applying it repeatedly safe.
type ApplyPlan struct {
	Send   []ApplySend
	Delete []ApplyDelete
	// DeviceKey is the public key of the device key the file installs, when
	// it carries one the instance does not already have.
	DeviceKey string
	// Errors holds per entry failures. An entry that fails is skipped along
	// with its children; the rest of the file still applies.
	Errors []error
}

// Empty reports whether the plan has nothing to do.
func (p ApplyPlan) Empty() bool {
	return len(p.Send) == 0 && len(p.Delete) == 0 && p.DeviceKey == ""
}

func (p ApplyPlan) String() string {
	if p.Empty() && len(p.Errors) == 0 {
		return "nothing to do\n"
	}

	var b strings.Builder

	for _, s := range p.Send {
		what := "update"
		if s.Created {
			what = "create"
		}

		key := s.Key
		if key == "" {
			key = s.Node.ID
		}

		fmt.Fprintf(&b, "%v %v %v (%v point(s))\n", what, s.Node.Type, key, len(s.Node.Points))
	}

	for _, d := range p.Delete {
		fmt.Fprintf(&b, "delete %v %v\n", d.Type, d.Key)
	}

	if p.DeviceKey != "" {
		fmt.Fprintf(&b, "install device key %v\n", p.DeviceKey)
	}

	for _, err := range p.Errors {
		fmt.Fprintf(&b, "error: %v\n", err)
	}

	return b.String()
}

// Apply makes the tree agree with a node file: nodes it describes are created
// or updated, nodes its delete: list names are removed, and nodes it does not
// mention are left alone. Applying the same file twice does what applying it
// once did, since nodes are matched by description rather than by ID.
func Apply(nc *nats.Conn, f data.NodeFile, o ApplyOptions) (ApplyPlan, error) {
	if f.APIVersion > data.NodeFileAPIVersion {
		return ApplyPlan{}, fmt.Errorf("this file is apiVersion %v, and this version of SIOT understands up to %v",
			f.APIVersion, data.NodeFileAPIVersion)
	}

	root, err := GetRootNode(nc)
	if err != nil {
		return ApplyPlan{}, fmt.Errorf("error getting root node: %w", err)
	}

	tree, err := getTree(nc, root.ID)
	if err != nil {
		return ApplyPlan{}, err
	}

	plan := planApply(f, tree, root.ID)

	if f.DeviceKey != "" {
		pubKey, err := ParseDeviceKey(f.DeviceKey)
		if err != nil {
			return plan, fmt.Errorf("error in deviceKey: %w", err)
		}
		_, current, err := GetDeviceKey(nc)
		if err != nil {
			return plan, fmt.Errorf("error getting device key: %w", err)
		}
		if current != pubKey {
			plan.DeviceKey = pubKey
		}
	}

	if o.DryRun {
		return plan, nil
	}

	if plan.DeviceKey != "" {
		if _, err := InstallDeviceKey(nc, f.DeviceKey); err != nil {
			return plan, fmt.Errorf("error installing device key: %w", err)
		}
	}

	for _, s := range plan.Send {
		if err := SendNode(nc, s.Node, o.Origin); err != nil {
			return plan, fmt.Errorf("error sending node %v: %w", s.Node.ID, err)
		}
	}

	for _, d := range plan.Delete {
		if err := DeleteNode(nc, d.ID, d.Parent, o.Origin); err != nil {
			return plan, fmt.Errorf("error deleting node %v: %w", d.ID, err)
		}
	}

	return plan, nil
}

// getTree fetches the subtree below a node, flattened, so that matching can
// work against a snapshot rather than a query per entry.
func getTree(nc *nats.Conn, rootID string) ([]data.NodeEdge, error) {
	root, err := GetNodes(nc, "all", rootID, "", false)
	if err != nil {
		return nil, fmt.Errorf("error getting root node: %w", err)
	}

	if len(root) < 1 {
		return nil, fmt.Errorf("node %v not found", rootID)
	}

	out := []data.NodeEdge{root[0]}
	seen := map[string]bool{}

	var walk func(id string) error
	walk = func(id string) error {
		if seen[id] {
			// a node can be mirrored under more than one parent
			return nil
		}
		seen[id] = true

		children, err := GetNodes(nc, id, "all", "", false)
		if err != nil {
			return fmt.Errorf("error getting children of %v: %w", id, err)
		}

		for _, c := range children {
			out = append(out, c)

			if err := walk(c.ID); err != nil {
				return err
			}
		}

		return nil
	}

	if err := walk(rootID); err != nil {
		return nil, err
	}

	return out, nil
}

// working is the tree a plan is built against: the snapshot plus the nodes the
// plan is going to create, so that an entry can attach to a node an earlier
// entry in the same file described.
type working struct {
	nodes   []data.NodeEdge
	claimed map[string]bool
}

func (w *working) children(parent string) []data.NodeEdge {
	var out []data.NodeEdge

	for _, n := range w.nodes {
		if n.Parent == parent {
			out = append(out, n)
		}
	}

	return out
}

func (w *working) hasID(id string) bool {
	for _, n := range w.nodes {
		if n.ID == id {
			return true
		}
	}

	return false
}

func (w *working) add(n data.NodeEdge) {
	w.nodes = append(w.nodes, n)
}

// findByKey resolves a parent: reference, which names a node anywhere in the
// tree by its match key.
func (w *working) findByKey(key string) (string, error) {
	var found []data.NodeEdge

	for _, n := range w.nodes {
		if n.Points.MatchKey() == key {
			found = append(found, n)
		}
	}

	switch len(found) {
	case 0:
		return "", fmt.Errorf("no node found with description %q", key)
	case 1:
		return found[0].ID, nil
	default:
		return "", fmt.Errorf("%v nodes have the description %q, so it does not say which one to use", len(found), key)
	}
}

// match finds the node an entry describes among the children of a parent. An
// entry with a match key matches on that; one without matches on node type,
// which is how the singletons -- one metrics node, one serial node -- are
// addressed.
func (w *working) match(entry data.NodeYAML, parent string) (*data.NodeEdge, error) {
	key := entry.Points.MatchKey()

	var found []data.NodeEdge

	for _, n := range w.children(parent) {
		if key != "" {
			if n.Points.MatchKey() == key {
				found = append(found, n)
			}
			continue
		}

		if n.Type == entry.Type {
			found = append(found, n)
		}
	}

	switch len(found) {
	case 0:
		return nil, nil
	case 1:
		n := found[0]

		if n.Type != entry.Type {
			return nil, fmt.Errorf("%q is a %v node here and a %v node in the file. "+
				"Delete it first if the type is meant to change", key, n.Type, entry.Type)
		}

		if w.claimed[n.ID] {
			return nil, fmt.Errorf("two entries in this file describe %q", describe(entry))
		}

		w.claimed[n.ID] = true

		return &n, nil
	default:
		return nil, fmt.Errorf("%v nodes here match %q, so it does not say which one to update",
			len(found), describe(entry))
	}
}

// describe names an entry for an error message.
func describe(entry data.NodeYAML) string {
	if key := entry.Points.MatchKey(); key != "" {
		return key
	}

	return "the " + entry.Type + " node"
}

// resolvedEntry is an entry with its place in the tree worked out.
type resolvedEntry struct {
	entry    data.NodeYAML
	id       string
	parent   string
	existing *data.NodeEdge
}

// planApply works out what a file will do to a tree. It is pure, so the
// interesting half of applying a file is testable without a server.
func planApply(f data.NodeFile, tree []data.NodeEdge, root string) ApplyPlan {
	w := &working{nodes: append([]data.NodeEdge{}, tree...), claimed: map[string]bool{}}

	plan := ApplyPlan{}

	var resolved []resolvedEntry

	var walk func(entries []data.NodeYAML, parent string, top bool)
	walk = func(entries []data.NodeYAML, parent string, top bool) {
		for _, e := range entries {
			if e.Type == "" {
				plan.Errors = append(plan.Errors, fmt.Errorf("a node needs a type"))
				continue
			}

			id := parent

			if e.Parent != "" {
				if !top {
					plan.Errors = append(plan.Errors, fmt.Errorf("%v: parent: only applies to a top level entry, "+
						"since a child attaches to the node it is nested under", describe(e)))
					continue
				}

				found, err := w.findByKey(e.Parent)
				if err != nil {
					plan.Errors = append(plan.Errors, fmt.Errorf("%v: %w", describe(e), err))
					continue
				}

				id = found
			}

			nodeParent := id

			existing, err := w.match(e, nodeParent)
			if err != nil {
				plan.Errors = append(plan.Errors, err)
				continue
			}

			var nodeID string

			if existing != nil {
				nodeID = existing.ID
			} else {
				nodeID = uuid.New().String()

				// so that a later entry can attach to what this one creates,
				// and so that a reference can name it
				w.add(data.NodeEdge{
					ID:     nodeID,
					Type:   e.Type,
					Parent: nodeParent,
					Points: e.Points,
				})
				w.claimed[nodeID] = true
			}

			resolved = append(resolved, resolvedEntry{entry: e, id: nodeID, parent: nodeParent, existing: existing})

			walk(e.Children, nodeID, false)
		}
	}

	walk(f.Nodes, root, true)

	for _, r := range resolved {
		points, err := resolveRefs(r.entry.Points, w)
		if err != nil {
			plan.Errors = append(plan.Errors, fmt.Errorf("%v: %w", describe(r.entry), err))
			continue
		}

		edgePoints, err := resolveRefs(r.entry.EdgePoints, w)
		if err != nil {
			plan.Errors = append(plan.Errors, fmt.Errorf("%v: %w", describe(r.entry), err))
			continue
		}

		created := r.existing == nil

		if !created {
			points = changedPoints(points, r.existing.Points)
			edgePoints = changedPoints(edgePoints, r.existing.EdgePoints)

			if len(points) == 0 && len(edgePoints) == 0 {
				continue
			}
		}

		plan.Send = append(plan.Send, ApplySend{
			Node: data.NodeEdge{
				ID:         r.id,
				Type:       r.entry.Type,
				Parent:     r.parent,
				Points:     points,
				EdgePoints: edgePoints,
			},
			Key:     r.entry.Points.MatchKey(),
			Created: created,
		})
	}

	for _, e := range f.Delete {
		parent := root

		if e.Parent != "" {
			found, err := w.findByKey(e.Parent)
			if err != nil {
				plan.Errors = append(plan.Errors, fmt.Errorf("delete %v: %w", describe(e), err))
				continue
			}

			parent = found
		}

		existing, err := w.match(e, parent)
		if err != nil {
			plan.Errors = append(plan.Errors, fmt.Errorf("delete: %w", err))
			continue
		}

		if existing == nil {
			// already gone, which is what the file asked for
			continue
		}

		plan.Delete = append(plan.Delete, ApplyDelete{
			ID:     existing.ID,
			Parent: existing.Parent,
			Key:    describe(e),
			Type:   existing.Type,
		})
	}

	return plan
}

// resolveRefs rewrites nodeID points, which name the node they point at by its
// description, into that node's ID. Resolution happens once the whole file has
// been walked, so a reference may name a node the file creates further down, or
// one that another file created.
func resolveRefs(points data.Points, w *working) (data.Points, error) {
	if len(points) == 0 {
		return nil, nil
	}

	out := make(data.Points, len(points))
	copy(out, points)

	for i, p := range out {
		if p.Type != data.PointTypeNodeID {
			continue
		}

		key := p.Txt()

		if key == "" || w.hasID(key) {
			// nothing to resolve, or it already names a node in this tree
			continue
		}

		id, err := w.findByKey(key)
		if err != nil {
			return nil, fmt.Errorf("nodeID: %w", err)
		}

		out[i].PutString(id)
	}

	return out, nil
}

// changedPoints returns the points that differ from what a node already holds.
// Values are compared by what they mean rather than by their stored bytes, so
// that an integer and a float of the same value are not a change and a file
// does not fight a client that writes its points with PutInt.
//
// A point with no value at all is ignored here: it says nothing to assert, and
// treating it as "clear this" would wipe a value the file never mentioned.
func changedPoints(points, existing data.Points) data.Points {
	var out data.Points

	for _, p := range points {
		if empty(p) {
			continue
		}

		e, ok := existing.Find(p.Type, p.Key)
		if ok && data.SameValue(e, p) {
			continue
		}

		out = append(out, p)
	}

	return out
}

// empty reports whether a point carries no value.
func empty(p data.Point) bool {
	return p.DataType == data.PointDataTypeUnknown && len(p.Data) == 0
}
