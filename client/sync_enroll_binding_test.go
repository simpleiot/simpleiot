package client_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/server"
)

// TestEnrollBinding covers what an enrollment request is tied to: the key
// on the connection, a device ID that can travel in a subject, and an
// operator's approval once the device already has a live credential. It
// also checks that the device key seed is not handed out on the bus.
func TestEnrollBinding(t *testing.T) {
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

	// the seed stays in the server process
	msg, err := ncD.Request(client.SubjectDeviceKey, nil, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(msg.Data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["seed"]; ok || strings.Contains(string(msg.Data), "SUA") {
		t.Fatalf("device key reply carries the seed: %s", msg.Data)
	}

	token := makeEnrollToken(t, ncU, rootU, "et-1", true)
	time.Sleep(500 * time.Millisecond)

	startEnrollSync(t, ncD, rootD, optsU.NatsServer, token)
	waitFor(t, 30*time.Second, "device did not connect", func() bool {
		creds := deviceCreds(t, ncU, rootD.ID)
		return len(creds) == 1 && !creds[0].Pending && creds[0].Connected
	})

	seed2, pub2, err := client.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	_, sign2, _ := client.SeedSigner(seed2)
	_, pub3, _ := client.GenerateDeviceKey()

	// a second key onto a device with a live credential waits for an
	// operator, even under an auto-approve token
	reply, err := client.Enroll(optsU.NatsServer, pub2, sign2, client.EnrollRequest{
		Token: token, DeviceID: rootD.ID, PubKey: pub2})
	if err != nil || reply.Status != client.EnrollPending {
		t.Fatalf("second key: %v %v", reply, err)
	}
	for _, c := range deviceCreds(t, ncU, rootD.ID) {
		if c.PubKey == pub2 && !c.Pending {
			t.Fatal("second key was approved")
		}
	}

	// pub2 is now a pending credential and can no longer connect with
	// the token; the remaining cases use a key the upstream has not seen
	seed4, pub4, err := client.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	_, sign4, _ := client.SeedSigner(seed4)

	// a request naming a key other than the connection's is refused
	_, err = client.Enroll(optsU.NatsServer, pub4, sign4, client.EnrollRequest{
		Token: token, DeviceID: "unit-9", PubKey: pub3})
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("foreign key: %v", err)
	}

	// a device ID that cannot be a subject token is refused before
	// anything is created
	for _, id := range []string{"a.b", "a*", "a>", "a b", ""} {
		_, err = client.Enroll(optsU.NatsServer, pub4, sign4, client.EnrollRequest{
			Token: token, DeviceID: id, PubKey: pub4})
		if err == nil {
			t.Fatalf("device ID %q accepted", id)
		}
	}
	devs, err := client.GetNodesType[client.Device](ncU, rootU.ID, "all")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range devs {
		if d.ID != rootD.ID {
			t.Fatalf("unexpected device node %v", d.ID)
		}
	}
}
