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

// Keys that have a meaning of their own inside a node body. Every other key is
// a point type. None of these is a point type, so the rule holds without
// exceptions -- in particular `id` is an ordinary point type, used by Modbus
// and OneWire nodes, and a node's own ID never appears in a file at all.
const (
	nodeKeyParent     = "parent"
	nodeKeyChildren   = "children"
	nodeKeyEdgePoints = "edgePoints"
)

func nodeKeyReserved(k string) bool {
	switch k {
	case nodeKeyParent, nodeKeyChildren, nodeKeyEdgePoints:
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

// NodeYAML is one node in a node file. The node type is the key and each point
// type is a key of its own:
//
//	nodes:
//	  - modbus:
//	      description: Modbus sensors
//	      port: /dev/ttyS1
//	      baud: 9600
//
// A file carries configuration and nothing else: no node IDs, no origins, and
// no points that carry no value.
type NodeYAML struct {
	// Type is the node type, which is the key the rest of the body hangs from.
	Type string
	// Parent is the match key of the node this one attaches to, and is only
	// meaningful on a top level entry.
	Parent     string
	Points     Points
	EdgePoints Points
	Children   []NodeYAML
}

// pointKey normalizes a point key, since an empty key and "0" mean the same
// thing everywhere else in the system.
func pointKey(k string) string {
	if k == "" {
		return "0"
	}

	return k
}

// writable reports whether a point has a spelling in a file. A tombstoned
// point is not configuration, and raw bytes have no readable form; no node type
// configures either today.
func writable(p Point) bool {
	if p.Tombstone != 0 || nodeKeyReserved(p.Type) {
		return false
	}

	switch p.DataType {
	case PointDataTypeString, PointDataTypeFloat, PointDataTypeInt:
		return true
	case PointDataTypeUnknown:
		return len(p.Data) == 0
	default:
		return false
	}
}

// scalarValue returns the YAML value for a point. A numeric point is written
// bare when its value is integral and with its decimal otherwise, so that files
// do not carry a decimal point on nearly every number.
func scalarValue(p Point) any {
	switch p.DataType {
	case PointDataTypeString:
		return p.Txt()
	case PointDataTypeInt:
		return int64(p.Val())
	case PointDataTypeFloat:
		v := p.Val()
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

	if n.Parent != "" {
		body = append(body, yaml.MapItem{Key: nodeKeyParent, Value: n.Parent})
	}

	body = append(body, pointsToYAML(n.Points)...)

	if edge := pointsToYAML(n.EdgePoints); len(edge) > 0 {
		body = append(body, yaml.MapItem{Key: nodeKeyEdgePoints, Value: edge})
	}

	if len(n.Children) > 0 {
		body = append(body, yaml.MapItem{Key: nodeKeyChildren, Value: n.Children})
	}

	return yaml.MapSlice{{Key: n.Type, Value: body}}, nil
}

// pointsToYAML writes points as keys, ordered by type so that exporting an
// unchanged tree produces an identical file.
func pointsToYAML(points Points) yaml.MapSlice {
	groups := map[string]Points{}
	types := []string{}

	for _, p := range points {
		if !writable(p) {
			continue
		}

		if _, ok := groups[p.Type]; !ok {
			types = append(types, p.Type)
		}

		groups[p.Type] = append(groups[p.Type], p)
	}

	sort.Strings(types)

	out := yaml.MapSlice{}

	for _, typ := range types {
		out = append(out, yaml.MapItem{Key: typ, Value: groupValue(groups[typ])})
	}

	return out
}

// groupValue renders the points of one type: a scalar when there is a single
// keyless point, a sequence when the keys are 0..n-1, and a mapping otherwise.
func groupValue(group Points) any {
	if len(group) == 1 && pointKey(group[0].Key) == "0" {
		return scalarValue(group[0])
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
			seq[i] = scalarValue(byKey[strconv.Itoa(i)])
		}

		return seq
	}

	ms := yaml.MapSlice{}
	for _, k := range keys {
		ms = append(ms, yaml.MapItem{Key: k, Value: scalarValue(byKey[k])})
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
// the AST rather than decoding into Go values so that how a value is written
// decides what it becomes: 9600 is a numeric point, "9600" is a text one, and
// 1 and 1.5 are an integer and a float.
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

	*n = NodeYAML{Type: values[0].Key.String()}

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
		case nodeKeyParent:
			if err := yaml.NodeToValue(f.Value, &n.Parent); err != nil {
				return fmt.Errorf("node %v: parent: %w", n.Type, err)
			}
		case nodeKeyChildren:
			if err := yaml.NodeToValue(f.Value, &n.Children); err != nil {
				return fmt.Errorf("node %v: children: %w", n.Type, err)
			}
		case nodeKeyEdgePoints:
			edge, err := edgePointsFromNode(f.Value)
			if err != nil {
				return fmt.Errorf("node %v: edgePoints: %w", n.Type, err)
			}

			n.EdgePoints = append(n.EdgePoints, edge...)
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

// edgePointsFromNode reads the edgePoints: mapping, which spells points the
// same way the node body does.
func edgePointsFromNode(node ast.Node) (Points, error) {
	values, err := mappingValues(node)
	if err != nil {
		return nil, err
	}

	var out Points

	for _, v := range values {
		points, err := pointsFromNode(v.Key.String(), v.Value)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", v.Key.String(), err)
		}

		out = append(out, points...)
	}

	return out, nil
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
	case *ast.LiteralNode:
		// a block scalar, written with | or >, which is how a file carries
		// text with newlines in it
		if n.Value == nil {
			p.PutString("")
			break
		}

		p.PutString(n.Value.Value)
	case *ast.IntegerNode:
		var v int64
		if err := yaml.NodeToValue(node, &v); err != nil {
			return p, err
		}

		p.PutInt(v)
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
func (n NodeYAML) ToNodeEdge(id, parent string) NodeEdge {
	return NodeEdge{
		ID:         id,
		Type:       n.Type,
		Parent:     parent,
		Points:     n.Points,
		EdgePoints: n.EdgePoints,
	}
}

// SameValue reports whether two points say the same thing. Numbers are
// compared by value and text by string, rather than by their stored bytes, so
// that an integer 5 and a float 5 are one value and a file does not fight a
// client that writes its points with PutInt.
func SameValue(a, b Point) bool {
	switch {
	case isNumeric(a) && isNumeric(b):
		return a.Val() == b.Val()
	case a.DataType == PointDataTypeString && b.DataType == PointDataTypeString:
		return a.Txt() == b.Txt()
	default:
		return a.DataType == b.DataType && string(a.Data) == string(b.Data)
	}
}

func isNumeric(p Point) bool {
	return p.DataType == PointDataTypeFloat || p.DataType == PointDataTypeInt
}
