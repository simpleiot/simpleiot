package data

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	yaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// Keys that have a meaning of their own inside a node body and are therefore
// not available as point types in the short form. A point of one of these
// types is written in the long form instead.
const (
	nodeKeyID         = "id"
	nodeKeyParent     = "parent"
	nodeKeyChildren   = "children"
	nodeKeyPoints     = "points"
	nodeKeyEdgePoints = "edgePoints"
)

func nodeKeyReserved(k string) bool {
	switch k {
	case nodeKeyID, nodeKeyParent, nodeKeyChildren, nodeKeyPoints, nodeKeyEdgePoints:
		return true
	}
	return false
}

// NodeFile is a file of nodes: what siot export writes, and what siot import
// and provisioning read.
type NodeFile struct {
	APIVersion int        `yaml:"apiVersion,omitempty"`
	Nodes      []NodeYAML `yaml:"nodes,omitempty"`
	Delete     []NodeYAML `yaml:"delete,omitempty"`
}

// NodeFileAPIVersion is the format version this build writes and understands.
const NodeFileAPIVersion = 1

// NodeYAML is one node in a node file. The YAML spelling puts the node type in
// the key position and each point type in a key of its own:
//
//	nodes:
//	  - modbus:
//	      description: Modbus sensors
//	      port: /dev/ttyS1
//	      baud: 9600
//
// A point whose value cannot be spelled that way without losing something is
// written in the points: list instead.
type NodeYAML struct {
	// ID is the id: field. In a file written by hand it is a label other
	// entries in the same file can refer to; in an exported file it is a UUID.
	ID string
	// Type is the node type, which is the key the rest of the body hangs from.
	Type string
	// Parent is the match key of the node this one attaches to, and is only
	// meaningful on a top level entry.
	Parent     string
	Points     Points
	EdgePoints Points
	Children   []NodeYAML
}

// pointYAML is the long form of a point, used for points the short form cannot
// express: those carrying data, an origin, a tombstone, an integer value, or a
// type that collides with a reserved key.
type pointYAML struct {
	Type      string        `yaml:"type,omitempty"`
	Key       string        `yaml:"key,omitempty"`
	Value     float64       `yaml:"value,omitempty"`
	Text      string        `yaml:"text,omitempty"`
	DataType  PointDataType `yaml:"dataType,omitempty"`
	Data      []byte        `yaml:"data,omitempty"`
	Tombstone int           `yaml:"tombstone,omitempty"`
	Origin    string        `yaml:"origin,omitempty"`
}

func pointToLong(p Point) pointYAML {
	py := pointYAML{
		Type:      p.Type,
		Key:       pointKey(p.Key),
		Tombstone: p.Tombstone,
		Origin:    p.Origin,
	}

	switch p.DataType {
	case PointDataTypeString:
		py.Text = p.Txt()
	case PointDataTypeFloat:
		py.Value = p.Val()
	case PointDataTypeInt:
		py.Value = p.Val()
		py.DataType = PointDataTypeInt
	default:
		if len(p.Data) > 0 {
			py.DataType = p.DataType
			py.Data = p.Data
		}
	}

	if py.Key == "0" {
		py.Key = ""
	}

	return py
}

func pointFromLong(py pointYAML) Point {
	p := Point{
		Type:      py.Type,
		Key:       pointKey(py.Key),
		Tombstone: py.Tombstone,
		Origin:    py.Origin,
	}

	switch {
	case py.DataType == PointDataTypeString || py.Text != "":
		p.PutString(py.Text)
	case py.DataType == PointDataTypeInt:
		p.PutInt(int64(py.Value))
	case len(py.Data) > 0:
		p.DataType = py.DataType
		p.Data = py.Data
	case py.Value != 0:
		p.PutFloat(py.Value)
	}

	return p
}

// pointKey normalizes a point key, since an empty key and "0" mean the same
// thing everywhere else in the system.
func pointKey(k string) string {
	if k == "" {
		return "0"
	}
	return k
}

// shortable reports whether a point can be written in the short form without
// losing anything.
func shortable(p Point) bool {
	if p.Tombstone != 0 || p.Origin != "" || nodeKeyReserved(p.Type) {
		return false
	}

	switch p.DataType {
	case PointDataTypeString, PointDataTypeFloat:
		return true
	case PointDataTypeUnknown:
		return len(p.Data) == 0
	default:
		// integers keep their type through the long form, and JSON and any
		// future type carries bytes the short form has no spelling for
		return false
	}
}

// shortValue returns the YAML value for a point in the short form.
func shortValue(p Point) any {
	switch p.DataType {
	case PointDataTypeString:
		return p.Txt()
	case PointDataTypeFloat:
		v := p.Val()
		// an integral float reads better without the trailing .0, and comes
		// back as a float either way
		if v == math.Trunc(v) && math.Abs(v) < 1e15 {
			return int64(v)
		}
		return v
	default:
		return nil
	}
}

// MarshalYAML implements the goccy/go-yaml InterfaceMarshaler.
func (n NodeYAML) MarshalYAML() (any, error) {
	body := yaml.MapSlice{}

	if n.ID != "" {
		body = append(body, yaml.MapItem{Key: nodeKeyID, Value: n.ID})
	}

	if n.Parent != "" {
		body = append(body, yaml.MapItem{Key: nodeKeyParent, Value: n.Parent})
	}

	short, long := splitPoints(n.Points)
	body = append(body, short...)

	if len(long) > 0 {
		pts := make([]pointYAML, len(long))
		for i, p := range long {
			pts[i] = pointToLong(p)
		}
		body = append(body, yaml.MapItem{Key: nodeKeyPoints, Value: pts})
	}

	if len(n.EdgePoints) > 0 {
		edge := make(Points, len(n.EdgePoints))
		copy(edge, n.EdgePoints)
		sort.Sort(ByTypeKey(edge))

		pts := make([]pointYAML, len(edge))
		for i, p := range edge {
			pts[i] = pointToLong(p)
		}
		body = append(body, yaml.MapItem{Key: nodeKeyEdgePoints, Value: pts})
	}

	if len(n.Children) > 0 {
		body = append(body, yaml.MapItem{Key: nodeKeyChildren, Value: n.Children})
	}

	return yaml.MapSlice{{Key: n.Type, Value: body}}, nil
}

// splitPoints divides points into those written in the short form, already
// ordered by type, and those left for the points: list. Points of one type
// travel together: if one of them needs the long form, all of them use it, so
// that a type never appears in both places.
func splitPoints(points Points) (yaml.MapSlice, Points) {
	groups := map[string]Points{}
	types := []string{}

	for _, p := range points {
		if _, ok := groups[p.Type]; !ok {
			types = append(types, p.Type)
		}
		groups[p.Type] = append(groups[p.Type], p)
	}

	sort.Strings(types)

	short := yaml.MapSlice{}
	var long Points

	for _, typ := range types {
		group := groups[typ]

		useShort := true
		for _, p := range group {
			if !shortable(p) {
				useShort = false
				break
			}
		}

		if !useShort {
			sorted := make(Points, len(group))
			copy(sorted, group)
			sort.Sort(ByTypeKey(sorted))
			long = append(long, sorted...)
			continue
		}

		short = append(short, yaml.MapItem{Key: typ, Value: groupValue(group)})
	}

	return short, long
}

// groupValue renders the points of one type: a scalar when there is a single
// keyless point, a sequence when the keys are 0..n-1, and a mapping otherwise.
func groupValue(group Points) any {
	if len(group) == 1 && pointKey(group[0].Key) == "0" {
		return shortValue(group[0])
	}

	byKey := map[string]Point{}
	keys := make([]string, 0, len(group))

	for _, p := range group {
		k := pointKey(p.Key)
		if _, ok := byKey[k]; !ok {
			keys = append(keys, k)
		}
		byKey[k] = p
	}

	sort.Strings(keys)

	if isIndexRun(keys) {
		seq := make([]any, len(keys))
		for i := range keys {
			seq[i] = shortValue(byKey[strconv.Itoa(i)])
		}
		return seq
	}

	ms := yaml.MapSlice{}
	for _, k := range keys {
		ms = append(ms, yaml.MapItem{Key: k, Value: shortValue(byKey[k])})
	}

	return ms
}

// isIndexRun reports whether keys are exactly "0".."n-1", which is how an
// array of points is spelled.
func isIndexRun(keys []string) bool {
	if len(keys) < 2 {
		return false
	}

	seen := map[int]bool{}
	for _, k := range keys {
		i, err := strconv.Atoi(k)
		if err != nil || i < 0 || i >= len(keys) || seen[i] {
			return false
		}
		seen[i] = true
	}

	return true
}

// UnmarshalYAML implements the goccy/go-yaml BytesUnmarshaler. It works from
// the AST rather than decoding into Go values so that a quoted number stays a
// string: 9600 is a numeric point and "9600" is a text one.
func (n *NodeYAML) UnmarshalYAML(b []byte) error {
	f, err := parser.ParseBytes(b, 0)
	if err != nil {
		return fmt.Errorf("error parsing node: %w", err)
	}

	if len(f.Docs) < 1 || f.Docs[0].Body == nil {
		return fmt.Errorf("empty node")
	}

	values, err := mappingValues(f.Docs[0].Body)
	if err != nil {
		return err
	}

	for _, v := range values {
		if v.Key.String() == "type" {
			return fmt.Errorf("this looks like the old export format, where a node names its type in a type: field. " +
				"Nodes are now spelled with the node type as the key: `- group:` with the points indented under it. " +
				"Re-export the tree with `siot export` to get a file in the current format")
		}
	}

	if len(values) != 1 {
		return fmt.Errorf("a node is a single key, the node type, with its points below it; found %v keys", len(values))
	}

	n.Type = values[0].Key.String()
	n.ID = ""
	n.Parent = ""
	n.Points = nil
	n.EdgePoints = nil
	n.Children = nil

	body := values[0].Value
	if body == nil {
		return nil
	}

	if _, ok := body.(*ast.NullNode); ok {
		return nil
	}

	fields, err := mappingValues(body)
	if err != nil {
		return fmt.Errorf("node %v: %w", n.Type, err)
	}

	for _, f := range fields {
		key := f.Key.String()

		switch key {
		case nodeKeyID:
			if err := yaml.NodeToValue(f.Value, &n.ID); err != nil {
				return fmt.Errorf("node %v: id: %w", n.Type, err)
			}
		case nodeKeyParent:
			if err := yaml.NodeToValue(f.Value, &n.Parent); err != nil {
				return fmt.Errorf("node %v: parent: %w", n.Type, err)
			}
		case nodeKeyChildren:
			if err := yaml.NodeToValue(f.Value, &n.Children); err != nil {
				return fmt.Errorf("node %v: children: %w", n.Type, err)
			}
		case nodeKeyPoints, nodeKeyEdgePoints:
			var pts []pointYAML
			if err := yaml.NodeToValue(f.Value, &pts); err != nil {
				return fmt.Errorf("node %v: %v: %w", n.Type, key, err)
			}

			points := make(Points, len(pts))
			for i, py := range pts {
				points[i] = pointFromLong(py)
			}

			if key == nodeKeyPoints {
				n.Points = append(n.Points, points...)
			} else {
				n.EdgePoints = append(n.EdgePoints, points...)
			}
		default:
			points, err := pointsFromNode(key, f.Value)
			if err != nil {
				return fmt.Errorf("node %v: %v: %w", n.Type, key, err)
			}
			n.Points = append(n.Points, points...)
		}
	}

	return nil
}

// mappingValues returns the key/value pairs of a mapping node, accepting both
// shapes the parser produces for one.
func mappingValues(node ast.Node) ([]*ast.MappingValueNode, error) {
	switch n := node.(type) {
	case *ast.MappingNode:
		return n.Values, nil
	case *ast.MappingValueNode:
		return []*ast.MappingValueNode{n}, nil
	default:
		return nil, fmt.Errorf("expected a mapping, found %v", node.Type())
	}
}

// pointsFromNode converts one key of a node body into points of that type. A
// scalar is a single point, a mapping is a point per key, and a sequence is a
// point per element keyed by its index.
func pointsFromNode(typ string, node ast.Node) (Points, error) {
	switch n := node.(type) {
	case *ast.MappingNode, *ast.MappingValueNode:
		values, err := mappingValues(n)
		if err != nil {
			return nil, err
		}

		points := make(Points, 0, len(values))
		for _, v := range values {
			p, err := pointFromScalar(typ, v.Key.String(), v.Value)
			if err != nil {
				return nil, err
			}
			points = append(points, p)
		}

		return points, nil

	case *ast.SequenceNode:
		points := make(Points, 0, len(n.Values))
		for i, v := range n.Values {
			p, err := pointFromScalar(typ, strconv.Itoa(i), v)
			if err != nil {
				return nil, err
			}
			points = append(points, p)
		}

		return points, nil

	default:
		p, err := pointFromScalar(typ, "", node)
		if err != nil {
			return nil, err
		}

		return Points{p}, nil
	}
}

// pointFromScalar builds a point from a scalar node, taking the data type from
// how the value is written.
func pointFromScalar(typ, key string, node ast.Node) (Point, error) {
	// keyless points carry "0" everywhere else in the system, and Points.Find
	// compares against that
	p := Point{Type: typ, Key: pointKey(key)}

	switch n := node.(type) {
	case *ast.StringNode:
		p.PutString(n.Value)
	case *ast.IntegerNode:
		var v float64
		if err := yaml.NodeToValue(node, &v); err != nil {
			return p, err
		}
		p.PutFloat(v)
	case *ast.FloatNode:
		p.PutFloat(n.Value)
	case *ast.BoolNode:
		if n.Value {
			p.PutFloat(1)
		} else {
			p.PutFloat(0)
		}
	case *ast.NullNode:
		// a point with no value at all
	default:
		return p, fmt.Errorf("a point value has to be a scalar, a mapping, or a sequence; found %v", node.Type())
	}

	return p, nil
}

// ToNodeEdge converts to the structure the rest of the system passes around.
// Points keep whatever IDs and parents the caller has resolved.
func (n NodeYAML) ToNodeEdge(id, parent string) NodeEdge {
	return NodeEdge{
		ID:         id,
		Type:       n.Type,
		Parent:     parent,
		Points:     n.Points,
		EdgePoints: n.EdgePoints,
	}
}

// NodeYAMLFromNodeEdge builds a file node from a tree node.
func NodeYAMLFromNodeEdge(ne NodeEdge, children []NodeYAML) NodeYAML {
	return NodeYAML{
		ID:         ne.ID,
		Type:       ne.Type,
		Points:     ne.Points,
		EdgePoints: ne.EdgePoints,
		Children:   children,
	}
}
