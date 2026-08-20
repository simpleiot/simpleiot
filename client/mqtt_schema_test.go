package client

import (
	"reflect"
	"testing"
)

func TestParseMqttSchema(t *testing.T) {
	tests := []struct {
		schema  string
		labels  []string
		filter  string
		wantErr bool
	}{
		{schema: "{site}/{device}", labels: []string{"site", "device"}, filter: "+/+/#"},
		{schema: "{device}", labels: []string{"device"}, filter: "+/#"},
		{schema: "{site}/{gateway}/{device}",
			labels: []string{"site", "gateway", "device"}, filter: "+/+/+/#"},
		{schema: "plant/{site}/{device}", labels: []string{"site", "device"}, filter: "plant/+/+/#"},
		{schema: "{site}/tags/{device}", labels: []string{"site", "device"}, filter: "+/tags/+/#"},
		{schema: "{site}/{device}/#", labels: []string{"site", "device"}, filter: "+/+/#"},

		{schema: "", wantErr: true},
		{schema: "{}/{device}", wantErr: true},
		{schema: "{site}//{device}", wantErr: true},
		{schema: "{site}/{site}", wantErr: true},
		{schema: "{site}/#/{device}", wantErr: true},
		{schema: "plant/line3", wantErr: true},
		{schema: "{site}/+/{device}", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.schema, func(t *testing.T) {
			s, err := parseMqttSchema(tc.schema)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", s)
				}
				return
			}

			if err != nil {
				t.Fatal("unexpected error: ", err)
			}

			if !reflect.DeepEqual(s.labels(), tc.labels) {
				t.Errorf("expected labels %v, got %v", tc.labels, s.labels())
			}

			if s.filter != tc.filter {
				t.Errorf("expected filter %v, got %v", tc.filter, s.filter)
			}
		})
	}
}

func TestMqttSchemaMatch(t *testing.T) {
	tests := []struct {
		desc   string
		schema string
		topic  string
		values []string
		rest   []string
		ok     bool
	}{
		{desc: "exact depth", schema: "{site}/{gateway}/{device}",
			topic:  "plant-07/kepware-l3/press",
			values: []string{"plant-07", "kepware-l3", "press"}, ok: true},
		{desc: "one level deeper", schema: "{site}/{gateway}/{device}",
			topic:  "plant-07/kepware-l3/press/tank_level",
			values: []string{"plant-07", "kepware-l3", "press"},
			rest:   []string{"tank_level"}, ok: true},
		{desc: "two levels deeper", schema: "{site}/{device}",
			topic:  "plant-07/press/tank/level",
			values: []string{"plant-07", "press"},
			rest:   []string{"tank", "level"}, ok: true},
		{desc: "literal prefix", schema: "plant/{site}/{device}",
			topic: "plant/07/press", values: []string{"07", "press"}, ok: true},
		{desc: "literal prefix that does not match", schema: "plant/{site}/{device}",
			topic: "site/07/press"},
		{desc: "too shallow", schema: "{site}/{gateway}/{device}",
			topic: "plant-07/kepware-l3"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			s, err := parseMqttSchema(tc.schema)
			if err != nil {
				t.Fatal("unexpected error: ", err)
			}

			values, rest, ok := s.match(tc.topic)

			if ok != tc.ok {
				t.Fatalf("expected match %v, got %v", tc.ok, ok)
			}

			if !ok {
				return
			}

			if !reflect.DeepEqual(values, tc.values) {
				t.Errorf("expected values %v, got %v", tc.values, values)
			}

			if len(rest) != len(tc.rest) {
				t.Fatalf("expected remainder %v, got %v", tc.rest, rest)
			}

			for i := range rest {
				if rest[i] != tc.rest[i] {
					t.Errorf("expected remainder %v, got %v", tc.rest, rest)
				}
			}
		})
	}
}

func TestMqttSchemaPoints(t *testing.T) {
	tests := []struct {
		desc    string
		payload string
		rest    []string
		keys    []string
		values  []float64
	}{
		{desc: "scalar payload at the schema depth",
			payload: `42.1`, keys: []string{""}, values: []float64{42.1}},
		{desc: "value object one level deeper",
			payload: `{"value": 42.1}`, rest: []string{"tank_level"},
			keys: []string{"tank_level"}, values: []float64{42.1}},
		{desc: "value object at the schema depth",
			payload: `{"value": 42.1}`, keys: []string{""}, values: []float64{42.1}},
		{desc: "object fields become the key",
			payload: `{"pump_rpm": 1730, "tank_level": 42.1}`,
			keys:    []string{"pump_rpm", "tank_level"}, values: []float64{1730, 42.1}},
		{desc: "remaining levels join with the field name",
			payload: `{"pump_rpm": 1730}`, rest: []string{"press", "motor"},
			keys: []string{"press/motor/pump_rpm"}, values: []float64{1730}},
		{desc: "level values are made subject safe",
			payload: `{"value": 1}`, rest: []string{"tank.level"},
			keys: []string{"tank_level"}, values: []float64{1}},
		{desc: "a value field beside another keeps its name",
			payload: `{"value": 1, "units": 2}`,
			keys:    []string{"units", "value"}, values: []float64{2, 1}},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			pts, err := mqttSchemaPoints([]byte(tc.payload), tc.rest)
			if err != nil {
				t.Fatal("unexpected error: ", err)
			}

			if len(pts) != len(tc.keys) {
				t.Fatalf("expected %v points, got %v", len(tc.keys), pts)
			}

			for i, p := range pts {
				if p.Key != tc.keys[i] {
					t.Errorf("expected key %q, got %q", tc.keys[i], p.Key)
				}

				if p.Val() != tc.values[i] {
					t.Errorf("%v: expected %v, got %v", p.Key, tc.values[i], p.Val())
				}
			}
		})
	}
}

func TestMqttFilterSubjects(t *testing.T) {
	tests := []struct {
		filter   string
		subjects []string
	}{
		{"plant/line3/tank", []string{"plant.line3.tank"}},
		{"plant/+/tank", []string{"plant.*.tank"}},
		// the remainder wildcard also matches the level above it, which the
		// NATS wildcard it converts to does not, so both are needed
		{"plant/#", []string{"plant.>", "plant"}},
		{"+/+/#", []string{"*.*.>", "*.*"}},
	}

	for _, tc := range tests {
		got, err := mqttFilterSubjects(tc.filter)
		if err != nil {
			t.Errorf("%v: unexpected error: %v", tc.filter, err)
			continue
		}

		if !reflect.DeepEqual(got, tc.subjects) {
			t.Errorf("%v: expected %v, got %v", tc.filter, tc.subjects, got)
		}
	}
}

func TestMqttFilterMatch(t *testing.T) {
	tests := []struct {
		filter string
		topic  string
		match  bool
	}{
		{"a/b/c", "a/b/c", true},
		{"a/b/c", "a/b/d", false},
		{"a/+/c", "a/b/c", true},
		{"a/+/c", "a/b/c/d", false},
		{"a/#", "a/b/c", true},
		{"a/#", "a", true},
		{"a/#", "b/c", false},
		{"#", "a/b", true},
		{"a/b", "a/b/c", false},
		{"a/b/c", "a/b", false},
	}

	for _, tc := range tests {
		got := mqttFilterMatch(tc.filter, tc.topic)
		if got != tc.match {
			t.Errorf("%v against %v: expected %v, got %v", tc.filter, tc.topic, tc.match, got)
		}
	}
}
