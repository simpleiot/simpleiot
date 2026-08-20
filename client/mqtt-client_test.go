package client_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/mqtttest"
	"github.com/simpleiot/simpleiot/server"
)

// mqttTestTimeout is how long an assertion waits for a message to travel
// through the broker, the client, and the store.
const mqttTestTimeout = time.Second * 15

func mqttTestBroker(t *testing.T, clientID string) *mqtttest.Client {
	t.Helper()

	addr := fmt.Sprintf("localhost:%v", server.TestServerOptions.NatsMQTTPort)

	c, err := mqtttest.Dial(addr, mqtttest.ClientID(clientID))
	if err != nil {
		t.Fatal("Error connecting to the built-in MQTT broker: ", err)
	}

	t.Cleanup(func() { _ = c.Close() })

	return c
}

// publishUntil keeps publishing until cond holds, since the client subscribes
// asynchronously and an MQTT message that arrives before it does is not
// retained for it.
func publishUntil(t *testing.T, c *mqtttest.Client, topic string, payload []byte, what string, cond func() bool) {
	t.Helper()

	start := time.Now()

	for {
		if cond() {
			return
		}

		if time.Since(start) > mqttTestTimeout {
			t.Fatalf("timeout waiting for %v", what)
		}

		if err := c.Publish(topic, payload, 0, false); err != nil {
			t.Fatal("Error publishing: ", err)
		}

		time.Sleep(time.Millisecond * 50)
	}
}

// TestMqttClient covers a scalar topic with a path, a multi-field device
// payload, the topic tag, editing the topic, and disabling a subscription.
func TestMqttClient(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	mq := client.Mqtt{
		ID:          "mqtt-1",
		Parent:      root.ID,
		Description: "Plant data",
	}

	if err := client.SendNodeType(nc, mq, "test"); err != nil {
		t.Fatal("Error creating mqtt node: ", err)
	}

	level := client.MqttSub{
		ID:          "mqtt-sub-level",
		Parent:      mq.ID,
		Description: "Tank level",
		Topic:       "plant-07/kepware-l3/press/tank_level",
		Path:        "$.value",
		Units:       "cm",
		Scale:       0.1,
	}

	if err := client.SendNodeType(nc, level, "test"); err != nil {
		t.Fatal("Error creating mqttSub node: ", err)
	}

	press := client.MqttSub{
		ID:          "mqtt-sub-press",
		Parent:      mq.ID,
		Description: "Press 1",
		Topic:       "plant-12/ignition/press1",
	}

	if err := client.SendNodeType(nc, press, "test"); err != nil {
		t.Fatal("Error creating mqttSub node: ", err)
	}

	levelGet, levelStop, err := client.NodeWatcher[client.MqttSub](nc, level.ID, mq.ID)
	if err != nil {
		t.Fatal("Error watching mqttSub node: ", err)
	}

	defer levelStop()

	pressGet, pressStop, err := client.NodeWatcher[client.MqttSub](nc, press.ID, mq.ID)
	if err != nil {
		t.Fatal("Error watching mqttSub node: ", err)
	}

	defer pressStop()

	pub := mqttTestBroker(t, "publisher")

	publishUntil(t, pub, level.Topic, []byte(`{"value": 421}`),
		"the scaled tank level point", func() bool {
			return math.Abs(pointValue(t, nc, level.ID, "")-42.1) < 1e-9
		})

	// the full topic is carried as a tag so a series can be traced back to
	// the message that produced it
	waitFor(t, mqttTestTimeout, "the topic tag", func() bool {
		return levelGet().Tags[data.PointTypeTopic] == level.Topic
	})

	// a multi-field payload with no path becomes one point per field
	publishUntil(t, pub, press.Topic,
		[]byte(`{"tank_level": 42.1, "pump_rpm": 1730}`),
		"a point per field", func() bool {
			return math.Abs(pointValue(t, nc, press.ID, "pump_rpm")-1730) < 1e-9 &&
				math.Abs(pointValue(t, nc, press.ID, "tank_level")-42.1) < 1e-9
		})

	// editing the topic resubscribes
	sendPoint(t, nc, level.ID, data.NewPointString(data.PointTypeTopic, "",
		"plant-07/kepware-l3/press/depth"))

	waitFor(t, mqttTestTimeout, "the topic edit to be applied", func() bool {
		return levelGet().Topic == "plant-07/kepware-l3/press/depth"
	})

	publishUntil(t, pub, "plant-07/kepware-l3/press/depth", []byte(`{"value": 500}`),
		"a point from the new topic", func() bool {
			return math.Abs(pointValue(t, nc, level.ID, "")-50) < 1e-9
		})

	// disabling stops the flow
	sendPoint(t, nc, press.ID, data.NewPointFloat(data.PointTypeDisabled, "", 1))

	waitFor(t, mqttTestTimeout, "the subscription to be disabled", func() bool {
		return pressGet().Disabled
	})

	if err := pub.Publish(press.Topic, []byte(`{"pump_rpm": 9999}`), 1, false); err != nil {
		t.Fatal("Error publishing: ", err)
	}

	// nothing should arrive; give the client a moment to prove it
	time.Sleep(time.Millisecond * 500)

	if v := pointValue(t, nc, press.ID, "pump_rpm"); math.Abs(v-1730) > 1e-9 {
		t.Fatalf("a disabled subscription still published points: %v", v)
	}

	// re-enabling resumes
	sendPoint(t, nc, press.ID, data.NewPointFloat(data.PointTypeDisabled, "", 0))

	publishUntil(t, pub, press.Topic, []byte(`{"pump_rpm": 1800}`),
		"points to resume after re-enabling", func() bool {
			return math.Abs(pointValue(t, nc, press.ID, "pump_rpm")-1800) < 1e-9
		})
}

// pointValue reads one value point off a node, returning NaN when it is not
// there yet.
func pointValue(t *testing.T, nc *nats.Conn, nodeID, key string) float64 {
	t.Helper()

	nodes, err := client.GetNodes(nc, "all", nodeID, "", false)
	if err != nil || len(nodes) < 1 {
		return math.NaN()
	}

	p, ok := nodes[0].Points.Find(data.PointTypeValue, key)
	if !ok {
		return math.NaN()
	}

	return p.Val()
}

// TestMqttExternalBroker checks that a configured URI reports that external
// brokers are not supported rather than failing quietly.
func TestMqttExternalBroker(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	mq := client.Mqtt{
		ID:          "mqtt-external",
		Parent:      root.ID,
		Description: "Plant broker",
		URI:         "mqtt://broker.example.com:1883",
	}

	if err := client.SendNodeType(nc, mq, "test"); err != nil {
		t.Fatal("Error creating mqtt node: ", err)
	}

	get, stopWatch, err := client.NodeWatcher[client.Mqtt](nc, mq.ID, root.ID)
	if err != nil {
		t.Fatal("Error watching mqtt node: ", err)
	}

	defer stopWatch()

	waitFor(t, mqttTestTimeout, "the external broker error", func() bool {
		return get().Error != ""
	})

	// clearing the URI should clear the error and start the subscriptions
	sendPoint(t, nc, mq.ID, data.NewPointString(data.PointTypeURI, "", ""))

	waitFor(t, mqttTestTimeout, "the error to clear", func() bool {
		return get().Error == ""
	})
}

// TestMqttTwoNodes checks the multi-site shape: two mqtt nodes against the
// same built-in broker, each receiving only its own topics.
func TestMqttTwoNodes(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	for i, site := range []string{"plant-07", "plant-12"} {
		mq := client.Mqtt{
			ID:          fmt.Sprintf("mqtt-site-%v", i),
			Parent:      root.ID,
			Description: site,
		}

		if err := client.SendNodeType(nc, mq, "test"); err != nil {
			t.Fatal("Error creating mqtt node: ", err)
		}

		sub := client.MqttSub{
			ID:          fmt.Sprintf("mqtt-site-sub-%v", i),
			Parent:      mq.ID,
			Description: site + " level",
			Topic:       site + "/gw/tank",
			Path:        "$.value",
		}

		if err := client.SendNodeType(nc, sub, "test"); err != nil {
			t.Fatal("Error creating mqttSub node: ", err)
		}
	}

	pub := mqttTestBroker(t, "two-node-publisher")

	publishUntil(t, pub, "plant-07/gw/tank", []byte(`{"value": 7}`),
		"the first site's point", func() bool {
			return math.Abs(pointValue(t, nc, "mqtt-site-sub-0", "")-7) < 1e-9
		})

	if !math.IsNaN(pointValue(t, nc, "mqtt-site-sub-1", "")) {
		t.Fatal("the second site received a message meant for the first")
	}

	publishUntil(t, pub, "plant-12/gw/tank", []byte(`{"value": 12}`),
		"the second site's point", func() bool {
			return math.Abs(pointValue(t, nc, "mqtt-site-sub-1", "")-12) < 1e-9
		})
}
