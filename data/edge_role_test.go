package data

import (
	"os"
	"regexp"
	"testing"
)

func TestEdgeRole(t *testing.T) {
	tests := []struct {
		name   string
		points Points
		exp    EdgeRole
	}{
		{"no points", nil, EdgeRoleNone},
		{"other points only",
			Points{NewPointFloat(PointTypeTombstone, "", 0)}, EdgeRoleNone},
		{"primary",
			Points{NewPointFloat(PointTypePrimary, "", 1)}, EdgeRolePrimary},
		{"mirror",
			Points{NewPointFloat(PointTypeMirror, "", 1)}, EdgeRoleMirror},
		{"primary cleared",
			Points{NewPointFloat(PointTypePrimary, "", 0)}, EdgeRoleNone},
		{"mirror cleared",
			Points{NewPointFloat(PointTypeMirror, "", 0)}, EdgeRoleNone},
		// the store rewrites a blank scalar key to "0", so a role written
		// in memory has to read the same way after a round trip
		{"primary with stored key",
			Points{NewPointFloat(PointTypePrimary, "0", 1)}, EdgeRolePrimary},
		{"mirror with stored key",
			Points{NewPointFloat(PointTypeMirror, "0", 1)}, EdgeRoleMirror},
		{"primary set, mirror cleared",
			Points{
				NewPointFloat(PointTypePrimary, "", 1),
				NewPointFloat(PointTypeMirror, "", 0),
			}, EdgeRolePrimary},
		// an edge should never carry both, but if it does, not running a
		// client is the safe direction to fail
		{"both set",
			Points{
				NewPointFloat(PointTypePrimary, "", 1),
				NewPointFloat(PointTypeMirror, "", 1),
			}, EdgeRoleMirror},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ne := NodeEdge{EdgePoints: test.points}
			if got := ne.EdgeRole(); got != test.exp {
				t.Errorf("NodeEdge.EdgeRole() = %v, expected %v", got, test.exp)
			}

			e := Edge{Points: test.points}
			if got := e.Role(); got != test.exp {
				t.Errorf("Edge.Role() = %v, expected %v", got, test.exp)
			}
		})
	}
}

// nodeTypeConst matches the node type constants declared in schema.go. Adding
// a client means adding one of these, which is what makes the coverage test
// below a reliable check that the new type was classified.
var nodeTypeConst = regexp.MustCompile(`NodeType\w+\s*=\s*"([^"]+)"`)

// schemaNodeTypes returns every node type value declared in schema.go.
func schemaNodeTypes(t *testing.T) []string {
	t.Helper()

	src, err := os.ReadFile("schema.go")
	if err != nil {
		t.Fatal("error reading schema.go:", err)
	}

	matches := nodeTypeConst.FindAllStringSubmatch(string(src), -1)
	if len(matches) < 30 {
		t.Fatalf("only found %v node types in schema.go, the pattern is probably wrong",
			len(matches))
	}

	types := make([]string, 0, len(matches))
	for _, m := range matches {
		types = append(types, m[1])
	}

	return types
}

// TestNodeTypesAreClassified fails when a node type is in neither
// primaryTypes nor treeScopedTypes. Deciding whether a new node type owns
// something outside the tree is a decision the client author has to make, and
// leaving it out would silently produce a type whose mirrors run a second
// client. Nothing else would catch that, so this test does.
func TestNodeTypesAreClassified(t *testing.T) {
	for _, typ := range schemaNodeTypes(t) {
		primary := primaryTypes[typ]
		treeScoped := treeScopedTypes[typ]

		switch {
		case primary && treeScoped:
			t.Errorf("node type %q is in both primaryTypes and treeScopedTypes", typ)
		case !primary && !treeScoped:
			t.Errorf("node type %q is classified in neither primaryTypes nor "+
				"treeScopedTypes -- does a node of this type own something "+
				"outside the tree (a bus, a socket, a destination), or does it "+
				"take its meaning from where it sits?", typ)
		}
	}
}

// TestClassificationNamesRealTypes catches a typo or a stale entry in any of
// the three maps.
func TestClassificationNamesRealTypes(t *testing.T) {
	known := make(map[string]bool)
	for _, typ := range schemaNodeTypes(t) {
		known[typ] = true
	}

	check := func(mapName, typ string) {
		if !known[typ] {
			t.Errorf("%v names %q, which is not a node type in schema.go",
				mapName, typ)
		}
	}

	for typ := range primaryTypes {
		check("primaryTypes", typ)
	}

	for typ := range treeScopedTypes {
		check("treeScopedTypes", typ)
	}

	for typ, owner := range nodeTypeOwners {
		check("nodeTypeOwners", typ)
		check("nodeTypeOwners", owner)
	}
}

func TestNodeTypeLookups(t *testing.T) {
	if !NodeTypeIsPrimary(NodeTypeGPIO) {
		t.Error("expected gpio to be primary")
	}

	if NodeTypeIsPrimary(NodeTypeUser) {
		t.Error("expected user not to be primary")
	}

	// a type the system knows nothing about keeps behaving as it does today
	if NodeTypeIsPrimary("someCustomType") {
		t.Error("expected an unknown type not to be primary")
	}

	if got := NodeTypeOwner(NodeTypeModbusIO); got != NodeTypeModbus {
		t.Errorf("NodeTypeOwner(modbusIo) = %q, expected %q", got, NodeTypeModbus)
	}

	if got := NodeTypeOwner(NodeTypeGPIO); got != "" {
		t.Errorf("NodeTypeOwner(gpio) = %q, expected a gpio node to be freely movable", got)
	}
}
