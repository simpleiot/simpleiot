package client_test

import (
	"strings"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/server"
)

// TestMqttScopedFilters verifies that an mqtt node below a group may not
// subscribe to every topic, while one directly under the root may.
func TestMqttScopedFilters(t *testing.T) {
	nc, root, stop, err := server.TestServerOpts(sec22ServerOptions)
	if err != nil {
		t.Fatal("Error starting test server:", err)
	}
	defer stop()

	send := func(n any) {
		t.Helper()
		if err := client.SendNodeType(nc, n, "test"); err != nil {
			t.Fatal(err)
		}
	}

	send(client.Group{ID: "ID-group", Parent: root.ID, Description: "plant"})
	send(client.Mqtt{ID: "ID-mqtt-scoped", Parent: "ID-group", Description: "in a group"})
	send(client.MqttSub{ID: "ID-sub-all", Parent: "ID-mqtt-scoped", Description: "all", Topic: "#"})
	send(client.MqttSub{ID: "ID-sub-plus", Parent: "ID-mqtt-scoped", Description: "plus", Topic: "+/temp"})
	send(client.MqttSub{ID: "ID-sub-site", Parent: "ID-mqtt-scoped", Description: "site", Topic: "plant-07/#"})

	send(client.Mqtt{ID: "ID-mqtt-top", Parent: root.ID, Description: "under root"})
	send(client.MqttSub{ID: "ID-sub-top-all", Parent: "ID-mqtt-top", Description: "all", Topic: "#"})

	// subError waits for the client to start and returns the error point on
	// a subscription node
	subError := func(id string) string {
		var last string
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			subs, err := client.GetNodesType[client.MqttSub](nc, "all", id)
			if err == nil && len(subs) > 0 {
				last = subs[0].Error
				if last != "" {
					return last
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
		return last
	}

	if e := subError("ID-sub-all"); !strings.Contains(e, "literal topic level") {
		t.Errorf("# below a group: got error %q, want a refusal", e)
	}
	if e := subError("ID-sub-plus"); !strings.Contains(e, "literal topic level") {
		t.Errorf("+/temp below a group: got error %q, want a refusal", e)
	}

	// the allowed subscriptions carry no error once the clients are up
	time.Sleep(250 * time.Millisecond)
	for _, id := range []string{"ID-sub-site", "ID-sub-top-all"} {
		subs, err := client.GetNodesType[client.MqttSub](nc, "all", id)
		if err != nil || len(subs) == 0 {
			t.Fatalf("%v: %v nodes, %v", id, len(subs), err)
		}
		if subs[0].Error != "" {
			t.Errorf("%v: unexpected error %q", id, subs[0].Error)
		}
	}
}
