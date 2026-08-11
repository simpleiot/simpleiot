package client

import (
	"reflect"
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

func TestExpandKeyLabels(t *testing.T) {
	tests := []struct {
		desc string
		key  string
		exp  map[string]string
	}{
		{
			desc: "a label set expands to its labels",
			key:  "code=200,method=post",
			exp:  map[string]string{"code": "200", "method": "post"},
		},
		{
			desc: "a single label",
			key:  "le=1",
			exp:  map[string]string{"le": "1"},
		},
		{
			desc: "a bucket boundary has its period restored",
			key:  "le=0_005",
			exp:  map[string]string{"le": "0.005"},
		},
		{
			desc: "a summary quantile has its period restored",
			key:  "quantile=0_99",
			exp:  map[string]string{"quantile": "0.99"},
		},
		{
			desc: "an +Inf bucket needs no restoration",
			key:  "le=+Inf",
			exp:  map[string]string{"le": "+Inf"},
		},
		{
			desc: "only le and quantile are restored, not a version string",
			key:  "version=1_4_2",
			exp:  map[string]string{"version": "1_4_2"},
		},
		{
			desc: "a le that is not numeric is left alone",
			key:  "le=a_b",
			exp:  map[string]string{"le": "a_b"},
		},
		{
			desc: "a path keeps its slashes",
			key:  "mountpoint=/,fstype=ext4",
			exp:  map[string]string{"mountpoint": "/", "fstype": "ext4"},
		},
		{
			desc: "an empty value is dropped, as Prometheus treats it as absent",
			key:  "code=200,method=",
			exp:  map[string]string{"code": "200"},
		},

		// keys the rest of SIOT writes, none of which is a label set
		{desc: "an empty key", key: "", exp: nil},
		{desc: "a keyless point stored as 0", key: "0", exp: nil},
		{desc: "a network interface", key: "eth0", exp: nil},
		{desc: "a mount point", key: "/dev/sda", exp: nil},
		{desc: "a CPU", key: "cpu0", exp: nil},
		{desc: "a sysfs sensor name", key: "tmp451_2", exp: nil},
		{desc: "a load average window", key: "15", exp: nil},

		// malformed label sets are declined whole rather than half read
		{
			desc: "a comma inside a label value makes the key unsplittable",
			key:  "note=a,b",
			exp:  nil,
		},
		{
			desc: "a chunk with no equals sign",
			key:  "code=200,method",
			exp:  nil,
		},
		{
			desc: "a label name that is not a label name",
			key:  "co de=200",
			exp:  nil,
		},
		{
			desc: "a label name starting with a digit",
			key:  "2fast=200",
			exp:  nil,
		},
		{
			desc: "an empty label name",
			key:  "=200",
			exp:  nil,
		},
		{
			desc: "every value empty leaves nothing to write",
			key:  "code=,method=",
			exp:  nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := expandKeyLabels(test.key)

			if !reflect.DeepEqual(got, test.exp) {
				t.Errorf("Expected %v, got %v", test.exp, got)
			}
		})
	}
}

func TestPointTags(t *testing.T) {
	newClient := func(expand bool) *DbClient {
		return &DbClient{
			config:        Db{ExpandKeyLabels: expand},
			nodeCache:     newNodeCache(nil, "parent"),
			keyLabelSkips: make(map[string]bool),
		}
	}

	pt := data.Point{Type: "myapp_requests_total", Key: "code=200,method=post"}

	t.Run("expansion adds a tag per label and keeps the key", func(t *testing.T) {
		tags := newClient(true).pointTags("node1", pt)

		exp := map[string]string{
			"type":   "myapp_requests_total",
			"key":    "code=200,method=post",
			"code":   "200",
			"method": "post",
		}

		if !reflect.DeepEqual(tags, exp) {
			t.Errorf("Expected %v, got %v", exp, tags)
		}
	})

	t.Run("expansion off writes what the client wrote before", func(t *testing.T) {
		tags := newClient(false).pointTags("node1", pt)

		exp := map[string]string{
			"type": "myapp_requests_total",
			"key":  "code=200,method=post",
		}

		if !reflect.DeepEqual(tags, exp) {
			t.Errorf("Expected %v, got %v", exp, tags)
		}
	})

	// a label named type or key would overwrite a tag the client sets, so the
	// point's own type has to survive
	t.Run("a colliding label does not overwrite the client's tags", func(t *testing.T) {
		p := data.Point{Type: "myapp_hits", Key: "type=foo,key=bar,method=post"}

		tags := newClient(true).pointTags("node1", p)

		if tags["type"] != "myapp_hits" {
			t.Errorf("Expected the type tag to be the point type, got %q",
				tags["type"])
		}

		if tags["key"] != "type=foo,key=bar,method=post" {
			t.Errorf("Expected the key tag to be the whole set, got %q",
				tags["key"])
		}

		if tags["method"] != "post" {
			t.Errorf("Expected the other labels to still expand, got %v", tags)
		}
	})

	// a key that is not a label set is what every other client writes
	t.Run("a key that is not a label set adds nothing", func(t *testing.T) {
		p := data.Point{Type: data.PointTypeMetricSysNetBytesRecv, Key: "eth0"}

		tags := newClient(true).pointTags("node1", p)

		exp := map[string]string{
			"type": data.PointTypeMetricSysNetBytesRecv,
			"key":  "eth0",
		}

		if !reflect.DeepEqual(tags, exp) {
			t.Errorf("Expected %v, got %v", exp, tags)
		}
	})

	// both write paths call pointTags, so the same point cannot produce
	// different tags depending on which one carried it
	t.Run("the HR path and the point path agree", func(t *testing.T) {
		dbc := newClient(true)

		if !reflect.DeepEqual(dbc.pointTags("node1", pt), dbc.pointTags("node1", pt)) {
			t.Error("Expected the same tags for the same point")
		}
	})
}

// The bucket keys a histogram produces have to arrive as numbers, or
// histogram_quantile cannot read them.
func TestExpandKeyLabelsHistogramBuckets(t *testing.T) {
	samples, _, err := parseExposition(mustOpen(t, "testdata/prom-client-golang.txt"))
	if err != nil {
		t.Fatal("Error parsing fixture:", err)
	}

	exp := map[string]string{
		"le=0_005": "0.005",
		"le=0_05":  "0.05",
		"le=+Inf":  "+Inf",
	}

	var found int

	for _, s := range samples {
		want, ok := exp[s.key]
		if !ok {
			continue
		}

		found++

		labels := expandKeyLabels(s.key)
		if labels["le"] != want {
			t.Errorf("Expected le %q from key %q, got %q",
				want, s.key, labels["le"])
		}
	}

	if found != len(exp) {
		t.Errorf("Expected %v bucket keys in the fixture, found %v",
			len(exp), found)
	}
}
