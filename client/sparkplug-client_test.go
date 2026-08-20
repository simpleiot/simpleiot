package client_test

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/pb/sparkplug"
	"github.com/simpleiot/simpleiot/server"
	"google.golang.org/protobuf/proto"
)

const (
	spGroup  = "plant-03"
	spEdge   = "ignition-edge"
	spDevice = "press-1"
)

func spMetric(name string, alias uint64, value float64) *sparkplug.Payload_Metric {
	m := &sparkplug.Payload_Metric{
		Value: &sparkplug.Payload_Metric_DoubleValue{DoubleValue: value},
	}

	if name != "" {
		m.Name = &name
	}

	if alias != 0 {
		m.Alias = &alias
	}

	return m
}

func spPayload(t *testing.T, metrics ...*sparkplug.Payload_Metric) []byte {
	t.Helper()

	ts := uint64(time.Now().UnixMilli())

	b, err := proto.Marshal(&sparkplug.Payload{Timestamp: &ts, Metrics: metrics})
	if err != nil {
		t.Fatal("Error encoding Sparkplug payload: ", err)
	}

	return b
}

// spFindNode looks up an auto-created node by its Sparkplug identity, which is
// the id point rather than the description.
func spFindNode(t *testing.T, nc *nats.Conn, parent, nodeType, identity string) string {
	t.Helper()

	nodes, err := client.GetNodes(nc, parent, "all", nodeType, false)
	if err != nil {
		return ""
	}

	for _, n := range nodes {
		if p, ok := n.Points.Find(data.PointTypeID, ""); ok && p.Txt() == identity {
			return n.ID
		}
	}

	return ""
}

func spNodePoint(t *testing.T, nc *nats.Conn, nodeID, typ, key string) (data.Point, bool) {
	t.Helper()

	nodes, err := client.GetNodes(nc, "all", nodeID, "", false)
	if err != nil || len(nodes) < 1 {
		return data.Point{}, false
	}

	return nodes[0].Points.Find(typ, key)
}

// TestSparkplugClient publishes birth, data, and death messages the way a
// gateway does, and checks the node structure that comes out of them.
func TestSparkplugClient(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	mq := client.Mqtt{
		ID:          "mqtt-sparkplug",
		Parent:      root.ID,
		Description: "Plant 03 Sparkplug",
		Sparkplug:   true,
	}

	if err := client.SendNodeType(nc, mq, "test"); err != nil {
		t.Fatal("Error creating mqtt node: ", err)
	}

	pub := mqttTestBroker(t, "sparkplug-publisher")

	nbirth := spPayload(t,
		spMetric("bdSeq", 1, 0),
		spMetric("Node Control/Rebirth", 2, 0),
		spMetric("uptime", 3, 120))

	var groupID, edgeID, deviceID string

	publishUntil(t, pub, "spBv1.0/"+spGroup+"/NBIRTH/"+spEdge, nbirth,
		"the group and edge node to be created", func() bool {
			groupID = spFindNode(t, nc, mq.ID, data.NodeTypeSparkplugGroup, spGroup)
			if groupID == "" {
				return false
			}

			edgeID = spFindNode(t, nc, groupID, data.NodeTypeSparkplugNode, spEdge)
			return edgeID != ""
		})

	// the group carries a tag so queries can select on it the way they select
	// on a hand-set tag
	waitFor(t, mqttTestTimeout, "the group tag", func() bool {
		p, ok := spNodePoint(t, nc, groupID, data.PointTypeTag, data.NodeTypeSparkplugGroup)
		return ok && p.Txt() == spGroup
	})

	waitFor(t, mqttTestTimeout, "the edge node metric from the birth", func() bool {
		p, ok := spNodePoint(t, nc, edgeID, data.PointTypeValue, "uptime")
		return ok && p.Val() == 120
	})

	waitFor(t, mqttTestTimeout, "the edge node to be marked online", func() bool {
		p, ok := spNodePoint(t, nc, edgeID, data.PointTypeSysState, "")
		return ok && p.Txt() == data.PointValueSysStateOnline
	})

	dbirth := spPayload(t,
		spMetric("tank_level", 10, 42.1),
		spMetric("pump_rpm", 11, 1730))

	publishUntil(t, pub, "spBv1.0/"+spGroup+"/DBIRTH/"+spEdge+"/"+spDevice, dbirth,
		"the device node and its metrics", func() bool {
			deviceID = spFindNode(t, nc, edgeID, data.NodeTypeSparkplugDevice, spDevice)
			if deviceID == "" {
				return false
			}

			p, ok := spNodePoint(t, nc, deviceID, data.PointTypeValue, "tank_level")
			return ok && p.Val() == 42.1
		})

	// a tag a person adds must survive the refresh a later birth performs
	sendPoint(t, nc, deviceID, data.NewPointString(data.PointTypeTag, "machine", "press-3"))

	// DDATA references metrics by the alias the birth assigned
	ddata := spPayload(t, spMetric("", 10, 55.5))

	publishUntil(t, pub, "spBv1.0/"+spGroup+"/DDATA/"+spEdge+"/"+spDevice, ddata,
		"the aliased metric update", func() bool {
			p, ok := spNodePoint(t, nc, deviceID, data.PointTypeValue, "tank_level")
			return ok && p.Val() == 55.5
		})

	// a second birth refreshes rather than duplicating. Its metrics carry a
	// fresh timestamp, since a point older than what the node already holds
	// is merged as history rather than as the current value.
	dbirth2 := spPayload(t,
		spMetric("tank_level", 10, 43.2),
		spMetric("pump_rpm", 11, 1740))

	publishUntil(t, pub, "spBv1.0/"+spGroup+"/DBIRTH/"+spEdge+"/"+spDevice, dbirth2,
		"the values from the second birth", func() bool {
			p, ok := spNodePoint(t, nc, deviceID, data.PointTypeValue, "tank_level")
			return ok && p.Val() == 43.2
		})

	devices, err := client.GetNodes(nc, edgeID, "all", data.NodeTypeSparkplugDevice, false)
	if err != nil {
		t.Fatal("Error getting device nodes: ", err)
	}

	if len(devices) != 1 {
		t.Fatalf("expected one device node after the rebirth, got %v", len(devices))
	}

	if p, ok := spNodePoint(t, nc, deviceID, data.PointTypeTag, "machine"); !ok || p.Txt() != "press-3" {
		t.Fatal("a tag set by hand did not survive the rebirth refresh")
	}

	// a device death marks the node offline and keeps it
	if err := pub.Publish("spBv1.0/"+spGroup+"/DDEATH/"+spEdge+"/"+spDevice,
		spPayload(t), 1, false); err != nil {
		t.Fatal("Error publishing: ", err)
	}

	waitFor(t, mqttTestTimeout, "the device to be marked offline", func() bool {
		p, ok := spNodePoint(t, nc, deviceID, data.PointTypeSysState, "")
		return ok && p.Txt() == data.PointValueSysStateOffline
	})

	// and a later birth brings it back online
	publishUntil(t, pub, "spBv1.0/"+spGroup+"/DBIRTH/"+spEdge+"/"+spDevice, dbirth2,
		"the device to come back online", func() bool {
			p, ok := spNodePoint(t, nc, deviceID, data.PointTypeSysState, "")
			return ok && p.Txt() == data.PointValueSysStateOnline
		})
}

// TestSparkplugRebirth covers the two halves of alias handling: data for a
// device Simple IoT has never seen asks the edge node for a rebirth, and the
// alias assignment a birth carries survives a restart, so the nodes are
// matched by identity rather than created again.
func TestSparkplugRebirth(t *testing.T) {
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
		ID:          "mqtt-rebirth",
		Parent:      root.ID,
		Description: "Plant 03 Sparkplug",
		Sparkplug:   true,
	}

	if err := client.SendNodeType(nc, mq, "test"); err != nil {
		t.Fatal("Error creating mqtt node: ", err)
	}

	// the rebirth request goes out on the subject the broker maps the NCMD
	// topic to, which is how it reaches the gateway with no MQTT client here
	ncmdSubject, err := data.MQTTTopicToSubject("spBv1.0/" + spGroup + "/NCMD/" + spEdge)
	if err != nil {
		t.Fatal("Error converting NCMD topic: ", err)
	}

	ncmd, err := nc.SubscribeSync(ncmdSubject)
	if err != nil {
		t.Fatal("Error subscribing to NCMD: ", err)
	}

	pub := mqttTestBroker(t, "rebirth-publisher")

	// data for a device that has never published a birth certificate, which
	// is what a gateway already running when Simple IoT starts looks like
	ddata := spPayload(t, spMetric("", 10, 55.5))

	var rebirth *nats.Msg

	waitFor(t, mqttTestTimeout, "a rebirth request", func() bool {
		if err := pub.Publish("spBv1.0/"+spGroup+"/DDATA/"+spEdge+"/"+spDevice,
			ddata, 0, false); err != nil {
			t.Fatal("Error publishing: ", err)
		}

		m, err := ncmd.NextMsg(time.Millisecond * 200)
		if err != nil {
			return false
		}

		rebirth = m
		return true
	})

	var p sparkplug.Payload

	if err := proto.Unmarshal(rebirth.Data, &p); err != nil {
		t.Fatal("Error decoding the rebirth request: ", err)
	}

	if len(p.GetMetrics()) != 1 || p.GetMetrics()[0].GetName() != "Node Control/Rebirth" {
		t.Fatalf("unexpected rebirth payload: %v", p.String())
	}

	if !p.GetMetrics()[0].GetBooleanValue() {
		t.Fatal("the rebirth request did not ask for a rebirth")
	}

	// the gateway answers with its birth certificates
	dbirth := spPayload(t, spMetric("tank_level", 10, 42.1))

	var groupID, edgeID, deviceID string

	publishUntil(t, pub, "spBv1.0/"+spGroup+"/DBIRTH/"+spEdge+"/"+spDevice, dbirth,
		"the Sparkplug node structure", func() bool {
			groupID = spFindNode(t, nc, mq.ID, data.NodeTypeSparkplugGroup, spGroup)
			if groupID == "" {
				return false
			}

			edgeID = spFindNode(t, nc, groupID, data.NodeTypeSparkplugNode, spEdge)
			if edgeID == "" {
				return false
			}

			deviceID = spFindNode(t, nc, edgeID, data.NodeTypeSparkplugDevice, spDevice)
			return deviceID != ""
		})

	stop()
	restarted = true

	nc2, _, stop2, err := server.TestServerOptsKeepStore(server.TestServerOptions)
	if err != nil {
		t.Fatal("Error restarting test server: ", err)
	}

	defer stop2()

	pub2 := mqttTestBroker(t, "rebirth-publisher-2")

	// the alias assignment came back with the edge node, so aliased data
	// resolves without another round trip through the gateway
	ddata2 := spPayload(t, spMetric("", 10, 66.6))

	publishUntil(t, pub2, "spBv1.0/"+spGroup+"/DDATA/"+spEdge+"/"+spDevice, ddata2,
		"the aliased update after the restart", func() bool {
			pt, ok := spNodePoint(t, nc2, deviceID, data.PointTypeValue, "tank_level")
			return ok && pt.Val() == 66.6
		})

	groups, err := client.GetNodes(nc2, mq.ID, "all", data.NodeTypeSparkplugGroup, false)
	if err != nil {
		t.Fatal("Error getting group nodes: ", err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected one group node after the restart, got %v", len(groups))
	}

	devices, err := client.GetNodes(nc2, edgeID, "all", data.NodeTypeSparkplugDevice, false)
	if err != nil {
		t.Fatal("Error getting device nodes: ", err)
	}

	if len(devices) != 1 {
		t.Fatalf("expected one device node after the restart, got %v", len(devices))
	}
}
