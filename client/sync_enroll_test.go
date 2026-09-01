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

// makeEnrollToken creates an enrollment token under the upstream root and
// returns the token.
func makeEnrollToken(t *testing.T, ncU *nats.Conn, rootU data.NodeEdge, id string, autoApprove bool) string {
	t.Helper()

	token, hash, err := client.GenerateEnrollToken()
	if err != nil {
		t.Fatal(err)
	}

	et := client.EnrollToken{ID: id, Parent: rootU.ID, Description: "fleet",
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

	token := makeEnrollToken(t, ncU, rootU, "et-1", false)
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
	_, otherPub, err := client.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	reply, err := client.Enroll(optsU.NatsServer, client.EnrollRequest{
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
	if _, err := client.Enroll(optsU.NatsServer, client.EnrollRequest{
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

	token := makeEnrollToken(t, ncU, rootU, "et-1", true)
	time.Sleep(500 * time.Millisecond)

	startEnrollSync(t, ncD, rootD, optsU.NatsServer, token)

	waitFor(t, 30*time.Second, "device did not connect", func() bool {
		creds := deviceCreds(t, ncU, rootD.ID)
		return len(creds) == 1 && !creds[0].Pending && creds[0].Connected
	})
	if e := syncError(t, ncD); e != "" {
		t.Fatalf("sync node reports %q after auto-approval", e)
	}
}

// TestEnrollTokenScope checks what a connection made with an enrollment
// token can do: ask to enroll, and nothing else.
func TestEnrollTokenScope(t *testing.T) {
	ncU, rootU, optsU, stopU := credUpstream(t)
	defer stopU()

	token := makeEnrollToken(t, ncU, rootU, "et-1", false)
	time.Sleep(500 * time.Millisecond)

	nc, err := nats.Connect(optsU.NatsServer, nats.Token(token), nats.NoReconnect())
	if err != nil {
		t.Fatal("enrollment token refused:", err)
	}
	defer nc.Close()

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
	_, pub, _ := client.GenerateDeviceKey()
	reply, err := client.Enroll(optsU.NatsServer, client.EnrollRequest{
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
