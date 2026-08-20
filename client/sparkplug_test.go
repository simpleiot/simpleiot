package client

import (
	"encoding/hex"
	"testing"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/pb/sparkplug"
	"google.golang.org/protobuf/proto"
)

func TestParseSparkplugTopic(t *testing.T) {
	tests := []struct {
		topic   string
		want    sparkplugTopic
		wantErr bool
	}{
		{topic: "spBv1.0/plant-03/NBIRTH/ignition-edge",
			want: sparkplugTopic{Group: "plant-03", MsgType: "NBIRTH", EdgeNode: "ignition-edge"}},
		{topic: "spBv1.0/plant-03/NDATA/ignition-edge",
			want: sparkplugTopic{Group: "plant-03", MsgType: "NDATA", EdgeNode: "ignition-edge"}},
		{topic: "spBv1.0/plant-03/NDEATH/ignition-edge",
			want: sparkplugTopic{Group: "plant-03", MsgType: "NDEATH", EdgeNode: "ignition-edge"}},
		{topic: "spBv1.0/plant-03/DBIRTH/ignition-edge/press-1",
			want: sparkplugTopic{Group: "plant-03", MsgType: "DBIRTH", EdgeNode: "ignition-edge", Device: "press-1"}},
		{topic: "spBv1.0/plant-03/DDATA/ignition-edge/press-1",
			want: sparkplugTopic{Group: "plant-03", MsgType: "DDATA", EdgeNode: "ignition-edge", Device: "press-1"}},
		{topic: "spBv1.0/plant-03/DDEATH/ignition-edge/press-1",
			want: sparkplugTopic{Group: "plant-03", MsgType: "DDEATH", EdgeNode: "ignition-edge", Device: "press-1"}},
		{topic: "spBv1.0/STATE/primary-host",
			want: sparkplugTopic{MsgType: "STATE", HostID: "primary-host"}},

		// a device level on an edge node message, and the reverse
		{topic: "spBv1.0/plant-03/NBIRTH/ignition-edge/press-1", wantErr: true},
		{topic: "spBv1.0/plant-03/DBIRTH/ignition-edge", wantErr: true},
		{topic: "spBv1.0/plant-03/REBIRTH/ignition-edge", wantErr: true},
		{topic: "spBv2.0/plant-03/NBIRTH/ignition-edge", wantErr: true},
		{topic: "spBv1.0//NBIRTH/ignition-edge", wantErr: true},
		{topic: "plant-03/kepware/press", wantErr: true},
		{topic: "spBv1.0", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.topic, func(t *testing.T) {
			got, err := parseSparkplugTopic(tc.topic)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", got)
				}
				return
			}

			if err != nil {
				t.Fatal("unexpected error: ", err)
			}

			if got != tc.want {
				t.Fatalf("expected %+v, got %+v", tc.want, got)
			}
		})
	}
}

// sparkplugGolden is a Sparkplug B payload carrying two metrics, encoded with
// the vendored Eclipse Tahu definition. Decoding it guards the field numbers
// the definition assigns, which is what a gateway on the other end encodes
// against.
const sparkplugGolden = "0880d095ffbc3112200a0a74616e6b5f6c6576656c100a1880d095ffbc31200a69cdcccccccc0c454" +
	"0121b0a057374617465100b1880d095ffbc31200c7a0772756e6e696e671803"

func TestSparkplugDecodeGolden(t *testing.T) {
	d, err := hex.DecodeString(sparkplugGolden)
	if err != nil {
		t.Fatal("Error decoding the fixture: ", err)
	}

	var p sparkplug.Payload

	if err := proto.Unmarshal(d, &p); err != nil {
		t.Fatal("Error decoding the payload: ", err)
	}

	if p.GetTimestamp() != 1700000000000 {
		t.Errorf("expected the payload timestamp, got %v", p.GetTimestamp())
	}

	if p.GetSeq() != 3 {
		t.Errorf("expected sequence 3, got %v", p.GetSeq())
	}

	metrics := p.GetMetrics()

	if len(metrics) != 2 {
		t.Fatalf("expected two metrics, got %v", len(metrics))
	}

	if metrics[0].GetName() != "tank_level" || metrics[0].GetAlias() != 10 ||
		metrics[0].GetDoubleValue() != 42.1 {
		t.Errorf("unexpected first metric: %v", metrics[0].String())
	}

	if metrics[1].GetName() != "state" || metrics[1].GetAlias() != 11 ||
		metrics[1].GetStringValue() != "running" {
		t.Errorf("unexpected second metric: %v", metrics[1].String())
	}

	s := &sparkplugState{}

	pt, ok := s.metricPoint(metrics[0], metrics[0].GetName(), p.GetTimestamp())
	if !ok || pt.Key != "tank_level" || pt.Val() != 42.1 {
		t.Errorf("unexpected point from the first metric: %v", pt)
	}
}

func spTestMetric(name string, value any) *sparkplug.Payload_Metric {
	m := &sparkplug.Payload_Metric{Name: &name}

	switch v := value.(type) {
	case uint32:
		m.Value = &sparkplug.Payload_Metric_IntValue{IntValue: v}
	case uint64:
		m.Value = &sparkplug.Payload_Metric_LongValue{LongValue: v}
	case float32:
		m.Value = &sparkplug.Payload_Metric_FloatValue{FloatValue: v}
	case float64:
		m.Value = &sparkplug.Payload_Metric_DoubleValue{DoubleValue: v}
	case bool:
		m.Value = &sparkplug.Payload_Metric_BooleanValue{BooleanValue: v}
	case string:
		m.Value = &sparkplug.Payload_Metric_StringValue{StringValue: v}
	}

	return m
}

func TestSparkplugMetricPoint(t *testing.T) {
	s := &sparkplugState{}

	tests := []struct {
		desc     string
		metric   *sparkplug.Payload_Metric
		key      string
		dataType data.PointDataType
		value    float64
		text     string
		skip     bool
	}{
		{desc: "int", metric: spTestMetric("pump_rpm", uint32(1730)),
			key: "pump_rpm", dataType: data.PointDataTypeInt, value: 1730},
		{desc: "long", metric: spTestMetric("counter", uint64(90000)),
			key: "counter", dataType: data.PointDataTypeInt, value: 90000},
		{desc: "float", metric: spTestMetric("tank_level", float32(42.5)),
			key: "tank_level", dataType: data.PointDataTypeFloat, value: 42.5},
		{desc: "double", metric: spTestMetric("pressure", 101.325),
			key: "pressure", dataType: data.PointDataTypeFloat, value: 101.325},
		{desc: "boolean true", metric: spTestMetric("running", true),
			key: "running", dataType: data.PointDataTypeInt, value: 1},
		{desc: "boolean false", metric: spTestMetric("running", false),
			key: "running", dataType: data.PointDataTypeInt, value: 0},
		{desc: "string", metric: spTestMetric("state", "idle"),
			key: "state", dataType: data.PointDataTypeString, text: "idle"},
		{desc: "name with characters a subject cannot carry",
			metric: spTestMetric("Node Control/Rebirth", true),
			key:    "Node_Control/Rebirth", dataType: data.PointDataTypeInt, value: 1},
		{desc: "no value", metric: &sparkplug.Payload_Metric{}, skip: true},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			p, ok := s.metricPoint(tc.metric, tc.metric.GetName(), 0)

			if tc.skip {
				if ok {
					t.Fatalf("expected the metric to be skipped, got %v", p)
				}
				return
			}

			if !ok {
				t.Fatal("expected a point")
			}

			if p.Type != data.PointTypeValue {
				t.Errorf("expected a value point, got %v", p.Type)
			}

			if p.Key != tc.key {
				t.Errorf("expected key %q, got %q", tc.key, p.Key)
			}

			if p.DataType != tc.dataType {
				t.Errorf("expected data type %v, got %v", tc.dataType, p.DataType)
			}

			if tc.text != "" {
				if p.Txt() != tc.text {
					t.Errorf("expected %q, got %q", tc.text, p.Txt())
				}
				return
			}

			if p.Val() != tc.value {
				t.Errorf("expected %v, got %v", tc.value, p.Val())
			}
		})
	}
}

// TestSparkplugMetricTimestamp checks that the payload timestamp is used when
// a metric does not carry one of its own.
func TestSparkplugMetricTimestamp(t *testing.T) {
	s := &sparkplugState{}

	const payloadMS = 1_700_000_000_000
	const metricMS = 1_700_000_001_000

	m := spTestMetric("tank_level", 42.0)

	p, ok := s.metricPoint(m, m.GetName(), payloadMS)
	if !ok {
		t.Fatal("expected a point")
	}

	if p.Time.UnixMilli() != payloadMS {
		t.Errorf("expected the payload timestamp, got %v", p.Time.UnixMilli())
	}

	ts := uint64(metricMS)
	m.Timestamp = &ts

	p, ok = s.metricPoint(m, m.GetName(), payloadMS)
	if !ok {
		t.Fatal("expected a point")
	}

	if p.Time.UnixMilli() != metricMS {
		t.Errorf("expected the metric timestamp, got %v", p.Time.UnixMilli())
	}
}
