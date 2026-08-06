package client

import (
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
)

// planApply is unexported, so these tests live in the client package and work
// against a fixed tree snapshot rather than a server.

const testRoot = "root-id"

func testTree() []data.NodeEdge {
	return []data.NodeEdge{
		{
			ID:     testRoot,
			Type:   data.NodeTypeDevice,
			Parent: "none",
		},
		{
			ID:     "sensors-id",
			Type:   data.NodeTypeGroup,
			Parent: testRoot,
			Points: data.Points{data.NewPointString(data.PointTypeDescription, "0", "Sensors")},
		},
		{
			ID:     "modbus-id",
			Type:   "modbus",
			Parent: "sensors-id",
			Points: data.Points{
				data.NewPointString(data.PointTypeDescription, "0", "Modbus sensors"),
				data.NewPointFloat("baud", "0", 9600),
			},
		},
		{
			ID:     "admin-id",
			Type:   data.NodeTypeUser,
			Parent: testRoot,
			Points: data.Points{
				data.NewPointString(data.PointTypeFirstName, "0", "Admin"),
				data.NewPointString(data.PointTypeLastName, "0", "User"),
				data.NewPointString(data.PointTypeEmail, "0", "admin@example.com"),
			},
		},
	}
}

func planYAML(t *testing.T, in string) ApplyPlan {
	t.Helper()

	var f data.NodeFile
	if err := yaml.Unmarshal([]byte(in), &f); err != nil {
		t.Fatalf("error parsing: %v", err)
	}

	return planApply(f, testTree(), testRoot)
}

func noErrors(t *testing.T, plan ApplyPlan) {
	t.Helper()

	if len(plan.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", plan.Errors)
	}
}

func TestApplyCreatesWhatIsMissing(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - group:
      description: Rules
`)

	noErrors(t, plan)

	if len(plan.Send) != 1 {
		t.Fatalf("expected 1 send, got %v", len(plan.Send))
	}

	s := plan.Send[0]
	if !s.Created {
		t.Error("node should be created")
	}
	if s.Node.Parent != testRoot {
		t.Errorf("parent: got %v", s.Node.Parent)
	}
	if s.Node.ID == "" {
		t.Error("a created node needs an ID")
	}
}

func TestApplyDoesNothingWhenTreeAgrees(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - group:
      description: Sensors
      children:
        - modbus:
            description: Modbus sensors
            baud: 9600
`)

	noErrors(t, plan)

	if !plan.Empty() {
		t.Fatalf("expected nothing to do, got:\n%v", plan)
	}
}

func TestApplySendsOnlyChangedPoints(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - group:
      description: Sensors
      children:
        - modbus:
            description: Modbus sensors
            baud: 115200
            port: /dev/ttyS1
`)

	noErrors(t, plan)

	if len(plan.Send) != 1 {
		t.Fatalf("expected 1 send, got %v: %v", len(plan.Send), plan)
	}

	s := plan.Send[0]
	if s.Created {
		t.Error("modbus node already exists")
	}
	if s.Node.ID != "modbus-id" {
		t.Errorf("should have matched the existing node, got %v", s.Node.ID)
	}

	if len(s.Node.Points) != 2 {
		t.Fatalf("expected baud and port only, got %v", s.Node.Points)
	}

	if _, ok := s.Node.Points.Find(data.PointTypeDescription, ""); ok {
		t.Error("an unchanged description should not be sent")
	}
}

func TestApplyMatchesParentByDescription(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - variable:
      parent: Sensors
      description: Tank level
`)

	noErrors(t, plan)

	if len(plan.Send) != 1 {
		t.Fatalf("expected 1 send, got %v", len(plan.Send))
	}

	if plan.Send[0].Node.Parent != "sensors-id" {
		t.Errorf("should attach under the Sensors group, got %v", plan.Send[0].Node.Parent)
	}
}

func TestApplyParentCreatedEarlierInFile(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - group:
      description: Tank farm
  - variable:
      parent: Tank farm
      description: Tank level
`)

	noErrors(t, plan)

	if len(plan.Send) != 2 {
		t.Fatalf("expected 2 sends, got %v", len(plan.Send))
	}

	if plan.Send[1].Node.Parent != plan.Send[0].Node.ID {
		t.Errorf("the variable should attach to the group the file just created")
	}
}

func TestApplyMissingParent(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - variable:
      parent: Nowhere
      description: Tank level
`)

	if len(plan.Errors) != 1 {
		t.Fatalf("expected 1 error, got %v", plan.Errors)
	}

	if !strings.Contains(plan.Errors[0].Error(), "Nowhere") {
		t.Errorf("the error should name the parent: %v", plan.Errors[0])
	}

	if !plan.Empty() {
		t.Error("nothing should be applied for a failed entry")
	}
}

func TestApplyAmbiguousMatch(t *testing.T) {
	tree := testTree()
	tree = append(tree, data.NodeEdge{
		ID:     "sensors-id-2",
		Type:   data.NodeTypeGroup,
		Parent: testRoot,
		Points: data.Points{data.NewPointString(data.PointTypeDescription, "0", "Sensors")},
	})

	var f data.NodeFile
	if err := yaml.Unmarshal([]byte(`
nodes:
  - group:
      description: Sensors
`), &f); err != nil {
		t.Fatal(err)
	}

	plan := planApply(f, tree, testRoot)

	if len(plan.Errors) != 1 {
		t.Fatalf("expected an ambiguity error, got %v", plan.Errors)
	}

	if !plan.Empty() {
		t.Error("an ambiguous entry should not be applied")
	}
}

func TestApplyTypeMismatch(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - variable:
      description: Sensors
`)

	if len(plan.Errors) != 1 {
		t.Fatalf("expected a type error, got %v", plan.Errors)
	}

	if !strings.Contains(plan.Errors[0].Error(), "group") {
		t.Errorf("the error should name the existing type: %v", plan.Errors[0])
	}

	if !plan.Empty() {
		t.Error("a type mismatch should not create a second node")
	}
}

func TestApplySingletonMatchesOnType(t *testing.T) {
	tree := testTree()
	tree = append(tree, data.NodeEdge{
		ID:     "metrics-id",
		Type:   data.NodeTypeMetrics,
		Parent: testRoot,
		Points: data.Points{data.NewPointFloat("period", "0", 60)},
	})

	var f data.NodeFile
	if err := yaml.Unmarshal([]byte(`
nodes:
  - metrics:
      period: 120
`), &f); err != nil {
		t.Fatal(err)
	}

	plan := planApply(f, tree, testRoot)
	noErrors(t, plan)

	if len(plan.Send) != 1 || plan.Send[0].Node.ID != "metrics-id" {
		t.Fatalf("a node with no description should match the one node of its type: %v", plan)
	}
}

func TestApplyUserMatchesOnEmail(t *testing.T) {
	// the last name changes, and the user is still matched on email
	plan := planYAML(t, `
nodes:
  - user:
      firstName: Admin
      lastName: Person
      email: admin@example.com
`)

	noErrors(t, plan)

	if len(plan.Send) != 1 {
		t.Fatalf("expected 1 send, got %v", len(plan.Send))
	}

	s := plan.Send[0]
	if s.Created {
		t.Fatal("the user should have been matched, not created")
	}
	if s.Node.ID != "admin-id" {
		t.Errorf("matched the wrong node: %v", s.Node.ID)
	}

	if len(s.Node.Points) != 1 {
		t.Fatalf("only the last name changed, got %v", s.Node.Points)
	}
}

func TestApplyDelete(t *testing.T) {
	plan := planYAML(t, `
delete:
  - modbus:
      parent: Sensors
      description: Modbus sensors
`)

	noErrors(t, plan)

	if len(plan.Delete) != 1 {
		t.Fatalf("expected 1 delete, got %v", len(plan.Delete))
	}

	if plan.Delete[0].ID != "modbus-id" || plan.Delete[0].Parent != "sensors-id" {
		t.Errorf("wrong node: %v", plan.Delete[0])
	}
}

func TestApplyDeleteMissingIsNoOp(t *testing.T) {
	plan := planYAML(t, `
delete:
  - group:
      description: Never existed
`)

	noErrors(t, plan)

	if !plan.Empty() {
		t.Errorf("deleting what is not there should do nothing, got %v", plan)
	}
}

func TestApplyResolvesReferencesByDescription(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - variable:
      description: Tank level
  - rule:
      description: Tank low
      children:
        - condition:
            description: Level below 10
            nodeID: Tank level
            value: 10
`)

	noErrors(t, plan)

	var variableID, conditionRef string

	for _, s := range plan.Send {
		if s.Node.Type == data.NodeTypeVariable {
			variableID = s.Node.ID
		}

		if s.Node.Type == data.NodeTypeCondition {
			p, ok := s.Node.Points.Find(data.PointTypeNodeID, "")
			if !ok {
				t.Fatal("condition should carry a nodeID point")
			}
			conditionRef = p.Txt()
		}
	}

	if variableID == "" || conditionRef == "" {
		t.Fatalf("expected both nodes in the plan: %v", plan)
	}

	if conditionRef != variableID {
		t.Errorf("the reference should resolve to the variable's ID: got %v, want %v", conditionRef, variableID)
	}
}

func TestApplyReferenceForward(t *testing.T) {
	// the rule refers to a variable the file creates further down
	plan := planYAML(t, `
nodes:
  - rule:
      description: Tank low
      children:
        - condition:
            description: Level below 10
            nodeID: Tank level
  - variable:
      description: Tank level
`)

	noErrors(t, plan)

	var variableID, ref string

	for _, s := range plan.Send {
		if s.Node.Type == data.NodeTypeVariable {
			variableID = s.Node.ID
		}

		if s.Node.Type == data.NodeTypeCondition {
			p, _ := s.Node.Points.Find(data.PointTypeNodeID, "")
			ref = p.Txt()
		}
	}

	if ref == "" || ref != variableID {
		t.Errorf("a reference should resolve forward: got %v, want %v", ref, variableID)
	}
}

func TestApplyReferenceNotFound(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - rule:
      description: Tank low
      children:
        - condition:
            description: Level below 10
            nodeID: Nothing here
`)

	if len(plan.Errors) != 1 {
		t.Fatalf("expected an error for an unresolvable reference, got %v", plan.Errors)
	}

	if !strings.Contains(plan.Errors[0].Error(), "Nothing here") {
		t.Errorf("the error should name the reference: %v", plan.Errors[0])
	}
}

func TestApplyIntegerAndFloatAreOneValue(t *testing.T) {
	tree := testTree()
	tree = append(tree, data.NodeEdge{
		ID:     "counter-id",
		Type:   "variable",
		Parent: testRoot,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "0", "Counter"),
			data.NewPointInt("value", "0", 5),
		},
	})

	var f data.NodeFile
	if err := yaml.Unmarshal([]byte(`
nodes:
  - variable:
      description: Counter
      value: 5
`), &f); err != nil {
		t.Fatal(err)
	}

	plan := planApply(f, tree, testRoot)
	noErrors(t, plan)

	if !plan.Empty() {
		t.Errorf("an integer 5 and a file saying 5 are the same value, got:\n%v", plan)
	}
}

func TestApplyCreatedNodesGetFreshUUIDs(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - group:
      description: Rules
`)

	noErrors(t, plan)

	if _, err := uuid.Parse(plan.Send[0].Node.ID); err != nil {
		t.Errorf("a created node should get a fresh UUID, got %v", plan.Send[0].Node.ID)
	}
}

func TestApplySecondPassIsEmpty(t *testing.T) {
	in := `
nodes:
  - group:
      description: Rules
      children:
        - rule:
            description: Tank low
  - variable:
      parent: Sensors
      description: Tank level
      value: 10
`

	var f data.NodeFile
	if err := yaml.Unmarshal([]byte(in), &f); err != nil {
		t.Fatal(err)
	}

	tree := testTree()
	plan := planApply(f, tree, testRoot)
	noErrors(t, plan)

	if len(plan.Send) != 3 {
		t.Fatalf("expected 3 nodes created, got %v: %v", len(plan.Send), plan)
	}

	// apply the plan to the snapshot, the way a server would
	for _, s := range plan.Send {
		tree = append(tree, data.NodeEdge{
			ID:     s.Node.ID,
			Type:   s.Node.Type,
			Parent: s.Node.Parent,
			Points: s.Node.Points,
		})
	}

	second := planApply(f, tree, testRoot)
	noErrors(t, second)

	if !second.Empty() {
		t.Errorf("applying the same file again should do nothing, got:\n%v", second)
	}
}

func TestApplyNullPointDoesNotClearAValue(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - group:
      description: Sensors
      children:
        - modbus:
            description: Modbus sensors
            baud:
`)

	noErrors(t, plan)

	if !plan.Empty() {
		t.Errorf("a point with no value says nothing to assert, got:\n%v", plan)
	}
}

func TestApplyParentOnChildIsAnError(t *testing.T) {
	plan := planYAML(t, `
nodes:
  - group:
      description: Rules
      children:
        - rule:
            parent: Sensors
            description: Tank low
`)

	if len(plan.Errors) != 1 {
		t.Fatalf("expected an error for parent: on a child, got %v", plan.Errors)
	}
}
