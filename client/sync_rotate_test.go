package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestSyncCredentialRotate swaps a device's key while it is publishing: a
// second credential is added with a key the upstream issued, the seed is
// installed on the device, and the old credential is disabled once the new
// one has connected. Every message published on the device has to reach the
// upstream's replica stream.
func TestSyncCredentialRotate(t *testing.T) {
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

	counter := client.Variable{ID: "counter", Parent: rootD.ID, Description: "counter"}
	if err := client.SendNodeType(ncD, counter, "test"); err != nil {
		t.Fatal(err)
	}

	// publish steadily through the rotation
	const n = 150
	published := make(chan struct{})
	go func() {
		defer close(published)
		for i := 1; i <= n; i++ {
			p := data.NewPointFloat(data.PointTypeValue, "", float64(i))
			if err := client.SendNodePoint(ncD, "counter", p, true); err != nil {
				t.Error("publish:", err)
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	fmt.Println("**** issue a second credential and install it")
	seed, pubKey, err := client.GenerateDeviceKey()
	if err != nil {
		t.Fatal(err)
	}
	cred := client.DeviceCred{ID: "cred-2", Parent: rootD.ID, Description: "rotated", PubKey: pubKey}
	if err := client.SendNodeType(ncU, cred, "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.InstallDeviceKey(ncD, seed); err != nil {
		t.Fatal(err)
	}

	waitFor(t, 30*time.Second, "new credential not connected", func() bool {
		return getCred(t, ncU, "cred-2").Connected
	})
	if getCred(t, ncU, "cred-2").LastConnect == 0 {
		t.Fatal("lastConnect not set on the new credential")
	}

	fmt.Println("**** retire the old credential")
	setCred(t, ncU, "cred-1", data.NewPointFloat(data.PointTypeDisabled, "", 1))

	<-published

	waitFor(t, 30*time.Second, "final value not upstream", func() bool {
		vars, err := client.GetNodesType[client.Variable](ncU, "all", "counter")
		return err == nil && len(vars) > 0 && vars[0].Value["0"] == n
	})

	// the replica stream on the upstream holds every message; a window
	// resent after a reconnect may add duplicates, never gaps
	js, err := jetstream.New(ncU)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := js.Stream(ctx, fmt.Sprintf("inst_%v_%v", rootD.ID, rootD.ID))
	if err != nil {
		t.Fatal(err)
	}
	subject := fmt.Sprintf("inst.%v.%v.counter.p.value.*", rootD.ID, rootD.ID)
	info, err := stream.Info(ctx, jetstream.WithSubjectFilter(subject))
	if err != nil {
		t.Fatal(err)
	}
	var count uint64
	for _, c := range info.State.Subjects {
		count += c
	}
	if count < n {
		t.Fatalf("replica stream holds %v value messages, expected at least %v", count, n)
	}

	// the sync node reports the new key and no error
	syncs, err := client.GetNodesType[client.Sync](ncD, "all", "sync-id")
	if err != nil || len(syncs) == 0 {
		t.Fatal("sync node not found")
	}
	if syncs[0].PubKey != pubKey {
		t.Fatalf("sync node shows %v, expected the installed key", syncs[0].PubKey)
	}
	if syncs[0].Error != "" {
		t.Fatalf("sync node reports an error after rotation: %v", syncs[0].Error)
	}
}
