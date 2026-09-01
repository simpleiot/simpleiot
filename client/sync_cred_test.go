package client_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// credUpstream starts an upstream that requires the shared token, so a
// device with no token can only get in with a credential.
func credUpstream(t *testing.T) (*nats.Conn, data.NodeEdge, server.Options, func()) {
	t.Helper()

	opts := server.TestServerOptions2
	opts.AuthToken = "upstream-token"

	nc, root, stop, err := server.TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting upstream test server: ", err)
	}

	return nc, root, opts, stop
}

// devicePubKey reads the key the downstream generated for itself.
func devicePubKey(t *testing.T, ncD *nats.Conn) string {
	t.Helper()
	_, pubKey, err := client.GetDeviceKey(ncD)
	if err != nil {
		t.Fatal("Error getting device key: ", err)
	}
	return pubKey
}

// enrollDevice creates a device node with the given ID on the upstream and
// a credential for pubKey under it.
func enrollDevice(t *testing.T, ncU *nats.Conn, rootU data.NodeEdge, deviceID, credID, pubKey string) {
	t.Helper()

	dev := client.Device{ID: deviceID, Parent: rootU.ID, Description: "device " + deviceID}
	if err := client.SendNodeType(ncU, dev, "test"); err != nil {
		t.Fatal("Error creating device node: ", err)
	}

	cred := client.DeviceCred{ID: credID, Parent: deviceID, Description: "cred", PubKey: pubKey}
	if err := client.SendNodeType(ncU, cred, "test"); err != nil {
		t.Fatal("Error creating credential node: ", err)
	}
}

// startDeviceSync creates a sync node on the downstream with no token, so
// it authenticates with the device key.
func startDeviceSync(t *testing.T, ncD *nats.Conn, rootD data.NodeEdge, uri string) {
	t.Helper()

	sync := client.Sync{
		ID:          "sync-id",
		Parent:      rootD.ID,
		Description: "sync to up",
		URI:         uri,
	}

	if err := client.SendNodeType(ncD, sync, "test"); err != nil {
		t.Fatal("Error sending sync node: ", err)
	}
}

func getCred(t *testing.T, nc *nats.Conn, id string) client.DeviceCred {
	t.Helper()
	creds, err := client.GetNodesType[client.DeviceCred](nc, "all", id)
	if err != nil || len(creds) == 0 {
		t.Fatalf("credential %v not found: %v", id, err)
	}
	return creds[0]
}

// setCred changes a point on a credential the way an operator would.
func setCred(t *testing.T, nc *nats.Conn, id string, p data.Point) {
	t.Helper()
	p.Origin = "test"
	if err := client.SendNodePoint(nc, id, p, true); err != nil {
		t.Fatal(err)
	}
}

// TestSyncCredential runs the sync round trip with the device authenticating
// by credential rather than by token: push, pull, and the status the
// upstream keeps on the credential.
func TestSyncCredential(t *testing.T) {
	ncU, rootU, optsU, stopU := credUpstream(t)
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting downstream test server: ", err)
	}
	defer stopD()

	enrollDevice(t, ncU, rootU, rootD.ID, "cred-1", devicePubKey(t, ncD))
	startDeviceSync(t, ncD, rootD, optsU.NatsServer)

	waitFor(t, 15*time.Second, "credential not marked connected", func() bool {
		return getCred(t, ncU, "cred-1").Connected
	})
	if getCred(t, ncU, "cred-1").LastConnect == 0 {
		t.Fatal("lastConnect not recorded")
	}

	// the sync node shows the key it connects with
	syncs, err := client.GetNodesType[client.Sync](ncD, "all", "sync-id")
	if err != nil || len(syncs) == 0 || syncs[0].PubKey != devicePubKey(t, ncD) {
		t.Fatalf("sync node does not show the device key: %+v", syncs)
	}

	fmt.Println("**** push: description set on device reaches upstream")
	err = client.SendNodePoint(ncD, rootD.ID,
		data.NewPointString(data.PointTypeDescription, "", "set down"), true)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "description not propagated upstream", func() bool {
		nodes, err := client.GetNodesType[client.Device](ncU, "all", rootD.ID)
		return err == nil && len(nodes) > 0 && nodes[0].Description == "set down"
	})

	fmt.Println("**** pull: description set upstream reaches device")
	err = client.SendNodePoint(ncU, rootD.ID,
		data.NewPointString(data.PointTypeDescription, "", "set up"), true)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "description not propagated downstream", func() bool {
		nodes, err := client.GetNodesType[client.Device](ncD, "all", rootD.ID)
		return err == nil && len(nodes) > 0 && nodes[0].Description == "set up"
	})

	fmt.Println("**** node created upstream under the device reaches it")
	varU := client.Variable{ID: "varUp", Parent: rootD.ID, Description: "varUp"}
	if err := client.SendNodeType(ncU, varU, "test"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "varUp not propagated downstream", func() bool {
		nodes, err := client.GetNodesType[client.Variable](ncD, "all", "varUp")
		return err == nil && len(nodes) > 0
	})
}

// TestSyncCredentialRevoke disables a credential while the device is
// connected, checks that the session is closed and nothing more gets
// through, then re-enables it and checks that sync resumes.
func TestSyncCredentialRevoke(t *testing.T) {
	old := client.SyncRefusedRetry
	client.SyncRefusedRetry = 2 * time.Second
	defer func() { client.SyncRefusedRetry = old }()

	ncU, rootU, optsU, stopU := credUpstream(t)
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting downstream test server: ", err)
	}
	defer stopD()

	enrollDevice(t, ncU, rootU, rootD.ID, "cred-1", devicePubKey(t, ncD))
	startDeviceSync(t, ncD, rootD, optsU.NatsServer)

	waitFor(t, 15*time.Second, "credential not marked connected", func() bool {
		return getCred(t, ncU, "cred-1").Connected
	})

	fmt.Println("**** disable the credential")
	setCred(t, ncU, "cred-1", data.NewPointFloat(data.PointTypeDisabled, "", 1))

	waitFor(t, 15*time.Second, "credential still marked connected", func() bool {
		return !getCred(t, ncU, "cred-1").Connected
	})

	// a change on the device must not reach the upstream now
	err = client.SendNodePoint(ncD, rootD.ID,
		data.NewPointString(data.PointTypeDescription, "", "while revoked"), true)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Second)
	nodes, err := client.GetNodesType[client.Device](ncU, "all", rootD.ID)
	if err != nil || len(nodes) == 0 {
		t.Fatal("device node missing upstream")
	}
	if nodes[0].Description == "while revoked" {
		t.Fatal("change reached the upstream through a revoked credential")
	}

	fmt.Println("**** re-enable the credential")
	setCred(t, ncU, "cred-1", data.NewPointFloat(data.PointTypeDisabled, "", 0))

	waitFor(t, 30*time.Second, "device did not reconnect", func() bool {
		return getCred(t, ncU, "cred-1").Connected
	})
	waitFor(t, 15*time.Second, "queued change not delivered after reconnect", func() bool {
		nodes, err := client.GetNodesType[client.Device](ncU, "all", rootD.ID)
		return err == nil && len(nodes) > 0 && nodes[0].Description == "while revoked"
	})
}

// TestSyncCredentialWrongDevice gives the device a credential that sits
// under a different device node. The connection is accepted, but the
// permissions are for the other device, so nothing the device does lands.
func TestSyncCredentialWrongDevice(t *testing.T) {
	ncU, rootU, optsU, stopU := credUpstream(t)
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting downstream test server: ", err)
	}
	defer stopD()

	enrollDevice(t, ncU, rootU, "some-other-device", "cred-other", devicePubKey(t, ncD))
	startDeviceSync(t, ncD, rootD, optsU.NatsServer)

	time.Sleep(8 * time.Second)

	nodes, err := client.GetNodes(ncU, "all", rootD.ID, "", true)
	if err != nil && err != data.ErrDocumentNotFound {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Fatal("device announced itself using another device's credential")
	}
}

// TestSyncCredentialUnknownKey checks that a key the upstream has never
// seen is refused outright and the device keeps running.
func TestSyncCredentialUnknownKey(t *testing.T) {
	old := client.SyncRefusedRetry
	client.SyncRefusedRetry = 2 * time.Second
	defer func() { client.SyncRefusedRetry = old }()

	ncU, _, optsU, stopU := credUpstream(t)
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting downstream test server: ", err)
	}
	defer stopD()

	startDeviceSync(t, ncD, rootD, optsU.NatsServer)

	time.Sleep(5 * time.Second)

	nodes, err := client.GetNodes(ncU, "all", rootD.ID, "", true)
	if err != nil && err != data.ErrDocumentNotFound {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Fatal("device got in with an unknown key")
	}

	// the device is still serving locally, and says why it is not synced
	if _, err := client.GetRootNode(ncD); err != nil {
		t.Fatal("device not serving after refusal:", err)
	}
	waitFor(t, 10*time.Second, "sync node does not report the refusal", func() bool {
		syncs, err := client.GetNodesType[client.Sync](ncD, "all", "sync-id")
		return err == nil && len(syncs) > 0 && syncs[0].Error == client.SyncErrorRefused
	})

	// enrolling the key clears the error once it connects
	rootU, err := client.GetRootNode(ncU)
	if err != nil {
		t.Fatal(err)
	}
	enrollDevice(t, ncU, rootU, rootD.ID, "cred-1", devicePubKey(t, ncD))
	waitFor(t, 90*time.Second, "device did not connect once enrolled", func() bool {
		syncs, err := client.GetNodesType[client.Sync](ncD, "all", "sync-id")
		return err == nil && len(syncs) > 0 && syncs[0].Error == ""
	})
}

// TestSyncCredentialDetachedDevice deletes the device node on the upstream
// while the device is connected. A detached device must not keep writing,
// so its credential stops authorizing; restoring the node restores it.
func TestSyncCredentialDetachedDevice(t *testing.T) {
	old := client.SyncRefusedRetry
	client.SyncRefusedRetry = 2 * time.Second
	defer func() { client.SyncRefusedRetry = old }()

	ncU, rootU, optsU, stopU := credUpstream(t)
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting downstream test server: ", err)
	}
	defer stopD()

	enrollDevice(t, ncU, rootU, rootD.ID, "cred-1", devicePubKey(t, ncD))
	startDeviceSync(t, ncD, rootD, optsU.NatsServer)

	waitFor(t, 15*time.Second, "credential not marked connected", func() bool {
		return getCred(t, ncU, "cred-1").Connected
	})

	fmt.Println("**** detach the device upstream")
	err = client.SendEdgePoint(ncU, rootD.ID, rootU.ID,
		data.NewPointFloat(data.PointTypeTombstone, "", 1), true)
	if err != nil {
		t.Fatal(err)
	}

	waitFor(t, 15*time.Second, "credential still connected after detach", func() bool {
		return !getCred(t, ncU, "cred-1").Connected
	})

	fmt.Println("**** restore the device upstream")
	err = client.SendEdgePoint(ncU, rootD.ID, rootU.ID,
		data.NewPointFloat(data.PointTypeTombstone, "", 0), true)
	if err != nil {
		t.Fatal(err)
	}

	waitFor(t, 30*time.Second, "device did not reconnect after restore", func() bool {
		return getCred(t, ncU, "cred-1").Connected
	})
}

// TestSyncCredentialInstallKey is the upstream-issued workflow: the
// upstream generates a key, enrolls its public half, and the seed is
// installed on the device, which then connects with it.
func TestSyncCredentialInstallKey(t *testing.T) {
	old := client.SyncRefusedRetry
	client.SyncRefusedRetry = 2 * time.Second
	defer func() { client.SyncRefusedRetry = old }()

	ncU, rootU, optsU, stopU := credUpstream(t)
	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting downstream test server: ", err)
	}
	defer stopD()

	seed, pubKey, err := client.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	enrollDevice(t, ncU, rootU, rootD.ID, "cred-1", pubKey)

	// the device's own key is not enrolled, so it is refused
	startDeviceSync(t, ncD, rootD, optsU.NatsServer)
	time.Sleep(3 * time.Second)
	if getCred(t, ncU, "cred-1").Connected {
		t.Fatal("connected before the issued key was installed")
	}

	fmt.Println("**** install the issued key on the device")
	got, err := client.InstallDeviceKey(ncD, seed)
	if err != nil {
		t.Fatal("Error installing key: ", err)
	}
	if got != pubKey {
		t.Fatalf("installed key %v, expected %v", got, pubKey)
	}

	waitFor(t, 30*time.Second, "device did not connect with the issued key", func() bool {
		return getCred(t, ncU, "cred-1").Connected
	})

	syncs, err := client.GetNodesType[client.Sync](ncD, "all", "sync-id")
	if err != nil || len(syncs) == 0 || syncs[0].PubKey != pubKey {
		t.Fatalf("sync node does not show the installed key: %+v", syncs)
	}

	// a seed that is not a user key is refused
	if _, err := client.InstallDeviceKey(ncD, "not a seed"); err == nil {
		t.Fatal("bad seed accepted")
	}
}
