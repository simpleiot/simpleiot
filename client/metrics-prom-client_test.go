package client_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// promTestTimeout is how long an assertion waits for a scrape to travel all the
// way around: a node created in the store, the client started, an HTTP fetch,
// and points back out. It allows for several scrape periods.
const promTestTimeout = 10 * time.Second

// nodePoint reads a point off the metrics node. A keyless point is published
// with an empty key and stored under "0", which is the convention throughout
// SIOT, so the two are looked up interchangeably here.
func nodePoint(t *testing.T, nc *nats.Conn, parent, id, typ, key string) (data.Point, bool) {
	t.Helper()

	nodes, err := client.GetNodes(nc, parent, id, "", false)
	if err != nil || len(nodes) < 1 {
		return data.Point{}, false
	}

	if key == "" {
		key = "0"
	}

	for _, p := range nodes[0].Points {
		k := p.Key
		if k == "" {
			k = "0"
		}

		if p.Type == typ && k == key && p.Tombstone%2 == 0 {
			return p, true
		}
	}

	return data.Point{}, false
}

// TestPrometheusMetricsClient exercises a scrape end to end: a node is created,
// the client fetches an endpoint, and the samples arrive as points on the node.
func TestPrometheusMetricsClient(t *testing.T) {
	// the counter climbs on each scrape so the delta has something to report
	var hits int64

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			n := atomic.AddInt64(&hits, 10)

			fmt.Fprintf(w, `# TYPE myapp_requests_total counter
myapp_requests_total{method="post",code="200"} %v
# TYPE myapp_queue_depth gauge
myapp_queue_depth 17
# TYPE myapp_request_duration_seconds histogram
myapp_request_duration_seconds_bucket{le="0.005"} 24054
myapp_request_duration_seconds_bucket{le="+Inf"} 144320
# TYPE go_goroutines gauge
go_goroutines 41
`, n)
		}))

	defer srv.Close()

	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	m := client.Metrics{
		ID:          "metrics-prom",
		Parent:      root.ID,
		Description: "test scrape",
		Type:        data.PointValuePrometheus,
		URI:         srv.URL,
		Period:      1,
		Prefixes:    []string{"myapp_"},
	}

	if err := client.SendNodeType(nc, m, "test"); err != nil {
		t.Fatal("Error creating metrics node: ", err)
	}

	// the scrape defaults are published back to the node so they are visible
	// and editable rather than implicit
	waitFor(t, promTestTimeout, "scrape defaults to be published", func() bool {
		p, ok := nodePoint(t, nc, root.ID, m.ID, data.PointTypeMaxSeries, "")
		if !ok || p.Val() != 200 {
			return false
		}

		p, ok = nodePoint(t, nc, root.ID, m.ID, data.PointTypeCounterDelta, "")

		return ok && p.Val() == 1
	})

	waitFor(t, promTestTimeout, "a gauge to arrive as a point", func() bool {
		p, ok := nodePoint(t, nc, root.ID, m.ID, "myapp_queue_depth", "")
		return ok && p.Val() == 17
	})

	// labels are sorted by name, not left in the order the endpoint served
	waitFor(t, promTestTimeout, "a labeled counter to arrive", func() bool {
		p, ok := nodePoint(t, nc, root.ID, m.ID,
			"myapp_requests_total", "code=200,method=post")
		return ok && p.Val() > 0
	})

	// a bucket boundary holds a period, which the store would reject, so the
	// key has to arrive sanitized
	waitFor(t, promTestTimeout, "a histogram bucket to arrive", func() bool {
		p, ok := nodePoint(t, nc, root.ID, m.ID,
			"myapp_request_duration_seconds_bucket", "le=0_005")
		return ok && p.Val() == 24054
	})

	waitFor(t, promTestTimeout, "an +Inf bucket to arrive", func() bool {
		_, ok := nodePoint(t, nc, root.ID, m.ID,
			"myapp_request_duration_seconds_bucket", "le=+Inf")
		return ok
	})

	// the second scrape onward publishes the change since the last one
	waitFor(t, promTestTimeout, "a counter delta to arrive", func() bool {
		p, ok := nodePoint(t, nc, root.ID, m.ID,
			"myapp_requests_total"+data.CounterDeltaSuffix, "code=200,method=post")
		return ok && p.Val() == 10
	})

	// the prefix keeps the application's own metrics and drops the rest
	if _, ok := nodePoint(t, nc, root.ID, m.ID, "go_goroutines", ""); ok {
		t.Error("Expected the prefix to drop go_goroutines")
	}

	// an endpoint that stops answering sets an error on the node rather than
	// leaving stale readings in place
	srv.Close()

	waitFor(t, promTestTimeout, "an error point after the endpoint goes away",
		func() bool {
			p, ok := nodePoint(t, nc, root.ID, m.ID, data.PointTypeError, "")
			return ok && p.Txt() != ""
		})
}

// A metric named after a node configuration point type would be merged into the
// client config, rewriting the node's own settings.
func TestPrometheusMetricsReservedNames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `period 42
description 1
myapp_hits 7
`)
		}))

	defer srv.Close()

	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	m := client.Metrics{
		ID:          "metrics-prom-reserved",
		Parent:      root.ID,
		Description: "reserved names",
		Type:        data.PointValuePrometheus,
		URI:         srv.URL,
		Period:      1,
	}

	if err := client.SendNodeType(nc, m, "test"); err != nil {
		t.Fatal("Error creating metrics node: ", err)
	}

	waitFor(t, promTestTimeout, "the scrape to publish a metric", func() bool {
		p, ok := nodePoint(t, nc, root.ID, m.ID, "myapp_hits", "")
		return ok && p.Val() == 7
	})

	// by the time the metric above has arrived, a colliding name would have
	// arrived with it
	get, getStop, err := client.NodeWatcher[client.Metrics](nc, m.ID, m.Parent)
	if err != nil {
		t.Fatal("Error watching metrics node: ", err)
	}

	defer getStop()

	if p := get().Period; p != 1 {
		t.Errorf("Expected the period to still be 1, got %v", p)
	}

	if d := get().Description; d != "reserved names" {
		t.Errorf("Expected the description to be unchanged, got %q", d)
	}
}
