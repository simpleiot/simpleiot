package client

import (
	"math"
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

func TestMqttJSONPath(t *testing.T) {
	payload := []byte(`{"value": 42.1, "a": {"b": 7}, "list": [10, 20], "name": "press-1"}`)

	tests := []struct {
		desc    string
		path    string
		value   float64
		text    string
		wantErr bool
	}{
		{desc: "top level", path: "$.value", value: 42.1},
		{desc: "no leading dollar", path: "value", value: 42.1},
		{desc: "nested", path: "$.a.b", value: 7},
		{desc: "array index", path: "$.list[0]", value: 10},
		{desc: "array index 1", path: "$.list[1]", value: 20},
		{desc: "string", path: "$.name", text: "press-1"},
		{desc: "missing", path: "$.nothere", wantErr: true},
		{desc: "missing nested", path: "$.a.c", wantErr: true},
		{desc: "index past the end", path: "$.list[5]", wantErr: true},
		{desc: "field of a number", path: "$.value.more", wantErr: true},
		{desc: "index into an object", path: "$.a[0]", wantErr: true},
		{desc: "empty field", path: "$..a", wantErr: true},
		{desc: "unterminated index", path: "$.list[0", wantErr: true},
		{desc: "selects nothing", path: "$", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			sub := MqttSub{Path: tc.path}

			pts, err := sub.points(payload)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %v", pts)
				}
				return
			}

			if err != nil {
				t.Fatal("unexpected error: ", err)
			}

			if len(pts) != 1 {
				t.Fatalf("expected one point, got %v", pts)
			}

			if tc.text != "" {
				if pts[0].Txt() != tc.text {
					t.Fatalf("expected text %q, got %q", tc.text, pts[0].Txt())
				}
				return
			}

			if math.Abs(pts[0].Val()-tc.value) > 1e-9 {
				t.Fatalf("expected %v, got %v", tc.value, pts[0].Val())
			}
		})
	}
}

func TestMqttScaleAndOffset(t *testing.T) {
	sub := MqttSub{Path: "$.value", Scale: 0.1, Offset: 2}

	pts, err := sub.points([]byte(`{"value": 100}`))
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	if math.Abs(pts[0].Val()-12) > 1e-9 {
		t.Fatalf("expected 12, got %v", pts[0].Val())
	}

	// an unset scale must not read every message as zero
	sub = MqttSub{Path: "$.value"}

	pts, err = sub.points([]byte(`{"value": 100}`))
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	if math.Abs(pts[0].Val()-100) > 1e-9 {
		t.Fatalf("expected 100, got %v", pts[0].Val())
	}
}

func TestMqttBlankPath(t *testing.T) {
	t.Run("bare number", func(t *testing.T) {
		pts, err := MqttSub{}.points([]byte(`42.1`))
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		if len(pts) != 1 || pts[0].Key != "" || math.Abs(pts[0].Val()-42.1) > 1e-9 {
			t.Fatalf("unexpected points: %v", pts)
		}
	})

	t.Run("bare string", func(t *testing.T) {
		pts, err := MqttSub{}.points([]byte(`"running"`))
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		if len(pts) != 1 || pts[0].Txt() != "running" {
			t.Fatalf("unexpected points: %v", pts)
		}
	})

	t.Run("object", func(t *testing.T) {
		pts, err := MqttSub{}.points(
			[]byte(`{"tank_level": 42.1, "pump_rpm": 1730, "state": "run", "alarm": true}`))
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		if len(pts) != 4 {
			t.Fatalf("expected four points, got %v", pts)
		}

		want := map[string]any{
			"alarm":      1.0,
			"pump_rpm":   1730.0,
			"state":      "run",
			"tank_level": 42.1,
		}

		for _, p := range pts {
			if p.Type != data.PointTypeValue {
				t.Errorf("expected a value point, got %v", p.Type)
			}

			switch w := want[p.Key].(type) {
			case float64:
				if math.Abs(p.Val()-w) > 1e-9 {
					t.Errorf("%v: expected %v, got %v", p.Key, w, p.Val())
				}
			case string:
				if p.Txt() != w {
					t.Errorf("%v: expected %q, got %q", p.Key, w, p.Txt())
				}
			default:
				t.Errorf("unexpected point key %v", p.Key)
			}
		}
	})

	t.Run("field names are made subject safe", func(t *testing.T) {
		pts, err := MqttSub{}.points([]byte(`{"tank.level": 1}`))
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		if pts[0].Key != "tank_level" {
			t.Fatalf("expected the key to be sanitized, got %q", pts[0].Key)
		}
	})

	t.Run("null", func(t *testing.T) {
		if _, err := (MqttSub{}).points([]byte(`null`)); err == nil {
			t.Fatal("expected an error for a null payload")
		}
	})

	t.Run("not JSON", func(t *testing.T) {
		if _, err := (MqttSub{}).points([]byte(`not json`)); err == nil {
			t.Fatal("expected an error for a payload that is not JSON")
		}
	})
}
