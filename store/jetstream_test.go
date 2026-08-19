package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/data"
)

func newTestNatsServer(t *testing.T, storeDir string) (*server.Server, *nats.Conn) {
	t.Helper()

	opts := &server.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  storeDir,
		NoSigs:    true,
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatal("Error creating NATS server:", err)
	}

	ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("NATS server failed to start")
	}

	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		t.Fatal("Error connecting to NATS:", err)
	}

	return ns, nc
}

func newTestJsDb(t *testing.T) (*DbJetStream, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
	if err != nil {
		t.Fatal("Error creating temp dir:", err)
	}

	ns, nc := newTestNatsServer(t, tmpDir)

	db, err := NewJetStreamDb(nc, "", JsConfig{})
	if err != nil {
		t.Fatal("Error creating JetStream db:", err)
	}

	cleanup := func() {
		nc.Close()
		ns.Shutdown()
		_ = os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestDbJetStream(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	if rootID == "" {
		t.Fatal("Root ID is blank")
	}

	rns, err := db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node:", err)
	}

	if len(rns) < 1 {
		t.Fatal("No root nodes returned")
	}

	rn := rns[0]

	if rn.ID == "" {
		t.Fatal("Root node ID is blank")
	}

	// modify a point and see if it changes
	err = db.nodePoints(rootID, data.Points{data.NewPointString(data.PointTypeDescription, "", "root")})
	if err != nil {
		t.Fatal(err)
	}

	rns, err = db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node:", err)
	}

	rn = rns[0]

	if rn.Desc() != "root" {
		t.Fatal("Description should have been root, got:", rn.Desc())
	}

	// send an old point and verify it does not change
	err = db.nodePoints(rootID, data.Points{func() data.Point {
		p := data.NewPointString(data.PointTypeDescription, "", "root with old time")
		p.Time = time.Now().Add(-time.Hour)
		return p
	}()})
	if err != nil {
		t.Fatal(err)
	}

	rns, err = db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node:", err)
	}
	rn = rns[0]

	if rn.Desc() != "root" {
		t.Fatal("Description should have stayed root, got:", rn.Desc())
	}

	// verify default admin user got set
	children, err := db.getNodes(nil, rootID, "all", "", false)
	if err != nil {
		t.Fatal("children error:", err)
	}

	if len(children) < 1 {
		t.Fatal("did not return any children")
	}

	if children[0].Parent != rootID {
		t.Fatal("Parent not correct:", children[0].Parent)
	}

	// test getNodes API
	adminID := children[0].ID

	adminNodes, err := db.getNodes(nil, rootID, adminID, "", false)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}

	if adminNodes[0].Type != data.NodeTypeUser {
		t.Fatal("getNodes did not return right node type for user")
	}

	adminNodes, err = db.getNodes(nil, "all", adminID, "", false)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}

	rootNodes, err := db.getNodes(nil, "root", "all", "", false)
	if err != nil {
		t.Fatal("Error getting root nodes", err)
	}

	if len(rootNodes) < 1 {
		t.Fatal("did not return root nodes")
	}

	if rootNodes[0].ID != rootID {
		t.Fatal("root node ID is not correct")
	}

	// test edge points
	err = db.edgePoints(adminID, rootID, data.Points{data.NewPointString(data.PointTypeRole, "", data.PointValueRoleAdmin)})
	if err != nil {
		t.Fatal("Error sending edge points:", err)
	}

	adminNodes, err = db.getNodes(nil, rootID, adminID, "", false)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	p, ok := adminNodes[0].EdgePoints.Find(data.PointTypeRole, "")
	if !ok {
		t.Fatal("point not found")
	}
	if p.Txt() != data.PointValueRoleAdmin {
		t.Fatal("point does not have right value")
	}

	// try two children
	groupNodeID := uuid.New().String()

	err = db.edgePoints(groupNodeID, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeGroup),
	})
	if err != nil {
		t.Fatal("Error creating group edge", err)
	}

	children, err = db.getNodes(nil, rootID, "all", "", false)
	if err != nil {
		t.Fatal("children error:", err)
	}

	if len(children) < 2 {
		t.Fatal("did not return 2 children")
	}

	// verify getNodes with "all" works
	start := time.Now()
	adminNodes, err = db.getNodes(nil, "all", adminID, "", false)
	fmt.Println("getNodes time:", time.Since(start))
	if err != nil {
		t.Fatal("Error getting admin nodes with all specified:", err)
	}

	if adminNodes[0].Parent != rootID {
		t.Fatal("Parent ID is not correct")
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}
}

func TestDbJetStreamUserCheck(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	nodes, err := db.userCheck("admin", "admin")
	if err != nil {
		t.Fatal("userCheck returned error:", err)
	}

	if len(nodes) < 1 {
		t.Fatal("userCheck did not return nodes")
	}
}

// A user replicated up from a downstream instance can carry the same
// credentials as the upstream's own admin. Login must resolve to the user
// closest to the root, and must do so on every call.
func TestDbJetStreamUserCheckDuplicateCredentials(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	// a downstream device, holding a replicated user with the same
	// email and password as the admin created with the root node
	deviceID := uuid.New().String()
	mkTestNode(t, db, rootID, deviceID, data.NodeTypeDevice, "downstream")

	dupID := uuid.New().String()
	mkTestNode(t, db, deviceID, dupID, data.NodeTypeUser, "")
	err := db.nodePoints(dupID, data.Points{
		data.NewPointString(data.PointTypeEmail, "", "admin"),
		data.NewPointString(data.PointTypePass, "", "admin"),
	})
	if err != nil {
		t.Fatal("Error writing downstream user points:", err)
	}

	for i := 0; i < 10; i++ {
		users, err := db.userCheck("admin", "admin")
		if err != nil {
			t.Fatal("userCheck returned error:", err)
		}
		if len(users) != 2 {
			t.Fatal("expected both matching users, got:", len(users))
		}
		if users[0].ID == dupID {
			t.Fatal("userCheck picked the downstream user over the root user")
		}
		if users[1].ID != dupID {
			t.Fatal("downstream user not ordered after the root user")
		}
	}
}

func TestDbJetStreamUp(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	children, err := db.getNodes(nil, rootID, "all", "", false)
	if err != nil {
		t.Fatal("Error getting children")
	}

	if len(children) < 1 {
		t.Fatal("no children")
	}

	childID := children[0].ID

	ups, err := db.up(childID, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(ups) < 1 {
		t.Fatal("No ups for admin user")
	}

	if ups[0] != rootID {
		t.Fatal("ups, wrong ID:", ups[0])
	}

	// try to get ups of root node
	ups, err = db.up(rootID, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(ups) < 1 {
		t.Fatal("No ups for root node")
	}

	if ups[0] != "root" {
		t.Fatal("ups, wrong ID for root:", ups[0])
	}
}

// TestDbJetStreamRestart re-opens the store against existing streams
// after a full NATS server restart, then verifies config points are
// intact, and that a write and read work against the reloaded caches.
func TestDbJetStreamRestart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
	if err != nil {
		t.Fatal("Error creating temp dir:", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ns, nc := newTestNatsServer(t, tmpDir)

	db, err := NewJetStreamDb(nc, "", JsConfig{})
	if err != nil {
		t.Fatal("Error creating JetStream db:", err)
	}

	rootID := db.rootNodeID()

	// create a group node with a description
	groupID := uuid.New().String()
	err = db.edgePoints(groupID, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeGroup),
	})
	if err != nil {
		t.Fatal("Error creating group edge:", err)
	}

	err = db.nodePoints(groupID, data.Points{
		data.NewPointString(data.PointTypeDescription, "", "pump house"),
	})
	if err != nil {
		t.Fatal("Error writing group points:", err)
	}

	// full server restart on the same store directory
	nc.Close()
	ns.Shutdown()
	ns.WaitForShutdown()

	ns, nc = newTestNatsServer(t, tmpDir)
	defer func() {
		nc.Close()
		ns.Shutdown()
	}()

	db, err = NewJetStreamDb(nc, "", JsConfig{})
	if err != nil {
		t.Fatal("Error re-opening JetStream db:", err)
	}

	if db.rootNodeID() != rootID {
		t.Fatalf("root ID changed across restart: %v != %v",
			db.rootNodeID(), rootID)
	}

	// config points must be intact after the restart
	nodes, err := db.getNodes(nil, rootID, groupID, "", false)
	if err != nil {
		t.Fatal("Error getting group node:", err)
	}
	if len(nodes) < 1 {
		t.Fatal("group node not found after restart")
	}
	if nodes[0].Desc() != "pump house" {
		t.Fatal("group description lost after restart, got:", nodes[0].Desc())
	}

	// this was the cache poisoning bug: the first write after restart
	// seeded the cache and the config points vanished until the next
	// restart — write a point, then verify config is still there
	err = db.nodePoints(groupID, data.Points{
		data.NewPointFloat(data.PointTypeValue, "", 42),
	})
	if err != nil {
		t.Fatal("Error writing point after restart:", err)
	}

	nodes, err = db.getNodes(nil, rootID, groupID, "", false)
	if err != nil {
		t.Fatal("Error getting group node after write:", err)
	}
	if nodes[0].Desc() != "pump house" {
		t.Fatal("config points lost after first write, got:", nodes[0].Desc())
	}
	v, _ := nodes[0].Points.Value(data.PointTypeValue, "")
	if v != 42 {
		t.Fatal("new point not visible after write, got:", v)
	}

	// the fresh instance should have a single boundary-origin stream
	// inst_<rootID>_<rootID> holding everything
	_, err = db.js.Stream(context.Background(), streamName(rootID, rootID))
	if err != nil {
		t.Fatal("expected stream inst_<rootID>_<rootID> to exist:", err)
	}
}

// streamSubjectCount returns how many subjects matching filter exist in
// the named stream, 0 if the stream does not exist.
func streamSubjectCount(t *testing.T, db *DbJetStream, name, filter string) int {
	t.Helper()
	ctx := context.Background()

	s, err := db.js.Stream(ctx, name)
	if err != nil {
		return 0
	}
	info, err := s.Info(ctx, jetstream.WithSubjectFilter(filter))
	if err != nil {
		t.Fatal("Error getting stream info:", err)
	}
	return len(info.State.Subjects)
}

// mkTestNode creates a node with an edge and optional description.
func mkTestNode(t *testing.T, db *DbJetStream, parent, id, typ, desc string) {
	t.Helper()
	err := db.edgePoints(id, parent, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", typ),
	})
	if err != nil {
		t.Fatalf("Error creating %v edge: %v", id, err)
	}
	if desc != "" {
		err = db.nodePoints(id, data.Points{
			data.NewPointString(data.PointTypeDescription, "", desc),
		})
		if err != nil {
			t.Fatalf("Error writing %v points: %v", id, err)
		}
	}
}

func TestDbJetStreamCrossBoundaryMove(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	devX := uuid.New().String()
	devY := uuid.New().String()
	sensor := uuid.New().String()

	mkTestNode(t, db, rootID, devX, data.NodeTypeDevice, "device X")
	mkTestNode(t, db, rootID, devY, data.NodeTypeDevice, "device Y")
	mkTestNode(t, db, devX, sensor, data.NodeTypeVariable, "sensor 1")

	// capture the original description timestamp; it must survive the
	// move unchanged
	nodes, err := db.getNodes(nil, devX, sensor, "", false)
	if err != nil || len(nodes) < 1 {
		t.Fatal("Error getting sensor:", err)
	}
	descBefore, ok := nodes[0].Points.Find(data.PointTypeDescription, "")
	if !ok {
		t.Fatal("description point missing")
	}

	// sensor subjects live in device X's boundary stream
	xStream := streamName(devX, rootID)
	yStream := streamName(devY, rootID)
	if streamSubjectCount(t, db, xStream, "inst."+devX+"."+rootID+"."+sensor+".>") == 0 {
		t.Fatal("sensor subjects not in device X stream before move")
	}

	// move: add edge under Y, tombstone edge under X
	err = db.edgePoints(sensor, devY, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeVariable),
	})
	if err != nil {
		t.Fatal("Error adding new edge:", err)
	}
	err = db.edgePoints(sensor, devX, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 1),
	})
	if err != nil {
		t.Fatal("Error tombstoning old edge:", err)
	}

	// sensor subjects now live in device Y's stream only
	if streamSubjectCount(t, db, yStream, "inst."+devY+"."+rootID+"."+sensor+".>") == 0 {
		t.Fatal("sensor subjects not in device Y stream after move")
	}
	if n := streamSubjectCount(t, db, xStream, "inst."+devX+"."+rootID+"."+sensor+".>"); n != 0 {
		t.Fatal("sensor subjects remain in device X stream after move:", n)
	}

	// node reads back under Y with the original point timestamp
	nodes, err = db.getNodes(nil, devY, sensor, "", false)
	if err != nil || len(nodes) < 1 {
		t.Fatal("Error getting sensor under Y:", err)
	}
	descAfter, ok := nodes[0].Points.Find(data.PointTypeDescription, "")
	if !ok {
		t.Fatal("description point missing after move")
	}
	if !descAfter.Time.Equal(descBefore.Time) {
		t.Fatalf("description timestamp changed in move: %v != %v",
			descAfter.Time, descBefore.Time)
	}

	// the old edge is tombstoned, not gone
	all, err := db.getNodes(nil, "all", sensor, "", true)
	if err != nil {
		t.Fatal("Error getting all sensor edges:", err)
	}
	if len(all) != 2 {
		t.Fatal("expected 2 edges for moved node, got:", len(all))
	}
}

func TestDbJetStreamSubtreeMoveIntoBoundary(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	devX := uuid.New().String()
	group := uuid.New().String()
	sensor := uuid.New().String()

	mkTestNode(t, db, rootID, devX, data.NodeTypeDevice, "device X")
	mkTestNode(t, db, rootID, group, data.NodeTypeGroup, "pump group")
	mkTestNode(t, db, group, sensor, data.NodeTypeVariable, "pump sensor")

	rootStream := streamName(rootID, rootID)
	xStream := streamName(devX, rootID)

	// group and sensor start in the root boundary stream
	if streamSubjectCount(t, db, rootStream, "inst."+rootID+"."+rootID+"."+sensor+".>") == 0 {
		t.Fatal("sensor subjects not in root stream before move")
	}

	// move the group (and its subtree) under device X
	err := db.edgePoints(group, devX, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeGroup),
	})
	if err != nil {
		t.Fatal("Error adding group edge under device:", err)
	}
	err = db.edgePoints(group, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 1),
	})
	if err != nil {
		t.Fatal("Error tombstoning group root edge:", err)
	}

	// the whole subtree moved: group points, group->sensor edge, and
	// sensor points are all in device X's stream now
	for _, filter := range []string{
		"inst." + devX + "." + rootID + "." + group + ".p.>",
		"inst." + devX + "." + rootID + "." + group + ".ep." + sensor,
		"inst." + devX + "." + rootID + "." + sensor + ".p.>",
	} {
		if streamSubjectCount(t, db, xStream, filter) == 0 {
			t.Fatal("expected subjects in device stream after move:", filter)
		}
	}

	// and purged from the root stream (the tombstoned root->group edge
	// stays: it belongs to the root boundary)
	for _, filter := range []string{
		"inst." + rootID + "." + rootID + "." + group + ".>",
		"inst." + rootID + "." + rootID + "." + sensor + ".>",
	} {
		if n := streamSubjectCount(t, db, rootStream, filter); n != 0 {
			t.Fatal("subjects remain in root stream after move:", filter)
		}
	}

	// reads stay consistent through the move
	nodes, err := db.getNodes(nil, group, sensor, "", false)
	if err != nil || len(nodes) < 1 {
		t.Fatal("Error getting sensor after move:", err)
	}
	if nodes[0].Desc() != "pump sensor" {
		t.Fatal("sensor description lost in move:", nodes[0].Desc())
	}
}

// TestDbJetStreamDefaultRetention verifies streams get the default
// per-subject retention when none is configured.
func TestDbJetStreamDefaultRetention(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	s, err := db.js.Stream(context.Background(), streamName(rootID, rootID))
	if err != nil {
		t.Fatal("Error getting stream:", err)
	}
	info, err := s.Info(context.Background())
	if err != nil {
		t.Fatal("Error getting stream info:", err)
	}
	if info.Config.MaxMsgsPerSubject != defaultMaxMsgsPerSubject {
		t.Fatalf("default retention = %v, want %v",
			info.Config.MaxMsgsPerSubject, defaultMaxMsgsPerSubject)
	}
}

// TestRetentionDescription verifies the startup log describes the policy the
// store actually applies, for each of the three ways it resolves.
func TestRetentionDescription(t *testing.T) {
	tests := []struct {
		desc   string
		config int64
		exp    string
	}{
		{
			desc:   "unconfigured reports the default",
			config: 0,
			exp: "20000 points per subject (default); current state is " +
				"always preserved",
		},
		{
			desc:   "a configured limit is reported",
			config: 20000,
			exp:    "20000 points per subject; current state is always preserved",
		},
		{
			desc:   "unlimited is reported as such",
			config: -1,
			exp:    "unlimited points per subject",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db := &DbJetStream{cfg: JsConfig{MaxMsgsPerSubject: test.config}}

			if got := db.retentionDescription(); got != test.exp {
				t.Errorf("description = %q, want %q", got, test.exp)
			}

			// the description has to agree with what streams are given, or
			// the log tells the operator something that is not true
			want := test.config
			if want == 0 {
				want = defaultMaxMsgsPerSubject
			} else if want < 0 {
				want = 0 // JetStream spells unlimited as zero
			}

			if got := db.maxMsgsForStream(""); got != want {
				t.Errorf("stream limit = %v, want %v", got, want)
			}
		})
	}
}

// TestDbJetStreamReplicaPolicy verifies the store applies its storage
// policy -- retention and compression -- to replica streams it discovers
// (the sync pumps create them bare). Both describe how this instance uses
// its own disk, so a replica follows the local policy.
func TestDbJetStreamReplicaPolicy(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
	if err != nil {
		t.Fatal("Error creating temp dir:", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ns, nc := newTestNatsServer(t, tmpDir)
	defer func() {
		nc.Close()
		ns.Shutdown()
	}()

	db, err := NewJetStreamDb(nc, "", JsConfig{MaxMsgsPerSubject: 7})
	if err != nil {
		t.Fatal("Error creating JetStream db:", err)
	}

	// a bare replica stream, as a sync pump would create it
	ctx := context.Background()
	_, err = db.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamName("boundary-b", "origin-o"),
		Subjects: []string{"inst.boundary-b.origin-o.>"},
	})
	if err != nil {
		t.Fatal("Error creating bare replica stream:", err)
	}

	rm := db.runReplicaManager()
	defer rm.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for {
		s, err := db.js.Stream(ctx, streamName("boundary-b", "origin-o"))
		if err == nil {
			info, err := s.Info(ctx)
			if err == nil && info.Config.MaxMsgsPerSubject == 7 &&
				info.Config.Compression == jetstream.S2Compression {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("replica stream never received the local storage policy")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestCompressionResolution verifies the setting resolves to what streams are
// given, and that the startup log describes the same thing.
func TestCompressionResolution(t *testing.T) {
	tests := []struct {
		desc    string
		config  string
		exp     jetstream.StoreCompression
		expDesc string
	}{
		{
			desc:    "unconfigured compresses",
			config:  "",
			exp:     jetstream.S2Compression,
			expDesc: "s2 (default)",
		},
		{
			desc:    "s2 is explicit",
			config:  CompressionS2,
			exp:     jetstream.S2Compression,
			expDesc: "s2",
		},
		{
			desc:    "none turns it off",
			config:  CompressionNone,
			exp:     jetstream.NoCompression,
			expDesc: "none",
		},
		{
			// args rejects this, but SIOT is also used as a library, and
			// failing to start over a compression setting would be worse
			// than compressing
			desc:    "an unrecognized setting falls back and says so",
			config:  "gzip",
			exp:     jetstream.S2Compression,
			expDesc: `s2 (default; "gzip" is not a compression setting)`,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db := &DbJetStream{cfg: JsConfig{Compression: test.config}}

			if got := db.compressionForStream(""); got != test.exp {
				t.Errorf("compression = %v, want %v", got, test.exp)
			}

			if got := db.compressionDescription(); got != test.expDesc {
				t.Errorf("description = %q, want %q", got, test.expDesc)
			}
		})
	}
}

// TestDbJetStreamCompression verifies streams are created compressed, and
// that turning it off is honored.
func TestDbJetStreamCompression(t *testing.T) {
	tests := []struct {
		desc   string
		config string
		exp    jetstream.StoreCompression
	}{
		{"on by default", "", jetstream.S2Compression},
		{"off when asked", CompressionNone, jetstream.NoCompression},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
			if err != nil {
				t.Fatal("Error creating temp dir:", err)
			}

			defer func() { _ = os.RemoveAll(tmpDir) }()

			ns, nc := newTestNatsServer(t, tmpDir)
			defer func() {
				nc.Close()
				ns.Shutdown()
			}()

			db, err := NewJetStreamDb(nc, "", JsConfig{Compression: test.config})
			if err != nil {
				t.Fatal("Error creating JetStream db:", err)
			}

			rootID := db.rootNodeID()

			s, err := db.js.Stream(context.Background(),
				streamName(rootID, rootID))
			if err != nil {
				t.Fatal("Error getting stream:", err)
			}

			info, err := s.Info(context.Background())
			if err != nil {
				t.Fatal("Error getting stream info:", err)
			}

			if info.Config.Compression != test.exp {
				t.Errorf("compression = %v, want %v",
					info.Config.Compression, test.exp)
			}
		})
	}
}

// A stream that already holds data has to accept compression being turned on,
// and its existing messages have to stay readable afterward. This is what
// happens to every stream on an instance that upgrades into this default.
func TestDbJetStreamCompressionOnExistingStream(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
	if err != nil {
		t.Fatal("Error creating temp dir:", err)
	}

	defer func() { _ = os.RemoveAll(tmpDir) }()

	ns, nc := newTestNatsServer(t, tmpDir)
	defer func() {
		nc.Close()
		ns.Shutdown()
	}()

	ctx := context.Background()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal("Error creating JetStream context:", err)
	}

	const name = "inst_existing_existing"

	// a stream as an instance from before this default would have it
	_, err = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:              name,
		Subjects:          []string{"inst.existing.existing.>"},
		MaxMsgsPerSubject: 5000,
		Compression:       jetstream.NoCompression,
	})
	if err != nil {
		t.Fatal("Error creating stream:", err)
	}

	const count = 500

	for i := 0; i < count; i++ {
		subject := fmt.Sprintf("inst.existing.existing.node%v.p.value.0", i%10)
		if _, err := js.Publish(ctx, subject,
			[]byte(fmt.Sprintf("point %v", i))); err != nil {
			t.Fatal("Error publishing:", err)
		}
	}

	// turning compression on is what the store does the first time it
	// ensures or discovers the stream after an upgrade
	cfg := jetstream.StreamConfig{
		Name:              name,
		Subjects:          []string{"inst.existing.existing.>"},
		MaxMsgsPerSubject: 5000,
		Compression:       jetstream.S2Compression,
	}

	if _, err := js.UpdateStream(ctx, cfg); err != nil {
		t.Fatal("Error enabling compression on an existing stream:", err)
	}

	s, err := js.Stream(ctx, name)
	if err != nil {
		t.Fatal("Error getting stream:", err)
	}

	info, err := s.Info(ctx)
	if err != nil {
		t.Fatal("Error getting stream info:", err)
	}

	if info.Config.Compression != jetstream.S2Compression {
		t.Fatalf("compression = %v, want S2", info.Config.Compression)
	}

	if info.State.Msgs != count {
		t.Errorf("Expected %v messages after enabling compression, got %v",
			count, info.State.Msgs)
	}

	// messages written before the change still read back
	msg, err := s.GetLastMsgForSubject(ctx,
		"inst.existing.existing.node9.p.value.0")
	if err != nil {
		t.Fatal("Error reading a message written before compression:", err)
	}

	if len(msg.Data) == 0 {
		t.Error("Expected the message written before compression to have data")
	}

	t.Logf("stream holds %v msgs, %v bytes after enabling compression",
		info.State.Msgs, info.State.Bytes)
}

// TestDbJetStreamRetention verifies per-subject retention drops old
// messages of a frequently-written subject while preserving the tips
// of rarely-written subjects (config points).
func TestDbJetStreamRetention(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
	if err != nil {
		t.Fatal("Error creating temp dir:", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ns, nc := newTestNatsServer(t, tmpDir)
	defer func() {
		nc.Close()
		ns.Shutdown()
	}()

	db, err := NewJetStreamDb(nc, "", JsConfig{MaxMsgsPerSubject: 5})
	if err != nil {
		t.Fatal("Error creating JetStream db:", err)
	}

	rootID := db.rootNodeID()

	// description is written once (a rarely-written config subject)
	err = db.nodePoints(rootID, data.Points{
		data.NewPointString(data.PointTypeDescription, "", "retention test"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// value is written many times (a fast-changing subject)
	base := time.Now()
	for i := range 20 {
		p := data.NewPointFloat(data.PointTypeValue, "", float64(i))
		p.Time = base.Add(time.Duration(i) * time.Millisecond)
		err = db.nodePoints(rootID, data.Points{p})
		if err != nil {
			t.Fatal(err)
		}
	}

	ctx := context.Background()
	s, err := db.js.Stream(ctx, streamName(rootID, rootID))
	if err != nil {
		t.Fatal("Error getting stream:", err)
	}

	valueSubject := nodePointSubject(rootID, rootID, rootID, data.PointTypeValue, "0")
	info, err := s.Info(ctx, jetstream.WithSubjectFilter(valueSubject))
	if err != nil {
		t.Fatal("Error getting stream info:", err)
	}
	if n := info.State.Subjects[valueSubject]; n > 5 {
		t.Fatal("retention did not limit value subject, count:", n)
	}

	// tip of the fast subject is the latest write
	msg, err := s.GetLastMsgForSubject(ctx, valueSubject)
	if err != nil {
		t.Fatal("Error getting value tip:", err)
	}
	pts, err := data.DecodePoints(msg.Data)
	if err != nil || len(pts) < 1 {
		t.Fatal("Error decoding value tip:", err)
	}
	if pts[0].Val() != 19 {
		t.Fatal("value tip is not the latest write:", pts[0].Val())
	}

	// the rarely-written config subject is preserved
	descSubject := nodePointSubject(rootID, rootID, rootID, data.PointTypeDescription, "0")
	_, err = s.GetLastMsgForSubject(ctx, descSubject)
	if err != nil {
		t.Fatal("description tip lost under retention:", err)
	}
}

func TestDbJetStreamTombstoneDeleteUndelete(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()
	groupID := uuid.New().String()

	mkTestNode(t, db, rootID, groupID, data.NodeTypeGroup, "doomed group")

	// delete
	err := db.edgePoints(groupID, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 1),
	})
	if err != nil {
		t.Fatal("Error tombstoning group:", err)
	}

	nodes, err := db.getNodes(nil, rootID, groupID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Fatal("deleted node still returned without includeDel")
	}

	nodes, err = db.getNodes(nil, rootID, groupID, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatal("deleted node not returned with includeDel")
	}

	// undelete
	err = db.edgePoints(groupID, rootID, data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
	})
	if err != nil {
		t.Fatal("Error undeleting group:", err)
	}

	nodes, err = db.getNodes(nil, rootID, groupID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatal("undeleted node not returned")
	}
	if nodes[0].Desc() != "doomed group" {
		t.Fatal("node points lost across delete/undelete:", nodes[0].Desc())
	}
}

func TestDbJetStreamReset(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()
	groupID := uuid.New().String()
	mkTestNode(t, db, rootID, groupID, data.NodeTypeGroup, "temporary")

	err := db.reset()
	if err != nil {
		t.Fatal("Error resetting store:", err)
	}

	if db.rootNodeID() != rootID {
		t.Fatal("root ID not preserved across reset")
	}

	nodes, err := db.getNodes(nil, rootID, groupID, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Fatal("group survived reset")
	}

	// admin user is re-created
	users, err := db.userCheck("admin", "admin")
	if err != nil || len(users) < 1 {
		t.Fatal("admin user missing after reset:", err)
	}
}

func TestDbJetStreamEdgeOlderTimestamp(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()
	groupID := uuid.New().String()
	mkTestNode(t, db, rootID, groupID, data.NodeTypeGroup, "group")

	now := time.Now()

	p := data.NewPointString(data.PointTypeRole, "", "current")
	p.Time = now
	err := db.edgePoints(groupID, rootID, data.Points{p})
	if err != nil {
		t.Fatal(err)
	}

	// an older incoming edge point must not replace the tip
	old := data.NewPointString(data.PointTypeRole, "", "stale")
	old.Time = now.Add(-time.Hour)
	err = db.edgePoints(groupID, rootID, data.Points{old})
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := db.getNodes(nil, rootID, groupID, "", false)
	if err != nil || len(nodes) < 1 {
		t.Fatal("Error getting group:", err)
	}
	role, _ := nodes[0].EdgePoints.Text(data.PointTypeRole, "")
	if role != "current" {
		t.Fatal("older edge point replaced newer tip:", role)
	}
}

func TestDbJetStreamBatchPoints(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()

	now := time.Now()

	pts := data.Points{
		{Time: now, Type: data.PointTypeValue},
		{Time: now.Add(-time.Second), Type: data.PointTypeValue},
		{Time: now.Add(-time.Second * 2), Type: data.PointTypeValue},
	}

	err := db.nodePoints(rootID, pts)
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node:", err)
	}

	n := nodes[0]

	// After collapse, only one point with the latest time should remain
	var valuePoints data.Points
	for _, p := range n.Points {
		if p.Type == data.PointTypeValue {
			valuePoints = append(valuePoints, p)
		}
	}

	if len(valuePoints) != 1 {
		t.Fatal("Error, point did not get merged, got:", len(valuePoints))
	}

	if !valuePoints[0].Time.Equal(now) {
		t.Fatal("Point collapsing did not pick latest")
	}
}

// TestDbJetStreamReplicaRootNotAdopted covers an upstream that holds a
// downstream's replica stream. Every instance anchors its own tree with an
// edge whose parent is the virtual "root", and that edge rides along in the
// stream the downstream pushes up. The upstream must not load it as a root of
// its own: doing so gave it two roots, and it would then serve the
// downstream's root as its own and start clients for the downstream's nodes.
func TestDbJetStreamReplicaRootNotAdopted(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "siot-js-test-*")
	if err != nil {
		t.Fatal("Error creating temp dir:", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ns, nc := newTestNatsServer(t, tmpDir)

	db, err := NewJetStreamDb(nc, "", JsConfig{})
	if err != nil {
		t.Fatal("Error creating JetStream db:", err)
	}

	rootID := db.rootNodeID()

	// a downstream instance and one of its nodes
	downID := uuid.New().String()
	serialID := uuid.New().String()

	// the replica stream a sync pump creates on the upstream, carrying the
	// downstream's own origin stream for its root boundary
	ctx := context.Background()
	_, err = db.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamName(downID, downID),
		Subjects: []string{fmt.Sprintf("inst.%v.%v.>", downID, downID)},
	})
	if err != nil {
		t.Fatal("Error creating replica stream:", err)
	}

	// the downstream's root anchor, and a node below it
	anchor := data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeDevice),
	}
	_, err = db.js.Publish(ctx,
		edgePointSubject(downID, downID, "root", downID), anchor.Encode())
	if err != nil {
		t.Fatal("Error publishing downstream root anchor:", err)
	}

	child := data.Points{
		data.NewPointFloat(data.PointTypeTombstone, "", 0),
		data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeSerialDev),
	}
	_, err = db.js.Publish(ctx,
		edgePointSubject(downID, downID, downID, serialID), child.Encode())
	if err != nil {
		t.Fatal("Error publishing downstream child edge:", err)
	}

	// restart so the streams are loaded from scratch
	nc.Close()
	ns.Shutdown()
	ns.WaitForShutdown()

	ns, nc = newTestNatsServer(t, tmpDir)
	defer func() {
		nc.Close()
		ns.Shutdown()
	}()

	db, err = NewJetStreamDb(nc, "", JsConfig{})
	if err != nil {
		t.Fatal("Error re-opening JetStream db:", err)
	}

	if db.rootNodeID() != rootID {
		t.Fatalf("root ID changed across restart: %v != %v",
			db.rootNodeID(), rootID)
	}

	roots, err := db.getNodes(nil, "root", "all", "", false)
	if err != nil {
		t.Fatal("Error getting root node:", err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected exactly one root node, got %v", len(roots))
	}
	if roots[0].ID != rootID {
		t.Fatalf("root node is %v, expected this instance's root %v",
			roots[0].ID, rootID)
	}

	// the downstream's own nodes still replicate: only its root anchor is
	// dropped, so its subtree stays reachable below it
	kids, err := db.getNodes(nil, downID, "all", "", false)
	if err != nil {
		t.Fatal("Error getting downstream children:", err)
	}
	if len(kids) != 1 || kids[0].ID != serialID {
		t.Fatalf("expected the downstream's serialDev below it, got %v", kids)
	}
}

// TestDbJetStreamStrayRootEdgeIgnored covers a store that already holds a
// second root edge, as one written by an earlier version would. The root is
// the one recorded in meta, so such a store recovers on restart rather than
// needing the stray edge removed by hand.
func TestDbJetStreamStrayRootEdgeIgnored(t *testing.T) {
	db, cleanup := newTestJsDb(t)
	defer cleanup()

	rootID := db.rootNodeID()
	strayID := uuid.New().String()

	db.edgeCache.MergeEdgePoints("root", strayID, data.NodeTypeDevice, strayID,
		data.Points{
			data.NewPointFloat(data.PointTypeTombstone, "", 0),
			data.NewPointString(data.PointTypeNodeType, "", data.NodeTypeDevice),
		})

	roots, err := db.getNodes(nil, "root", "all", "", false)
	if err != nil {
		t.Fatal("Error getting root node:", err)
	}
	if len(roots) != 1 || roots[0].ID != rootID {
		t.Fatalf("expected only this instance's root %v, got %v", rootID, roots)
	}

	// asking for the stray by ID must not hand it back as a root either
	nodes, err := db.getNodes(nil, "root", strayID, "", false)
	if err != nil {
		t.Fatal("Error getting stray root node:", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("stray edge served as a root node: %v", nodes)
	}
}
