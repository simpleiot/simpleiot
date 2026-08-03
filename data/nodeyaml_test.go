package data_test

import (
	"os"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/simpleiot/simpleiot/data"
)

// readFixture returns the tree fixture shared with the client and server tests.
func readFixture(t *testing.T) []byte {
	t.Helper()

	b, err := os.ReadFile("../testdata/tree.yaml")
	if err != nil {
		t.Fatalf("error reading fixture: %v", err)
	}

	return b
}

func parseFile(t *testing.T, in string) data.NodeFile {
	t.Helper()

	var f data.NodeFile
	if err := yaml.Unmarshal([]byte(in), &f); err != nil {
		t.Fatalf("error parsing: %v", err)
	}

	return f
}

func findPoint(t *testing.T, points data.Points, typ, key string) data.Point {
	t.Helper()

	p, ok := points.Find(typ, key)
	if !ok {
		t.Fatalf("point %v:%v not found in %v", typ, key, points)
	}

	return p
}

func TestNodeYAMLValueKinds(t *testing.T) {
	f := parseFile(t, `
nodes:
  - modbus:
      description: Modbus sensors
      port: "9600"
      baud: 9600
      scale: 0.5
      debug: true
      off: false
      blank:
`)

	if len(f.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %v", len(f.Nodes))
	}

	n := f.Nodes[0]
	if n.Type != "modbus" {
		t.Fatalf("expected type modbus, got %v", n.Type)
	}

	if got := findPoint(t, n.Points, "description", "").Txt(); got != "Modbus sensors" {
		t.Errorf("description: got %q", got)
	}

	// a quoted number is text, an unquoted one is a value
	port := findPoint(t, n.Points, "port", "")
	if port.Txt() != "9600" {
		t.Errorf("port should be text, got %v", port)
	}
	if port.Val() != 0 {
		t.Errorf("port should have no numeric value, got %v", port.Val())
	}

	baud := findPoint(t, n.Points, "baud", "")
	if baud.Val() != 9600 {
		t.Errorf("baud: got %v", baud.Val())
	}
	if baud.Txt() != "" {
		t.Errorf("baud should not be text, got %q", baud.Txt())
	}

	if got := findPoint(t, n.Points, "scale", "").Val(); got != 0.5 {
		t.Errorf("scale: got %v", got)
	}

	if got := findPoint(t, n.Points, "debug", "").Val(); got != 1 {
		t.Errorf("debug: got %v", got)
	}

	if got := findPoint(t, n.Points, "off", "").Val(); got != 0 {
		t.Errorf("off: got %v", got)
	}

	blank := findPoint(t, n.Points, "blank", "")
	if blank.Val() != 0 || blank.Txt() != "" || len(blank.Data) != 0 {
		t.Errorf("blank should carry no value, got %v", blank)
	}
}

func TestNodeYAMLKeyedPoints(t *testing.T) {
	f := parseFile(t, `
nodes:
  - metrics:
      metricSysCPUFreq:
        cpu0: 1400
        cpu1: 1600
      tag: [alpha, beta]
`)

	n := f.Nodes[0]

	if got := findPoint(t, n.Points, "metricSysCPUFreq", "cpu0").Val(); got != 1400 {
		t.Errorf("cpu0: got %v", got)
	}
	if got := findPoint(t, n.Points, "metricSysCPUFreq", "cpu1").Val(); got != 1600 {
		t.Errorf("cpu1: got %v", got)
	}

	// a sequence is keyed by index
	if got := findPoint(t, n.Points, "tag", "0").Txt(); got != "alpha" {
		t.Errorf("tag 0: got %q", got)
	}
	if got := findPoint(t, n.Points, "tag", "1").Txt(); got != "beta" {
		t.Errorf("tag 1: got %q", got)
	}
}

func TestNodeYAMLChildrenAndIDs(t *testing.T) {
	f := parseFile(t, `
nodes:
  - group:
      id: sensors
      description: Sensors
      children:
        - modbus:
            description: Modbus sensors
            children:
              - modbusIo:
                  description: Tank level
  - variable:
      parent: Sensors
      description: Tank level
`)

	if len(f.Nodes) != 2 {
		t.Fatalf("expected 2 top level nodes, got %v", len(f.Nodes))
	}

	group := f.Nodes[0]
	if group.ID != "sensors" {
		t.Errorf("id: got %q", group.ID)
	}

	if len(group.Children) != 1 {
		t.Fatalf("expected 1 child, got %v", len(group.Children))
	}

	modbus := group.Children[0]
	if modbus.Type != "modbus" {
		t.Errorf("child type: got %v", modbus.Type)
	}

	if len(modbus.Children) != 1 || modbus.Children[0].Type != "modbusIo" {
		t.Fatalf("expected a modbusIo grandchild, got %v", modbus.Children)
	}

	v := f.Nodes[1]
	if v.Parent != "Sensors" {
		t.Errorf("parent: got %q", v.Parent)
	}
}

func TestNodeYAMLLongForm(t *testing.T) {
	f := parseFile(t, `
nodes:
  - user:
      firstName: Admin
      points:
        - type: id
          text: not-a-node-id
        - type: count
          value: 3
          dataType: 2
      edgePoints:
        - type: role
          text: admin
`)

	n := f.Nodes[0]

	if got := findPoint(t, n.Points, "firstName", "").Txt(); got != "Admin" {
		t.Errorf("firstName: got %q", got)
	}

	// a point whose type collides with a reserved key still round trips
	if got := findPoint(t, n.Points, "id", "").Txt(); got != "not-a-node-id" {
		t.Errorf("id point: got %q", got)
	}

	count := findPoint(t, n.Points, "count", "")
	if count.DataType != data.PointDataTypeInt || count.Val() != 3 {
		t.Errorf("count should be an int point, got %v", count)
	}

	if got := findPoint(t, n.EdgePoints, "role", "").Txt(); got != "admin" {
		t.Errorf("role: got %q", got)
	}
}

func TestNodeYAMLOldFormatRejected(t *testing.T) {
	var f data.NodeFile
	err := yaml.Unmarshal([]byte(`
nodes:
  - id: inst1
    type: device
    points:
      - type: description
        text: hi
`), &f)

	if err == nil {
		t.Fatal("expected an error for a file in the old format")
	}

	if !strings.Contains(err.Error(), "old export format") {
		t.Errorf("error should explain the format change, got: %v", err)
	}
}

func TestNodeYAMLMarshalShortForm(t *testing.T) {
	n := data.NodeYAML{
		Type: "modbus",
		ID:   "sensor-modbus",
		Points: data.Points{
			data.NewPointString("description", "", "Modbus sensors"),
			data.NewPointFloat("baud", "", 9600),
			data.NewPointString("port", "", "9600"),
			data.NewPointFloat("scale", "", 0.5),
		},
	}

	out, err := yaml.Marshal(data.NodeFile{Nodes: []data.NodeYAML{n}})
	if err != nil {
		t.Fatal(err)
	}

	got := string(out)

	for _, want := range []string{
		"- modbus:",
		"id: sensor-modbus",
		"baud: 9600",
		`port: "9600"`,
		"scale: 0.5",
		"description: Modbus sensors",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in:\n%v", want, got)
		}
	}

	// points are ordered by type, so an unchanged tree exports identically
	if i, j := strings.Index(got, "baud:"), strings.Index(got, "description:"); i > j {
		t.Errorf("points should be sorted by type:\n%v", got)
	}
}

func TestNodeYAMLRoundTrip(t *testing.T) {
	orig := data.NodeFile{
		APIVersion: data.NodeFileAPIVersion,
		Nodes: []data.NodeYAML{
			{
				Type: "group",
				ID:   "1a2b",
				Points: data.Points{
					data.NewPointString("description", "", "Sensors"),
				},
				Children: []data.NodeYAML{
					{
						Type: "modbus",
						ID:   "3c4d",
						Points: data.Points{
							data.NewPointString("description", "", "Modbus sensors"),
							data.NewPointFloat("baud", "", 9600),
							data.NewPointString("port", "", "502"),
							data.NewPointFloat("scale", "", 0.125),
							data.NewPointFloat("metric", "cpu0", 1400),
							data.NewPointFloat("metric", "cpu1", 1600),
							data.NewPointString("tag", "0", "alpha"),
							data.NewPointString("tag", "1", "beta"),
							data.NewPointInt("count", "", 3),
							{Type: "blob", DataType: data.PointDataTypeJSON, Data: []byte(`{"a":1}`)},
							{Type: "retired", Tombstone: 1},
							func() data.Point {
								p := data.NewPointString("stamped", "", "hi")
								p.Origin = "somewhere"
								return p
							}(),
						},
						EdgePoints: data.Points{
							data.NewPointString("role", "", "admin"),
						},
					},
				},
			},
		},
	}

	out, err := yaml.Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}

	var back data.NodeFile
	if err := yaml.Unmarshal(out, &back); err != nil {
		t.Fatalf("error parsing what we wrote: %v\n%v", err, string(out))
	}

	out2, err := yaml.Marshal(back)
	if err != nil {
		t.Fatal(err)
	}

	if string(out) != string(out2) {
		t.Errorf("round trip changed the file:\n--- first ---\n%v\n--- second ---\n%v", string(out), string(out2))
	}

	if back.APIVersion != data.NodeFileAPIVersion {
		t.Errorf("apiVersion: got %v", back.APIVersion)
	}

	if len(back.Nodes) != 1 || len(back.Nodes[0].Children) != 1 {
		t.Fatalf("tree shape changed: %v", back.Nodes)
	}

	origPoints := orig.Nodes[0].Children[0].Points
	backPoints := back.Nodes[0].Children[0].Points

	if len(origPoints) != len(backPoints) {
		t.Fatalf("expected %v points, got %v:\n%v", len(origPoints), len(backPoints), string(out))
	}

	for _, want := range origPoints {
		got := findPoint(t, backPoints, want.Type, want.Key)
		if got.DataType != want.DataType {
			t.Errorf("%v: data type %v, want %v", want.Type, got.DataType, want.DataType)
		}
		if got.Val() != want.Val() || got.Txt() != want.Txt() {
			t.Errorf("%v: got %v, want %v", want.Type, got, want)
		}
		if got.Tombstone != want.Tombstone {
			t.Errorf("%v: tombstone %v, want %v", want.Type, got.Tombstone, want.Tombstone)
		}
		if got.Origin != want.Origin {
			t.Errorf("%v: origin %q, want %q", want.Type, got.Origin, want.Origin)
		}
		if string(got.Data) != string(want.Data) {
			t.Errorf("%v: data %q, want %q", want.Type, got.Data, want.Data)
		}
	}

	if got := findPoint(t, back.Nodes[0].Children[0].EdgePoints, "role", "").Txt(); got != "admin" {
		t.Errorf("edge point role: got %q", got)
	}
}

func TestNodeYAMLFixtureParses(t *testing.T) {
	// the fixture the import and provisioning tests share
	b := readFixture(t)

	var f data.NodeFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		t.Fatalf("error parsing the shared fixture: %v", err)
	}

	if len(f.Nodes) < 4 {
		t.Fatalf("expected the fixture to carry several nodes, got %v", len(f.Nodes))
	}
}
