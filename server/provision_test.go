package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// testServerNoProvisioning starts a server with provisioning turned off, for
// the tests that drive a provisioner themselves. Two provisioners applying the
// same sources at once would race to create the same nodes.
func testServerNoProvisioning(t *testing.T) (*nats.Conn, data.NodeEdge, func()) {
	t.Helper()

	opts := TestServerOptions
	opts.ProvisioningDisable = true

	nc, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	return nc, root, stop
}

// readFixture returns the tree fixture shared with the data and client tests.
func readFixture(t *testing.T) []byte {
	t.Helper()

	b, err := os.ReadFile("../testdata/tree.yaml")
	if err != nil {
		t.Fatal("Error reading fixture: ", err)
	}

	return b
}

// writeFile puts a provisioning file in a directory.
func writeFile(t *testing.T, dir, name string, contents []byte) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), contents, 0644); err != nil {
		t.Fatal("Error writing provisioning file: ", err)
	}
}

// findByDesc returns the one node below root described the given way.
func findByDesc(t *testing.T, nc *nats.Conn, rootID, desc string) data.NodeEdge {
	t.Helper()

	nodes := findAllByDesc(t, nc, rootID, desc)

	if len(nodes) != 1 {
		t.Fatalf("expected exactly one node described %q, found %v", desc, len(nodes))
	}

	return nodes[0]
}

func findAllByDesc(t *testing.T, nc *nats.Conn, rootID, desc string) []data.NodeEdge {
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

	return found
}

// stateNodes returns the provisioningFile nodes recording what was applied.
func stateNodes(t *testing.T, nc *nats.Conn) []data.NodeEdge {
	t.Helper()

	root, err := client.GetRootNode(nc)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	provisioning, err := client.GetNodes(nc, root.ID, "all", data.NodeTypeProvisioning, false)
	if err != nil {
		t.Fatal("Error getting provisioning node: ", err)
	}

	if len(provisioning) < 1 {
		return nil
	}

	nodes, err := client.GetNodes(nc, provisioning[0].ID, "all", data.NodeTypeProvisioningFile, false)
	if err != nil {
		t.Fatal("Error getting provisioning state: ", err)
	}

	return nodes
}

func TestProvisionFromDirectory(t *testing.T) {
	nc, root, stop := testServerNoProvisioning(t)

	defer stop()

	dir := t.TempDir()
	writeFile(t, dir, "10-tree.yaml", readFixture(t))

	p := &provisioner{nc: nc, dir: dir}

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning: ", err)
	}

	// the tree the file describes is there
	sensors := findByDesc(t, nc, root.ID, "Sensors")
	if sensors.Parent != root.ID {
		t.Error("Sensors should be a child of the root node")
	}

	// including the entries that find their parent by description
	tankFarm := findByDesc(t, nc, root.ID, "Tank farm")
	modbus := findByDesc(t, nc, root.ID, "Modbus sensors")

	if modbus.Parent != tankFarm.ID {
		t.Error("Modbus sensors should attach under Tank farm")
	}

	// edge points come through too, which is the one part of a file that is
	// not a point on the node itself
	user := findByDesc(t, nc, root.ID, "admin@example.com")
	if role, ok := user.EdgePoints.Find(data.PointTypeRole, ""); !ok || role.Txt() != "admin" {
		t.Errorf("the user's role edge point should have been applied, got %v", role)
	}

	// a reference resolves to the node it names
	variable := findByDesc(t, nc, root.ID, "Tank level")
	condition := findByDesc(t, nc, root.ID, "Level below 10")

	if ref, ok := condition.Points.Find(data.PointTypeNodeID, ""); !ok || ref.Txt() != variable.ID {
		t.Errorf("nodeID should resolve to the variable: got %v, want %v", ref.Txt(), variable.ID)
	}

	// and the state node records what was applied
	states := stateNodes(t, nc)
	if len(states) != 1 {
		t.Fatalf("expected 1 state node, got %v", len(states))
	}

	desc, _ := states[0].Points.Text(data.PointTypeDescription, "")
	if desc != "10-tree.yaml" {
		t.Errorf("state node should be named for the file, got %q", desc)
	}

	hash, _ := states[0].Points.Text(data.PointTypeHash, "")
	if hash == "" {
		t.Error("state node should record a hash")
	}

	if errText, _ := states[0].Points.Text(data.PointTypeError, ""); errText != "" {
		t.Errorf("unexpected error recorded: %v", errText)
	}

	// a second pass over an unchanged file creates nothing
	if err := p.run(); err != nil {
		t.Fatal("Error provisioning a second time: ", err)
	}

	if got := findAllByDesc(t, nc, root.ID, "Sensors"); len(got) != 1 {
		t.Fatalf("a second pass should not duplicate anything, found %v Sensors nodes", len(got))
	}
}

func TestProvisionAppliesChanges(t *testing.T) {
	nc, root, stop := testServerNoProvisioning(t)

	defer stop()

	dir := t.TempDir()
	writeFile(t, dir, "10-tree.yaml", readFixture(t))

	p := &provisioner{nc: nc, dir: dir}

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning: ", err)
	}

	// change one point in the file
	writeFile(t, dir, "10-tree.yaml", []byte(`
nodes:
  - modbus:
      parent: Tank farm
      description: Modbus sensors
      baud: 115200
`))

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning after a change: ", err)
	}

	modbus := findByDesc(t, nc, root.ID, "Modbus sensors")
	if baud, ok := modbus.Points.Find("baud", ""); !ok || baud.Val() != 115200 {
		t.Errorf("baud should have been updated, got %v", baud)
	}
}

func TestProvisionRemovedFile(t *testing.T) {
	nc, root, stop := testServerNoProvisioning(t)

	defer stop()

	dir := t.TempDir()
	writeFile(t, dir, "10-tree.yaml", readFixture(t))

	p := &provisioner{nc: nc, dir: dir}

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning: ", err)
	}

	if err := os.Remove(filepath.Join(dir, "10-tree.yaml")); err != nil {
		t.Fatal("Error removing file: ", err)
	}

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning after removing the file: ", err)
	}

	// the nodes stay, since provisioning does not own them
	findByDesc(t, nc, root.ID, "Sensors")

	// and the state goes
	if states := stateNodes(t, nc); len(states) != 0 {
		t.Errorf("expected the state node to be removed, got %v", len(states))
	}
}

func TestProvisionIsolatesBadFiles(t *testing.T) {
	nc, root, stop := testServerNoProvisioning(t)

	defer stop()

	dir := t.TempDir()
	writeFile(t, dir, "10-bad.yaml", []byte("nodes: [this is not a node]\n"))
	writeFile(t, dir, "20-good.yaml", []byte(`
nodes:
  - group:
      description: Still applied
`))

	p := &provisioner{nc: nc, dir: dir}

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning: ", err)
	}

	// the good file still applied
	findByDesc(t, nc, root.ID, "Still applied")

	// and the bad one recorded its error
	var badState data.NodeEdge

	for _, s := range stateNodes(t, nc) {
		if desc, _ := s.Points.Text(data.PointTypeDescription, ""); desc == "10-bad.yaml" {
			badState = s
		}
	}

	if badState.ID == "" {
		t.Fatal("expected a state node for the bad file")
	}

	if errText, _ := badState.Points.Text(data.PointTypeError, ""); errText == "" {
		t.Error("the bad file should have recorded an error")
	}
}

func TestProvisionFromTree(t *testing.T) {
	nc, root, stop := testServerNoProvisioning(t)

	defer stop()

	p := &provisioner{nc: nc}

	// a file uploaded through the UI is a file node under the provisioning node
	node, err := p.provisioningNode(true)
	if err != nil {
		t.Fatal("Error creating provisioning node: ", err)
	}

	fileID := uuid.New().String()

	err = client.SendNode(nc, data.NodeEdge{
		ID:     fileID,
		Type:   data.NodeTypeFile,
		Parent: node,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "", "sensors.yaml"),
			data.NewPointString(data.PointTypeData, "", string(readFixture(t))),
			data.NewPointFloat(data.PointTypeCreated, "", float64(time.Now().Unix())),
		},
	}, "test")

	if err != nil {
		t.Fatal("Error sending file node: ", err)
	}

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning: ", err)
	}

	findByDesc(t, nc, root.ID, "Sensors")

	// the status lives on the file node itself
	nodes, err := client.GetNodes(nc, "all", fileID, "", false)
	if err != nil {
		t.Fatal("Error getting file node: ", err)
	}

	hash, _ := nodes[0].Points.Text(data.PointTypeProvisionHash, "")
	if hash == "" {
		t.Error("the file node should record what was applied")
	}

	if errText, _ := nodes[0].Points.Text(data.PointTypeError, ""); errText != "" {
		t.Errorf("unexpected error recorded: %v", errText)
	}

	// and no second node appears beside it
	if states := stateNodes(t, nc); len(states) != 0 {
		t.Errorf("a file node keeps its own state, got %v state nodes", len(states))
	}

	// applying again does nothing
	if err := p.run(); err != nil {
		t.Fatal("Error provisioning a second time: ", err)
	}

	if got := findAllByDesc(t, nc, root.ID, "Sensors"); len(got) != 1 {
		t.Errorf("a second pass should not duplicate anything, found %v", len(got))
	}
}

// A file node is added first and its contents arrive when someone uploads
// them, so provisioning waits for the upload rather than recording a pass
// against an empty file.
func TestProvisionWaitsForUpload(t *testing.T) {
	nc, root, stop := testServerNoProvisioning(t)

	defer stop()

	p := &provisioner{nc: nc}

	node, err := p.provisioningNode(true)
	if err != nil {
		t.Fatal("Error creating provisioning node: ", err)
	}

	fileID := uuid.New().String()

	err = client.SendNode(nc, data.NodeEdge{
		ID:     fileID,
		Type:   data.NodeTypeFile,
		Parent: node,
		Points: data.Points{
			data.NewPointFloat(data.PointTypeCreated, "", float64(time.Now().Unix())),
		},
	}, "test")

	if err != nil {
		t.Fatal("Error sending file node: ", err)
	}

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning: ", err)
	}

	nodes, err := client.GetNodes(nc, "all", fileID, "", false)
	if err != nil {
		t.Fatal("Error getting file node: ", err)
	}

	if hash, _ := nodes[0].Points.Text(data.PointTypeProvisionHash, ""); hash != "" {
		t.Error("a file with nothing uploaded should not be recorded as provisioned")
	}

	if errText, _ := nodes[0].Points.Text(data.PointTypeError, ""); errText != "" {
		t.Errorf("a file with nothing uploaded should not record an error, got %v", errText)
	}

	// once the contents arrive, the file applies
	err = client.SendNodePoints(nc, fileID, data.Points{
		data.NewPointString(data.PointTypeName, "", "sensors.yaml"),
		data.NewPointString(data.PointTypeData, "", string(readFixture(t))),
	}, true)

	if err != nil {
		t.Fatal("Error uploading file contents: ", err)
	}

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning after upload: ", err)
	}

	findByDesc(t, nc, root.ID, "Sensors")

	nodes, err = client.GetNodes(nc, "all", fileID, "", false)
	if err != nil {
		t.Fatal("Error getting file node: ", err)
	}

	if hash, _ := nodes[0].Points.Text(data.PointTypeProvisionHash, ""); hash == "" {
		t.Error("the file node should record what was applied")
	}
}

func TestProvisionDirectoryBeforeTree(t *testing.T) {
	nc, root, stop := testServerNoProvisioning(t)

	defer stop()

	dir := t.TempDir()
	writeFile(t, dir, "10-groups.yaml", []byte(`
nodes:
  - group:
      description: Shipped
`))

	p := &provisioner{nc: nc, dir: dir}

	node, err := p.provisioningNode(true)
	if err != nil {
		t.Fatal("Error creating provisioning node: ", err)
	}

	// an uploaded file attaches to a group the shipped file creates, which
	// only works because files on disk apply first
	err = client.SendNode(nc, data.NodeEdge{
		ID:     uuid.New().String(),
		Type:   data.NodeTypeFile,
		Parent: node,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "", "extra.yaml"),
			data.NewPointString(data.PointTypeData, "", `
nodes:
  - variable:
      parent: Shipped
      description: Added later
`),
			data.NewPointFloat(data.PointTypeCreated, "", float64(time.Now().Unix())),
		},
	}, "test")

	if err != nil {
		t.Fatal("Error sending file node: ", err)
	}

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning: ", err)
	}

	shipped := findByDesc(t, nc, root.ID, "Shipped")
	added := findByDesc(t, nc, root.ID, "Added later")

	if added.Parent != shipped.ID {
		t.Error("the uploaded file should attach under the group the shipped file created")
	}
}

func TestProvisionFileNodesInCreatedOrder(t *testing.T) {
	nc, root, stop := testServerNoProvisioning(t)

	defer stop()

	p := &provisioner{nc: nc}

	node, err := p.provisioningNode(true)
	if err != nil {
		t.Fatal("Error creating provisioning node: ", err)
	}

	now := time.Now()

	// the older file creates the group, the newer one attaches to it, and the
	// names are in the opposite order so that only creation time can explain
	// the result
	send := func(desc, contents string, created time.Time) {
		t.Helper()

		err := client.SendNode(nc, data.NodeEdge{
			ID:     uuid.New().String(),
			Type:   data.NodeTypeFile,
			Parent: node,
			Points: data.Points{
				data.NewPointString(data.PointTypeDescription, "", desc),
				data.NewPointString(data.PointTypeData, "", contents),
				data.NewPointFloat(data.PointTypeCreated, "", float64(created.Unix())),
			},
		}, "test")

		if err != nil {
			t.Fatal("Error sending file node: ", err)
		}
	}

	send("z-first.yaml", `
nodes:
  - group:
      description: Uploaded group
`, now.Add(-time.Hour))

	send("a-second.yaml", `
nodes:
  - variable:
      parent: Uploaded group
      description: Uploaded variable
`, now)

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning: ", err)
	}

	group := findByDesc(t, nc, root.ID, "Uploaded group")
	variable := findByDesc(t, nc, root.ID, "Uploaded variable")

	if variable.Parent != group.ID {
		t.Error("the newer file should attach to what the older one created")
	}
}

func TestProvisionDuplicateNames(t *testing.T) {
	nc, root, stop := testServerNoProvisioning(t)

	defer stop()

	dir := t.TempDir()
	writeFile(t, dir, "shared.yaml", []byte(`
nodes:
  - group:
      description: From disk
`))

	p := &provisioner{nc: nc, dir: dir}

	node, err := p.provisioningNode(true)
	if err != nil {
		t.Fatal("Error creating provisioning node: ", err)
	}

	fileID := uuid.New().String()

	err = client.SendNode(nc, data.NodeEdge{
		ID:     fileID,
		Type:   data.NodeTypeFile,
		Parent: node,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "", "shared.yaml"),
			data.NewPointString(data.PointTypeData, "", `
nodes:
  - group:
      description: From the tree
`),
			data.NewPointFloat(data.PointTypeCreated, "", float64(time.Now().Unix())),
		},
	}, "test")

	if err != nil {
		t.Fatal("Error sending file node: ", err)
	}

	if err := p.run(); err != nil {
		t.Fatal("Error provisioning: ", err)
	}

	// the first source with the name applied
	findByDesc(t, nc, root.ID, "From disk")

	// the second did not, and said why
	if got := findAllByDesc(t, nc, root.ID, "From the tree"); len(got) != 0 {
		t.Error("the second source sharing a name should not have been applied")
	}

	nodes, err := client.GetNodes(nc, "all", fileID, "", false)
	if err != nil {
		t.Fatal("Error getting file node: ", err)
	}

	errText, _ := nodes[0].Points.Text(data.PointTypeError, "")
	if !strings.Contains(errText, "shared.yaml") {
		t.Errorf("the error should name the file, got %q", errText)
	}
}

// waitFor polls until check passes or the deadline expires, which is how these
// tests wait for a pass the server decided to run on its own.
func waitFor(t *testing.T, what string, check func() bool) {
	t.Helper()

	timeout := time.After(time.Second * 10)

	for {
		if check() {
			return
		}

		select {
		case <-timeout:
			t.Fatalf("timeout waiting for %v", what)
		case <-time.After(time.Millisecond * 50):
		}
	}
}

// testServerProvisioning starts a server that provisions from dir. The rescan
// interval is long on purpose: a test that passes here passed because the
// directory watch or the tree subscription noticed, not because a timer came
// around.
func testServerProvisioning(t *testing.T, dir string) (*nats.Conn, data.NodeEdge, func()) {
	t.Helper()

	opts := TestServerOptions
	opts.ProvisioningDir = dir
	opts.ProvisioningInterval = time.Minute * 10

	nc, root, stop, err := TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	return nc, root, stop
}

func TestProvisionAtStartup(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "10-tree.yaml", readFixture(t))

	nc, root, stop := testServerProvisioning(t, dir)
	defer stop()

	waitFor(t, "the file to be applied", func() bool {
		return len(findAllByDesc(t, nc, root.ID, "Modbus sensors")) == 1
	})
}

func TestProvisionWatchesTheDirectory(t *testing.T) {
	dir := t.TempDir()

	nc, root, stop := testServerProvisioning(t, dir)
	defer stop()

	// a file that appears after the server is up is picked up by the watch
	writeFile(t, dir, "10-tree.yaml", []byte(`
nodes:
  - group:
      description: Appeared later
`))

	waitFor(t, "the new file to be applied", func() bool {
		return len(findAllByDesc(t, nc, root.ID, "Appeared later")) == 1
	})

	// as is a change to it
	writeFile(t, dir, "10-tree.yaml", []byte(`
nodes:
  - group:
      description: Appeared later
      children:
        - variable:
            description: Added by an edit
`))

	waitFor(t, "the edit to be applied", func() bool {
		return len(findAllByDesc(t, nc, root.ID, "Added by an edit")) == 1
	})
}

func TestProvisionWatchesFileNodes(t *testing.T) {
	nc, root, stop := testServerProvisioning(t, t.TempDir())
	defer stop()

	p := &provisioner{nc: nc}

	node, err := p.provisioningNode(true)
	if err != nil {
		t.Fatal("Error creating provisioning node: ", err)
	}

	// uploading a file through the UI is sending a file node
	err = client.SendNode(nc, data.NodeEdge{
		ID:     uuid.New().String(),
		Type:   data.NodeTypeFile,
		Parent: node,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "", "uploaded.yaml"),
			data.NewPointString(data.PointTypeData, "", `
nodes:
  - group:
      description: Uploaded through the UI
`),
		},
	}, "test")

	if err != nil {
		t.Fatal("Error sending file node: ", err)
	}

	waitFor(t, "the uploaded file to be applied", func() bool {
		return len(findAllByDesc(t, nc, root.ID, "Uploaded through the UI")) == 1
	})
}

func TestProvisionSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "10-tree.yaml", readFixture(t))

	nc, root, stop := testServerProvisioning(t, dir)

	waitFor(t, "the file to be applied", func() bool {
		return len(findAllByDesc(t, nc, root.ID, "Modbus sensors")) == 1
	})

	stop()

	// the same store and the same directory: nothing should be created twice
	opts := TestServerOptions
	opts.ProvisioningDir = dir
	opts.ProvisioningInterval = time.Minute * 10

	nc2, root2, stop2, err := TestServerOptsKeepStore(opts)
	if err != nil {
		t.Fatal("Error restarting test server: ", err)
	}

	defer stop2()

	// give the start-up pass time to do the wrong thing if it is going to
	time.Sleep(time.Second)

	for _, desc := range []string{"Sensors", "Tank farm", "Modbus sensors", "Tank level"} {
		if got := findAllByDesc(t, nc2, root2.ID, desc); len(got) != 1 {
			t.Errorf("expected exactly one %q after a restart, found %v", desc, len(got))
		}
	}
}
