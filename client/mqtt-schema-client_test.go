package client_test

import (
	"math"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// schemaFind walks a path of auto-created nodes, matching each level on the
// identity point rather than the description, and returns the node at the end.
func schemaFind(t *testing.T, nc *nats.Conn, parent string, path ...string) string {
	t.Helper()

	id := parent

	for _, level := range path {
		nodes, err := client.GetNodes(nc, id, "all", "", false)
		if err != nil {
			return ""
		}

		found := ""

		for _, n := range nodes {
			if p, ok := n.Points.Find(data.PointTypeID, ""); ok && p.Txt() == level {
				found = n.ID
				break
			}
		}

		if found == "" {
			return ""
		}

		id = found
	}

	return id
}

// TestMqttTopicSchema covers auto-creation, updates without new nodes, the
// tag and identity points, and what survives a restart.
func TestMqttTopicSchema(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	restarted := false

	defer func() {
		if !restarted {
			stop()
		}
	}()

	mq := client.Mqtt{
		ID:          "mqtt-schema",
		Parent:      root.ID,
		Description: "Plant data",
		TopicSchema: "{site}/{gateway}/{device}",
	}

	if err := client.SendNodeType(nc, mq, "test"); err != nil {
		t.Fatal("Error creating mqtt node: ", err)
	}

	pub := mqttTestBroker(t, "schema-publisher")

	var deviceID string

	publishUntil(t, pub, "plant-07/kepware-l3/press/tank_level", []byte(`{"value": 42.1}`),
		"the node path to be created with its point", func() bool {
			deviceID = schemaFind(t, nc, mq.ID, "plant-07", "kepware-l3", "press")
			if deviceID == "" {
				return false
			}

			return math.Abs(pointValue(t, nc, deviceID, "tank_level")-42.1) < 1e-9
		})

	// each named level carries a tag named by its schema label
	gatewayID := schemaFind(t, nc, mq.ID, "plant-07", "kepware-l3")

	if p, ok := spNodePoint(t, nc, gatewayID, data.PointTypeTag, "gateway"); !ok || p.Txt() != "kepware-l3" {
		t.Fatal("the gateway level did not get its tag")
	}

	if p, ok := spNodePoint(t, nc, deviceID, data.PointTypeTag, "device"); !ok || p.Txt() != "press" {
		t.Fatal("the device level did not get its tag")
	}

	// a second message on the same topic updates the point and creates nothing
	publishUntil(t, pub, "plant-07/kepware-l3/press/tank_level", []byte(`{"value": 43.2}`),
		"the point to update", func() bool {
			return math.Abs(pointValue(t, nc, deviceID, "tank_level")-43.2) < 1e-9
		})

	devices, err := client.GetNodes(nc, gatewayID, "all", data.NodeTypeMqttDevice, false)
	if err != nil {
		t.Fatal("Error getting device nodes: ", err)
	}

	if len(devices) != 1 {
		t.Fatalf("expected one device node, got %v", len(devices))
	}

	// a rename and a tag added by hand must survive later messages
	sendPoint(t, nc, deviceID, data.NewPointString(data.PointTypeDescription, "", "Press 3"))
	sendPoint(t, nc, deviceID, data.NewPointString(data.PointTypeTag, "machine", "press-3"))

	stop()
	restarted = true

	nc2, _, stop2, err := server.TestServerOptsKeepStore(server.TestServerOptions)
	if err != nil {
		t.Fatal("Error restarting test server: ", err)
	}

	defer stop2()

	pub2 := mqttTestBroker(t, "schema-publisher-2")

	publishUntil(t, pub2, "plant-07/kepware-l3/press/tank_level", []byte(`{"value": 44.3}`),
		"the point to update after the restart", func() bool {
			return math.Abs(pointValue(t, nc2, deviceID, "tank_level")-44.3) < 1e-9
		})

	if id := schemaFind(t, nc2, mq.ID, "plant-07", "kepware-l3", "press"); id != deviceID {
		t.Fatalf("the device node was not matched by identity after the restart: %v", id)
	}

	if p, ok := spNodePoint(t, nc2, deviceID, data.PointTypeDescription, ""); !ok || p.Txt() != "Press 3" {
		t.Fatal("the renamed description did not survive")
	}

	if p, ok := spNodePoint(t, nc2, deviceID, data.PointTypeTag, "machine"); !ok || p.Txt() != "press-3" {
		t.Fatal("a tag set by hand did not survive")
	}
}

// TestMqttTopicSchemaMaxNodes checks the guard against a topic level that
// carries an unbounded value.
func TestMqttTopicSchemaMaxNodes(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	mq := client.Mqtt{
		ID:          "mqtt-max-nodes",
		Parent:      root.ID,
		Description: "Plant data",
		TopicSchema: "{site}/{device}",
		// one site node and one device node
		MaxNodes: 2,
	}

	if err := client.SendNodeType(nc, mq, "test"); err != nil {
		t.Fatal("Error creating mqtt node: ", err)
	}

	get, stopWatch, err := client.NodeWatcher[client.Mqtt](nc, mq.ID, root.ID)
	if err != nil {
		t.Fatal("Error watching mqtt node: ", err)
	}

	defer stopWatch()

	pub := mqttTestBroker(t, "max-nodes-publisher")

	var deviceID string

	publishUntil(t, pub, "plant-07/press", []byte(`{"value": 1}`),
		"the first device", func() bool {
			deviceID = schemaFind(t, nc, mq.ID, "plant-07", "press")
			return deviceID != ""
		})

	// the limit is reached, so a new topic is dropped and reported
	waitFor(t, mqttTestTimeout, "the node limit to be reported", func() bool {
		if err := pub.Publish("plant-07/oven", []byte(`{"value": 2}`), 0, false); err != nil {
			t.Fatal("Error publishing: ", err)
		}

		return get().Error != ""
	})

	if id := schemaFind(t, nc, mq.ID, "plant-07", "oven"); id != "" {
		t.Fatal("a node was created past the limit")
	}

	// the nodes that already exist keep updating
	publishUntil(t, pub, "plant-07/press", []byte(`{"value": 3}`),
		"the existing device to keep updating", func() bool {
			return math.Abs(pointValue(t, nc, deviceID, "")-3) < 1e-9
		})
}

// TestMqttTopicSchemaPrecedence checks that a topic named by a subscription
// node is handled by that node alone.
func TestMqttTopicSchemaPrecedence(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	mq := client.Mqtt{
		ID:          "mqtt-precedence",
		Parent:      root.ID,
		Description: "Plant data",
		TopicSchema: "{site}/{device}",
	}

	if err := client.SendNodeType(nc, mq, "test"); err != nil {
		t.Fatal("Error creating mqtt node: ", err)
	}

	sub := client.MqttSub{
		ID:          "mqtt-precedence-sub",
		Parent:      mq.ID,
		Description: "Tank level",
		Topic:       "plant-07/press/tank_level",
		Path:        "$.value",
		Units:       "cm",
		Scale:       0.1,
	}

	if err := client.SendNodeType(nc, sub, "test"); err != nil {
		t.Fatal("Error creating mqttSub node: ", err)
	}

	pub := mqttTestBroker(t, "precedence-publisher")

	publishUntil(t, pub, sub.Topic, []byte(`{"value": 421}`),
		"the subscription point, with its scale applied", func() bool {
			return math.Abs(pointValue(t, nc, sub.ID, "")-42.1) < 1e-9
		})

	// the schema must not have built a second path for the same topic
	time.Sleep(time.Millisecond * 500)

	if id := schemaFind(t, nc, mq.ID, "plant-07", "press"); id != "" {
		t.Fatal("the topic schema handled a topic an explicit subscription names")
	}

	// a topic the subscription does not name still goes through the schema
	publishUntil(t, pub, "plant-07/oven/temp", []byte(`{"value": 180}`),
		"the schema to handle an unnamed topic", func() bool {
			id := schemaFind(t, nc, mq.ID, "plant-07", "oven")
			return id != "" && math.Abs(pointValue(t, nc, id, "temp")-180) < 1e-9
		})
}
