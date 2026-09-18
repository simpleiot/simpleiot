package client

import (
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// TestDropSerialConfigPoints checks that an MCU cannot rewrite the serial
// client's own configuration.
func TestDropSerialConfigPoints(t *testing.T) {
	pts := data.Points{
		data.NewPointFloat("temp", "0", 21.5),
		data.NewPointString(data.PointTypePort, "", "/dev/ttyUSB9"),
		data.NewPointFloat(data.PointTypeSyncParent, "", 1),
		data.NewPointString(data.PointTypeHRDest, "", "some-node"),
		data.NewPointFloat(data.PointTypeDisabled, "", 1),
		data.NewPointFloat(data.PointTypeUptime, "", 100),
		data.NewPointString(data.PointTypeDescription, "", "renamed"),
	}

	kept := dropSerialConfigPoints(pts)

	if len(kept) != 2 {
		t.Fatalf("got %v points, want 2: %v", len(kept), kept)
	}
	if kept[0].Type != "temp" || kept[1].Type != data.PointTypeUptime {
		t.Errorf("wrong points kept: %v", kept)
	}

	if got := dropSerialConfigPoints(nil); len(got) != 0 {
		t.Errorf("nil in, got %v", got)
	}
}

// TestParticlePoints checks that a Particle device cannot rewrite the
// node's token or disable it.
func TestParticlePoints(t *testing.T) {
	now := time.Now()
	pts := particlePoints([]particlePoint{
		{ID: "0", Type: "temp", Value: 21.5},
		{ID: "", Type: data.PointTypeAuthToken, Value: 1},
		{ID: "", Type: data.PointTypeDisabled, Value: 1},
		{ID: "", Type: data.PointTypeDescription, Value: 1},
		{ID: "1", Type: "hum", Value: 50},
	}, now)

	if len(pts) != 2 {
		t.Fatalf("got %v points, want 2: %v", len(pts), pts)
	}
	if pts[0].Type != "temp" || pts[1].Type != "hum" {
		t.Errorf("wrong points kept: %v", pts)
	}
	if !pts[0].Time.Equal(now) {
		t.Errorf("event time not applied: %v", pts[0].Time)
	}
}

func TestMqttFilterScoped(t *testing.T) {
	for _, test := range []struct {
		filter string
		ok     bool
	}{
		{"#", false},
		{"+", false},
		{"+/#", false},
		{"+/temp", false},
		{"#/x", false},
		{"plant-07/#", true},
		{"plant-07/+/temp", true},
		{"plant-07", true},
		{"spBv1.0/#", true},
		{"$SYS/#", true},
	} {
		err := mqttFilterScoped(test.filter)
		if test.ok && err != nil {
			t.Errorf("%q: unexpected error %v", test.filter, err)
		}
		if !test.ok && err == nil {
			t.Errorf("%q: expected error", test.filter)
		}
	}

	// the check applies below the root only
	c := &MqttClient{underRoot: true}
	if err := c.scopedFilterError("#"); err != nil {
		t.Errorf("node under root refused #: %v", err)
	}
	c.underRoot = false
	if err := c.scopedFilterError("#"); err == nil {
		t.Error("node below root allowed #")
	}
}

func TestSparkplugRoomForNode(t *testing.T) {
	s := newSparkplugState(nil, "mqtt-1", "test", 0, 2)

	if err := s.roomForNode(); err != nil {
		t.Fatalf("empty state has no room: %v", err)
	}
	s.nodes["g"] = "id-g"
	if err := s.roomForNode(); err != nil {
		t.Fatalf("one below the limit has no room: %v", err)
	}
	s.nodes["g/e"] = "id-e"
	if err := s.roomForNode(); err == nil {
		t.Fatal("at the limit still has room")
	}

	// zero means the default
	s.setMaxNodes(0)
	if s.maxNodes != mqttMaxNodesDefault {
		t.Errorf("default not applied: %v", s.maxNodes)
	}
}
