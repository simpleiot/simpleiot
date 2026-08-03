package client_test

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// FIXME, need tests for duplicate, move, and mirror node

func TestExportNodes(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	y, err := client.ExportNodes(nc, root.ID)

	if err != nil {
		t.Fatal("Error exporting nodes: ", err)
	}

	// convert back to nodes and check a few
	var exp data.NodeFile

	err = yaml.Unmarshal(y, &exp)
	if err != nil {
		t.Fatal("Unmarshal error: ", err)
	}

	if len(exp.Nodes) < 1 {
		t.Fatal("no top level node")
	}

	// the root device node is the instance rather than configuration, so an
	// export starts with what is under it
	for _, n := range exp.Nodes {
		if n.Type == data.NodeTypeDevice {
			t.Fatal("the root device node should not be exported")
		}
	}

	// and the file carries no node IDs at all
	if strings.Contains(string(y), root.ID) {
		t.Errorf("a file should carry no node IDs:\n%v", string(y))
	}

	if exp.Nodes[0].Type != data.NodeTypeUser {
		t.Fatal("expected the admin user, got: ", exp.Nodes[0].Type)
	}
}

func TestExportDuplicateDescriptions(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	for range 2 {
		err = client.SendNode(nc, data.NodeEdge{
			ID:     uuid.New().String(),
			Type:   data.NodeTypeGroup,
			Parent: root.ID,
			Points: data.Points{data.NewPointString(data.PointTypeDescription, "", "Sensors")},
		}, "test")

		if err != nil {
			t.Fatal("Error sending node: ", err)
		}
	}

	_, err = client.ExportNodes(nc, root.ID)
	if err == nil {
		t.Fatal("Two nodes described the same, so an export without IDs should have failed")
	}

	if !strings.Contains(err.Error(), "Sensors") {
		t.Error("the error should name the description: ", err)
	}

}

func TestExportImportNodes(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	ne, err := client.UserCheck(nc, "admin", "admin")
	if err != nil {
		t.Fatal("User check error: ", err)
	}

	if len(ne) != 2 {
		t.Fatal("Expected exactly nodes from auth request")
	}

	y, err := client.ExportNodes(nc, root.ID)

	if err != nil {
		t.Fatal("Error exporting nodes: ", err)
	}

	// fmt.Println("export: ", string(y))

	plan, err := client.ImportNodes(nc, y, "test", false)

	if err != nil {
		t.Fatal("Error importing nodes: ", err)
	}

	if len(plan.Errors) > 0 {
		t.Fatal("Import errors: ", plan.Errors)
	}

	// importing an export of the same tree changes nothing, since every node
	// in the file matches the node it came from
	if !plan.Empty() {
		t.Fatal("Importing a tree's own export should do nothing, got:\n", plan)
	}

	// the original device node is updated in place rather than replaced
	ne, err = client.GetNodes(nc, "all", "inst1", "", false)
	if err != nil {
		t.Fatal("Error getting original device node: ", err)
	}

	if len(ne) != 1 {
		t.Fatal("Original device node should still be here, got: ", len(ne))
	}

	// check user auth check
	ne, err = client.UserCheck(nc, "admin", "admin")
	if err != nil {
		t.Fatal("User check error: ", err)
	}

	// should return exactly 2 nodes, a user and jwt node
	if len(ne) != 2 {
		fmt.Println("ne: ", ne)
		t.Fatal("Expected at exactly two nodes from auth request, got: ", len(ne))
	}

	userNodeFound := false

	for _, n := range ne {
		if n.Type == data.NodeTypeUser {
			userNodeFound = true

			nodes, err := client.GetNodesForUser(nc, n.ID)
			if err != nil {
				log.Println("Error getting nodes for user:", err)
			}

			// there should be two nodes in the new system -- a device and user node
			if len(nodes) != 2 {
				fmt.Println("nodes for user: ", nodes)
				t.Fatal("Should be exactly 2 nodes for user after import, got: ", len(nodes))
			}
		}
	}

	if !userNodeFound {
		t.Fatal("User node not found")
	}

	ne, err = client.GetNodes(nc, "root", "all", "", false)
	if err != nil {
		t.Fatal("error getting nodes: ", err)
	}

	if len(ne) != 1 {
		t.Fatal("Expected only one device node")
	}

	// the import updated the device node rather than replacing it
	if ne[0].ID != "inst1" {
		t.Fatal("Expected the original device node, got: ", ne[0].ID)
	}
}

var testImportNodesYaml = `
nodes:
  - group:
      description: group 1
      children:
        - variable:
            description: var 1
            value: 10
`

func TestImportNodes(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	plan, err := client.ImportNodes(nc, []byte(testImportNodesYaml), "test", false)
	if err != nil {
		t.Fatal("Error importing: ", err)
	}

	if len(plan.Errors) > 0 {
		t.Fatal("Import errors: ", plan.Errors)
	}

	// importing the same file again does nothing
	plan, err = client.ImportNodes(nc, []byte(testImportNodesYaml), "test", false)
	if err != nil {
		t.Fatal("Error importing a second time: ", err)
	}

	if !plan.Empty() {
		t.Fatal("A second import should do nothing, got:\n", plan)
	}

	children, err := client.GetNodes(nc, root.ID, "all", "", false)
	if err != nil {
		t.Fatal("Error getting children: ", err)
	}

	if len(children) < 2 {
		t.Fatal("Should be at least 2 children")
	}

	var g data.NodeEdge

	for _, c := range children {
		if c.Type == data.NodeTypeGroup {
			g = c
			break
		}
	}

	if g.Type != data.NodeTypeGroup {
		t.Fatal("group node not found")
	}

	children, err = client.GetNodes(nc, g.ID, "all", "", false)
	if err != nil {
		t.Fatal("error getting group children")
	}

	if len(children) < 1 {
		t.Fatal("Group should have at least 1 child")
	}
}

var testImportNodesYamlWithIDs = `
nodes:
  - group:
      description: group 1
      children:
        - variable:
            description: var 1
            value: 10
`

func TestImportNodesGeneratesIDs(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	plan, err := client.ImportNodes(nc, []byte(testImportNodesYamlWithIDs), "test", false)
	if err != nil {
		t.Fatal("Error importing: ", err)
	}

	if len(plan.Errors) > 0 {
		t.Fatal("Import errors: ", plan.Errors)
	}

	children, err := client.GetNodes(nc, root.ID, "all", "", false)
	if err != nil {
		t.Fatal("Error getting children: ", err)
	}

	var g data.NodeEdge

	for _, c := range children {
		if c.Type == data.NodeTypeGroup {
			g = c
			break
		}
	}

	if g.Type != data.NodeTypeGroup {
		t.Fatal("group node not found")
	}

	if g.ID == "" {
		t.Fatal("the created node should have an ID")
	}

	if _, err := uuid.Parse(g.ID); err != nil {
		t.Fatal("a created node gets a fresh UUID, got: ", g.ID)
	}

	children, err = client.GetNodes(nc, g.ID, "all", "", false)
	if err != nil {
		t.Fatal("error getting group children")
	}

	if len(children) < 1 {
		t.Fatal("Group should have at least 1 child")
	}

}

var testImportListOfNodesYaml = `
nodes:
  - variable:
      parent: Test sensor group
      description: temperature sensor
      value: 23.5
  - variable:
      parent: Test sensor group
      description: humidity sensor
      value: 65.0
  - variable:
      parent: Test sensor group
      description: pressure sensor
      value: 1013.25
`

func TestImportListOfNodes(t *testing.T) {
	nc, root, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	// First create a group node
	groupNode := data.NodeEdge{
		ID:     "test-group-123",
		Type:   data.NodeTypeGroup,
		Parent: root.ID,
		Points: []data.Point{
			data.NewPointString(data.PointTypeDescription, "", "Test sensor group"),
		},
	}

	err = client.SendNode(nc, groupNode, "test")
	if err != nil {
		t.Fatal("Error creating group node: ", err)
	}

	// Now import 3 variable nodes under this group
	plan, err := client.ImportNodes(nc, []byte(testImportListOfNodesYaml), "test", false)
	if err != nil {
		t.Fatal("Error importing variable nodes: ", err)
	}

	if len(plan.Errors) > 0 {
		t.Fatal("Import errors: ", plan.Errors)
	}

	// Verify the variables were imported
	variables, err := client.GetNodes(nc, groupNode.ID, "all", "", false)
	if err != nil {
		t.Fatal("Error getting variable nodes: ", err)
	}

	if len(variables) != 3 {
		t.Fatalf("Expected 3 variable nodes, got %d", len(variables))
	}

	// Verify all are variable type and check their points
	descriptions := make([]string, 0)
	for _, v := range variables {
		if v.Type != data.NodeTypeVariable {
			t.Fatalf("Expected variable node, got %s", v.Type)
		}

		// Check the points embedded in the node
		for _, point := range v.Points {
			if point.Type == data.PointTypeDescription {
				descriptions = append(descriptions, point.Txt())
			}
		}
	}

	expectedDescriptions := []string{"temperature sensor", "humidity sensor", "pressure sensor"}
	if len(descriptions) != 3 {
		t.Fatalf("Expected 3 descriptions, got %d", len(descriptions))
	}

	for _, expected := range expectedDescriptions {
		if !slices.Contains(descriptions, expected) {
			t.Fatalf("Expected description '%s' not found", expected)
		}
	}
}

// readFixture returns the tree fixture shared with the data and server tests.
func readFixture(t *testing.T) []byte {
	t.Helper()

	b, err := os.ReadFile("../testdata/tree.yaml")
	if err != nil {
		t.Fatal("Error reading fixture: ", err)
	}

	return b
}

// findByDesc returns the one node below root with the given description.
func findByDesc(t *testing.T, nc *nats.Conn, rootID, desc string) data.NodeEdge {
	t.Helper()

	var found []data.NodeEdge

	var walk func(id string)
	walk = func(id string) {
		children, err := client.GetNodes(nc, id, "all", "", false)
		if err != nil {
			t.Fatal("Error getting children: ", err)
		}

		for _, c := range children {
			if c.Points.MatchKey() == desc {
				found = append(found, c)
			}
			walk(c.ID)
		}
	}

	walk(rootID)

	if len(found) != 1 {
		t.Fatalf("expected exactly one node described %q, found %v", desc, len(found))
	}

	return found[0]
}

func TestImportFixture(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	fixture := readFixture(t)

	plan, err := client.ImportNodes(nc, fixture, "test", false)
	if err != nil {
		t.Fatal("Error importing fixture: ", err)
	}

	if len(plan.Errors) > 0 {
		t.Fatal("Import errors: ", plan.Errors)
	}

	sensors := findByDesc(t, nc, root.ID, "Sensors")
	if sensors.Parent != root.ID {
		t.Error("Sensors should be a child of the root node")
	}

	tankFarm := findByDesc(t, nc, root.ID, "Tank farm")
	if tankFarm.Parent != sensors.ID {
		t.Error("Tank farm should be nested under Sensors")
	}

	// these two entries are at the top of the file and find their parent by
	// description rather than by nesting
	modbus := findByDesc(t, nc, root.ID, "Modbus sensors")
	if modbus.Parent != tankFarm.ID {
		t.Errorf("Modbus sensors should attach under Tank farm, got parent %v", modbus.Parent)
	}

	if modbus.Type != "modbus" {
		t.Errorf("expected a modbus node, got %v", modbus.Type)
	}

	variable := findByDesc(t, nc, root.ID, "Tank level")
	if variable.Parent != tankFarm.ID {
		t.Errorf("Tank level should attach under Tank farm, got parent %v", variable.Parent)
	}

	// a text point that looks like a number stays text, and a numeric one
	// stays numeric
	if port, ok := modbus.Points.Find("port", ""); !ok || port.Txt() != "/dev/ttyS1" {
		t.Errorf("port point: %v", port)
	}

	if baud, ok := modbus.Points.Find("baud", ""); !ok || baud.Val() != 9600 {
		t.Errorf("baud point: %v", baud)
	}

	// the condition refers to the variable by label, which resolves to its ID
	condition := findByDesc(t, nc, root.ID, "Level below 10")
	ref, ok := condition.Points.Find(data.PointTypeNodeID, "")
	if !ok {
		t.Fatal("condition should carry a nodeID point")
	}

	if ref.Txt() != variable.ID {
		t.Errorf("nodeID should resolve to the variable: got %v, want %v", ref.Txt(), variable.ID)
	}

	// the user is created with its role edge point
	user := findByDesc(t, nc, root.ID, "admin@example.com")
	if role, ok := user.EdgePoints.Find(data.PointTypeRole, ""); !ok || role.Txt() != "admin" {
		t.Errorf("user role edge point: %v", role)
	}

	// applying the same file again does nothing
	plan, err = client.ImportNodes(nc, fixture, "test", false)
	if err != nil {
		t.Fatal("Error importing fixture a second time: ", err)
	}

	if !plan.Empty() {
		t.Fatalf("a second import should do nothing, got:\n%v", plan)
	}
}

func TestImportFixtureUpdatesAndDeletes(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	if _, err := client.ImportNodes(nc, readFixture(t), "test", false); err != nil {
		t.Fatal("Error importing fixture: ", err)
	}

	// change one point
	plan, err := client.ImportNodes(nc, []byte(`
nodes:
  - modbus:
      parent: Tank farm
      description: Modbus sensors
      baud: 115200
`), "test", false)
	if err != nil {
		t.Fatal("Error importing update: ", err)
	}

	if len(plan.Send) != 1 || len(plan.Send[0].Node.Points) != 1 {
		t.Fatalf("expected one point to be sent, got:\n%v", plan)
	}

	modbus := findByDesc(t, nc, root.ID, "Modbus sensors")
	if baud, ok := modbus.Points.Find("baud", ""); !ok || baud.Val() != 115200 {
		t.Errorf("baud should have been updated, got %v", baud)
	}

	// and remove a node
	plan, err = client.ImportNodes(nc, []byte(`
delete:
  - modbus:
      parent: Tank farm
      description: Modbus sensors
`), "test", false)
	if err != nil {
		t.Fatal("Error importing delete: ", err)
	}

	if len(plan.Delete) != 1 {
		t.Fatalf("expected one delete, got:\n%v", plan)
	}

	nodes, err := client.GetNodes(nc, "all", modbus.ID, "", false)
	if err != nil {
		t.Fatal("Error getting deleted node: ", err)
	}

	if len(nodes) > 0 {
		t.Error("the modbus node should be gone")
	}

	// deleting what is already gone is a no-op
	plan, err = client.ImportNodes(nc, []byte(`
delete:
  - modbus:
      parent: Tank farm
      description: Modbus sensors
`), "test", false)
	if err != nil {
		t.Fatal("Error importing delete a second time: ", err)
	}

	if !plan.Empty() {
		t.Errorf("deleting a node that is gone should do nothing, got:\n%v", plan)
	}
}

func TestExportRoundTripBetweenInstances(t *testing.T) {
	nc1, root1, stop1, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop1()

	if _, err := client.ImportNodes(nc1, readFixture(t), "test", false); err != nil {
		t.Fatal("Error importing fixture: ", err)
	}

	first, err := client.ExportNodes(nc1, root1.ID)
	if err != nil {
		t.Fatal("Error exporting: ", err)
	}

	nc2, root2, stop2, err := server.TestServer("second")
	if err != nil {
		t.Fatal("Error starting second test server: ", err)
	}

	defer stop2()

	plan, err := client.ImportNodes(nc2, first, "test", false)
	if err != nil {
		t.Fatal("Error importing into the second instance: ", err)
	}

	if len(plan.Errors) > 0 {
		t.Fatal("Import errors: ", plan.Errors)
	}

	second, err := client.ExportNodes(nc2, root2.ID)
	if err != nil {
		t.Fatal("Error exporting the second instance: ", err)
	}

	if string(first) != string(second) {
		t.Errorf("an export imported elsewhere should export identically:\n--- first ---\n%v\n--- second ---\n%v",
			string(first), string(second))
	}
}
