package client_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// makeEnrollToken creates an enrollment token under a node on the upstream
// and returns the token.
func makeEnrollToken(t *testing.T, ncU *nats.Conn, parent, id string, autoApprove bool) string {
	t.Helper()

	token, hash, err := client.GenerateEnrollToken()
	if err != nil {
		t.Fatal(err)
	}

	et := client.EnrollToken{ID: id, Parent: parent, Description: "fleet",
		TokenHash: hash, AutoApprove: autoApprove}
	if err := client.SendNodeType(ncU, et, "test"); err != nil {
		t.Fatal(err)
	}

	return token
}

// startEnrollSync creates a sync node on the downstream with an enrollment
// token and no auth token.
func startEnrollSync(t *testing.T, ncD *nats.Conn, rootD data.NodeEdge, uri, token string) {
	t.Helper()

	sync := client.Sync{ID: "sync-id", Parent: rootD.ID, Description: "sync to up",
		URI: uri, EnrollToken: token}
	if err := client.SendNodeType(ncD, sync, "test"); err != nil {
		t.Fatal(err)
	}
}

func syncError(t *testing.T, ncD *nats.Conn) string {
	t.Helper()
	syncs, err := client.GetNodesType[client.Sync](ncD, "all", "sync-id")
	if err != nil || len(syncs) == 0 {
		return ""
	}
	return syncs[0].Error
}

func deviceCreds(t *testing.T, ncU *nats.Conn, deviceID string) []client.DeviceCred {
	t.Helper()
	creds, err := client.GetNodesType[client.DeviceCred](ncU, deviceID, "all")
	if err != nil && err != data.ErrDocumentNotFound {
		t.Fatal(err)
	}
	return creds
}

// TestSyncEnroll is the device-initiated round trip: the device enrolls
// with the fleet token, waits as pending, and connects once approved.
func TestSyncEnroll(t *testing.T) {
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

	token := makeEnrollToken(t, ncU, rootU.ID, "et-1", false)
	// the authorizer indexes the token from the tree change
	time.Sleep(500 * time.Millisecond)

	startEnrollSync(t, ncD, rootD, optsU.NatsServer, token)

	fmt.Println("**** device enrolls and waits for approval")
	waitFor(t, 20*time.Second, "device node not created by enrollment", func() bool {
		nodes, err := client.GetNodes(ncU, "all", rootD.ID, "", false)
		return err == nil && len(nodes) > 0
	})
	waitFor(t, 10*time.Second, "pending credential not created", func() bool {
		creds := deviceCreds(t, ncU, rootD.ID)
		return len(creds) == 1 && creds[0].Pending && creds[0].PubKey == devicePubKey(t, ncD)
	})
	waitFor(t, 10*time.Second, "sync node does not say pending", func() bool {
		return syncError(t, ncD) == client.SyncErrorPending
	})

	// pending means not authorized; give it a retry cycle
	time.Sleep(3 * time.Second)
	if deviceCreds(t, ncU, rootD.ID)[0].Connected {
		t.Fatal("pending credential connected")
	}

	fmt.Println("**** approve")
	credID := deviceCreds(t, ncU, rootD.ID)[0].ID
	setCred(t, ncU, credID, data.NewPointFloat(data.PointTypePending, "", 0))

	waitFor(t, 30*time.Second, "device did not connect after approval", func() bool {
		return getCred(t, ncU, credID).Connected
	})
	waitFor(t, 10*time.Second, "sync error not cleared", func() bool {
		return syncError(t, ncD) == ""
	})

	// and it syncs
	err = client.SendNodePoint(ncD, rootD.ID,
		data.NewPointString(data.PointTypeDescription, "", "enrolled device"), true)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "description not propagated upstream", func() bool {
		nodes, err := client.GetNodesType[client.Device](ncU, "all", rootD.ID)
		return err == nil && len(nodes) > 0 && nodes[0].Description == "enrolled device"
	})

	fmt.Println("**** a second key for the same device is held as pending")
	otherSeed, otherPub, err := client.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	_, otherSign, _ := client.SeedSigner(otherSeed)
	reply, err := client.Enroll(optsU.NatsServer, otherPub, otherSign, client.EnrollRequest{
		Token: token, DeviceID: rootD.ID, PubKey: otherPub})
	if err != nil || reply.Status != client.EnrollPending {
		t.Fatalf("second enrollment: %v %v", reply, err)
	}
	creds := deviceCreds(t, ncU, rootD.ID)
	if len(creds) != 2 {
		t.Fatalf("expected 2 credentials, got %v", len(creds))
	}
	for _, c := range creds {
		if c.PubKey == otherPub && !c.Pending {
			t.Fatal("second key was approved")
		}
		if c.ID == credID && c.Pending {
			t.Fatal("approved key was replaced")
		}
	}

	fmt.Println("**** revoking the token does not affect the enrolled device")
	setCred(t, ncU, "et-1", data.NewPointFloat(data.PointTypeDisabled, "", 1))
	time.Sleep(time.Second)
	if _, err := client.Enroll(optsU.NatsServer, otherPub, otherSign, client.EnrollRequest{
		Token: token, DeviceID: "another", PubKey: otherPub}); err == nil {
		t.Fatal("revoked enrollment token accepted")
	}
	if !getCred(t, ncU, credID).Connected {
		t.Fatal("enrolled device lost its connection when the token was revoked")
	}
}

// TestSyncEnrollAutoApprove enrolls against a token that approves straight
// away, so the device connects with nobody involved.
func TestSyncEnrollAutoApprove(t *testing.T) {
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

	makeGroup(t, ncU, rootU.ID, "group-a")
	makeGroup(t, ncU, rootU.ID, "group-b")
	token := makeEnrollToken(t, ncU, "group-a", "et-1", true)
	time.Sleep(500 * time.Millisecond)

	startEnrollSync(t, ncD, rootD, optsU.NatsServer, token)

	waitFor(t, 30*time.Second, "device did not connect", func() bool {
		creds := deviceCreds(t, ncU, rootD.ID)
		return len(creds) == 1 && !creds[0].Pending && creds[0].Connected
	})
	if e := syncError(t, ncD); e != "" {
		t.Fatalf("sync node reports %q after auto-approval", e)
	}
	if got := nodeParents(t, ncU, rootD.ID); len(got) != 1 || got[0] != "group-a" {
		t.Fatalf("device enrolled with group-a's token sits under %v", got)
	}
	if by := deviceCreds(t, ncU, rootD.ID)[0].EnrolledBy; by != "et-1" {
		t.Fatalf("credential records enrolledBy %q", by)
	}

	fmt.Println("**** moving the device keeps its credential and connection")
	if err := client.MoveNode(ncU, rootD.ID, "group-a", "group-b", "test"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	creds := deviceCreds(t, ncU, rootD.ID)
	if len(creds) != 1 || creds[0].Pending || !creds[0].Connected {
		t.Fatalf("moved device lost its credential or connection: %+v", creds)
	}
	err = client.SendNodePoint(ncD, rootD.ID,
		data.NewPointString(data.PointTypeDescription, "", "moved device"), true)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 10*time.Second, "moved device does not sync", func() bool {
		nodes, err := client.GetNodesType[client.Device](ncU, "group-b", rootD.ID)
		return err == nil && len(nodes) > 0 && nodes[0].Description == "moved device"
	})
}

func makeGroup(t *testing.T, nc *nats.Conn, parent, id string) {
	t.Helper()
	g := client.Group{ID: id, Parent: parent, Description: id}
	if err := client.SendNodeType(nc, g, "test"); err != nil {
		t.Fatal(err)
	}
}

// nodeParents lists the parents of a node's live edges.
func nodeParents(t *testing.T, nc *nats.Conn, id string) []string {
	t.Helper()
	nodes, err := client.GetNodes(nc, "all", id, "", false)
	if err != nil && err != data.ErrDocumentNotFound {
		t.Fatal(err)
	}
	var parents []string
	for _, n := range nodes {
		parents = append(parents, n.Parent)
	}
	return parents
}

// enrollAs asks the upstream to enroll a fresh key for a device ID.
func enrollAs(t *testing.T, uri, token, deviceID string) (client.EnrollReply, string, error) {
	t.Helper()
	seed, pub, err := client.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	_, sign, _ := client.SeedSigner(seed)
	reply, err := client.Enroll(uri, pub, sign, client.EnrollRequest{
		Token: token, DeviceID: deviceID, PubKey: pub})
	return reply, pub, err
}

// TestEnrollPlacement checks where enrollment creates a device: under the
// group its token sits in, and nowhere when the token sits anywhere but
// the root or a group still in the tree, or in two places. A device that
// already exists stays where it is and its new key waits for approval.
func TestEnrollPlacement(t *testing.T) {
	ncU, rootU, optsU, stopU := credUpstream(t)
	defer stopU()
	uri := optsU.NatsServer

	makeGroup(t, ncU, rootU.ID, "group-a")
	makeGroup(t, ncU, rootU.ID, "group-b")
	makeGroup(t, ncU, rootU.ID, "group-c")
	makeGroup(t, ncU, "group-c", "group-c2")
	dev := client.Device{ID: "dev-x", Parent: rootU.ID, Description: "a device"}
	if err := client.SendNodeType(ncU, dev, "test"); err != nil {
		t.Fatal(err)
	}

	tokA := makeEnrollToken(t, ncU, "group-a", "et-a", true)
	tokB := makeEnrollToken(t, ncU, "group-b", "et-b", false)
	tokRoot := makeEnrollToken(t, ncU, rootU.ID, "et-root", false)
	tokDev := makeEnrollToken(t, ncU, "dev-x", "et-dev", true)
	tokC := makeEnrollToken(t, ncU, "group-c", "et-c", true)
	tokC2 := makeEnrollToken(t, ncU, "group-c2", "et-c2", true)
	time.Sleep(500 * time.Millisecond)

	placed := func(token, deviceID, wantParent, wantStatus string) {
		t.Helper()
		reply, _, err := enrollAs(t, uri, token, deviceID)
		if err != nil {
			t.Fatalf("enrolling %v: %v", deviceID, err)
		}
		if reply.Status != wantStatus {
			t.Errorf("%v enrolled %v, want %v", deviceID, reply.Status, wantStatus)
		}
		if got := nodeParents(t, ncU, deviceID); len(got) != 1 || got[0] != wantParent {
			t.Errorf("%v sits under %v, want %v", deviceID, got, wantParent)
		}
	}

	refused := func(token, deviceID string) {
		t.Helper()
		if _, _, err := enrollAs(t, uri, token, deviceID); err == nil ||
			!strings.Contains(err.Error(), "not in the root or a group") {
			t.Errorf("enrolling %v: want a placement refusal, got %v", deviceID, err)
		}
		if got := nodeParents(t, ncU, deviceID); len(got) != 0 {
			t.Errorf("refused enrollment created %v under %v", deviceID, got)
		}
	}

	fmt.Println("**** each token enrolls into its own place")
	placed(tokA, "d-a", "group-a", client.EnrollApproved)
	placed(tokB, "d-b", "group-b", client.EnrollPending)
	placed(tokRoot, "d-root", rootU.ID, client.EnrollPending)
	if by := deviceCreds(t, ncU, "d-a")[0].EnrolledBy; by != "et-a" {
		t.Errorf("credential records enrolledBy %q, want et-a", by)
	}

	fmt.Println("**** a token under a device enrolls nothing")
	refused(tokDev, "d-dev")

	fmt.Println("**** a mirrored token enrolls nothing until the mirror goes")
	if err := client.MirrorNode(ncU, "et-b", "group-b", "group-a", "test"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	refused(tokB, "d-mirror")
	if err := client.DeleteNode(ncU, "et-b", "group-a", "test"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	placed(tokB, "d-mirror", "group-b", client.EnrollPending)

	fmt.Println("**** a moved token sends the next device to its new place")
	if err := client.MoveNode(ncU, "et-root", rootU.ID, "group-b", "test"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	placed(tokRoot, "d-moved", "group-b", client.EnrollPending)

	fmt.Println("**** a token under a deleted group, or below one, enrolls nothing")
	if err := client.DeleteNode(ncU, "group-c", rootU.ID, "test"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	refused(tokC, "d-c")
	refused(tokC2, "d-c2")

	fmt.Println("**** an existing device stays put and its new key waits")
	if err := client.MoveNode(ncU, "d-a", "group-a", "group-b", "test"); err != nil {
		t.Fatal(err)
	}
	placed(tokA, "d-a", "group-b", client.EnrollPending)
	creds := deviceCreds(t, ncU, "d-a")
	if len(creds) != 2 {
		t.Fatalf("expected 2 credentials on d-a, got %v", len(creds))
	}
	approved := 0
	for _, c := range creds {
		if !c.Pending {
			approved++
		}
	}
	if approved != 1 {
		t.Errorf("d-a has %v approved credentials, want the original one", approved)
	}

	fmt.Println("**** auto-approve does not cover a device made by hand")
	byHand := client.Device{ID: "d-hand", Parent: "group-b", Description: "made by hand"}
	if err := client.SendNodeType(ncU, byHand, "test"); err != nil {
		t.Fatal(err)
	}
	placed(tokA, "d-hand", "group-b", client.EnrollPending)
}

// TestEnrollTokenScope checks what a connection made with an enrollment
// token can do: ask to enroll, and nothing else.
func TestEnrollTokenScope(t *testing.T) {
	ncU, rootU, optsU, stopU := credUpstream(t)
	defer stopU()

	token := makeEnrollToken(t, ncU, rootU.ID, "et-1", false)
	time.Sleep(500 * time.Millisecond)

	// the token on its own is not a credential: the key being enrolled
	// comes with it, so the connection can be given an inbox of its own
	if _, err := nats.Connect(optsU.NatsServer, nats.Token(token),
		nats.NoReconnect()); err == nil {
		t.Fatal("enrollment token accepted without a key")
	}

	seed, pub, err := client.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		t.Fatal(err)
	}

	errs := make(chan string, 8)
	nc, err := nats.Connect(optsU.NatsServer, nats.Token(token),
		nats.Nkey(pub, kp.Sign), nats.CustomInboxPrefix(client.InboxPrefix(pub)),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			select {
			case errs <- err.Error():
			default:
			}
		}), nats.NoReconnect())
	if err != nil {
		t.Fatal("enrollment token refused:", err)
	}
	defer nc.Close()

	// it cannot read anyone else's replies
	if _, err := nc.SubscribeSync("_INBOX.>"); err != nil {
		t.Fatal(err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-errs:
		if !strings.Contains(e, "Permissions Violation") {
			t.Fatalf("subscribing to _INBOX.> gave %v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("enrolling connection could subscribe to the shared inbox")
	}

	if _, err := nc.Request("nodes.root.all", nil, 500*time.Millisecond); err == nil {
		t.Fatal("enrollment token could read the tree")
	}
	if _, err := nc.Request("auth.deviceKey", nil, 500*time.Millisecond); err == nil {
		t.Fatal("enrollment token could read the device key")
	}
	if err := nc.Publish("p."+rootU.ID, []byte("x")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if _, err := client.GetNodes(ncU, "all", "nope", "", false); err != nil &&
		err != data.ErrDocumentNotFound {
		t.Fatal(err)
	}

	// the one thing it can do
	_, sign, _ := client.SeedSigner(seed)
	reply, err := client.Enroll(optsU.NatsServer, pub, sign, client.EnrollRequest{
		Token: token, DeviceID: "unit-7", PubKey: pub, Description: "unit 7"})
	if err != nil || reply.Status != client.EnrollPending {
		t.Fatalf("enroll: %v %v", reply, err)
	}
	devs, err := client.GetNodesType[client.Device](ncU, "all", "unit-7")
	if err != nil || len(devs) == 0 || devs[0].Description != "unit 7" {
		t.Fatalf("device node not created: %v %v", devs, err)
	}

	// a wrong token is refused outright
	if _, err := nats.Connect(optsU.NatsServer, nats.Token("ETnope")); err == nil {
		t.Fatal("bad token accepted")
	}
}
