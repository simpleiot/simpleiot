package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpleiot/simpleiot/data"
)

// newPromTestClient builds a client that can exercise everything except the
// point publishing, which the end-to-end test in metrics-prom-client_test.go
// covers.
func newPromTestClient(config Metrics) *MetricsClient {
	if config.MaxSeries == 0 {
		config.MaxSeries = promDefaultMaxSeries
	}

	return &MetricsClient{
		config:       config,
		promCounters: make(map[string]float64),
		promSkipped:  make(map[string]bool),
	}
}

// findPoint returns the point with the given type and key
func findPoint(pts data.Points, typ, key string) (data.Point, bool) {
	for _, p := range pts {
		if p.Type == typ && p.Key == key {
			return p, true
		}
	}

	return data.Point{}, false
}

// serveExposition starts a test endpoint serving the given body
func serveExposition(t *testing.T, body string) *httptest.Server {
	t.Helper()

	s := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if a := r.Header.Get("Accept"); !strings.Contains(a, "text/plain") {
				t.Errorf("Expected an Accept header asking for text, got %q", a)
			}

			w.Header().Set("Content-Type",
				"text/plain; version=0.0.4; charset=utf-8")
			fmt.Fprint(w, body)
		}))

	t.Cleanup(s.Close)

	return s
}

func TestPromScrape(t *testing.T) {
	s := serveExposition(t, `# TYPE myapp_requests_total counter
myapp_requests_total{method="post",code="200"} 1027
myapp_queue_depth 17
`)

	m := newPromTestClient(Metrics{URI: s.URL, Period: 10})

	samples, bad, err := m.promScrape()
	if err != nil {
		t.Fatal("Error scraping:", err)
	}

	if bad != 0 {
		t.Errorf("Expected no bad lines, got %v", bad)
	}

	if len(samples) != 2 {
		t.Fatalf("Expected 2 samples, got %+v", samples)
	}
}

func TestPromScrapeErrors(t *testing.T) {
	t.Run("non-200 response", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "nope", http.StatusInternalServerError)
			}))

		defer s.Close()

		m := newPromTestClient(Metrics{URI: s.URL, Period: 10})

		if _, _, err := m.promScrape(); err == nil {
			t.Error("Expected an error from a 500 response")
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(
			func(http.ResponseWriter, *http.Request) {}))
		uri := s.URL
		s.Close()

		m := newPromTestClient(Metrics{URI: uri, Period: 10})

		if _, _, err := m.promScrape(); err == nil {
			t.Error("Expected an error when nothing is listening")
		}
	})

	t.Run("body over the limit is truncated", func(t *testing.T) {
		// one sample per line, enough of them to run past promMaxBody
		line := "myapp_padding_bytes{k=\"" + strings.Repeat("x", 200) + "\"} 1\n"

		s := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				for n := 0; n < promMaxBody/len(line)+100; n++ {
					fmt.Fprint(w, line)
				}
			}))

		defer s.Close()

		m := newPromTestClient(Metrics{URI: s.URL, Period: 10})

		samples, _, err := m.promScrape()
		if err != nil {
			t.Fatal("Error scraping:", err)
		}

		if len(samples) > promMaxBody/len(line)+1 {
			t.Errorf("Expected the body to be truncated, got %v samples",
				len(samples))
		}
	})
}

func TestPromPointsMapping(t *testing.T) {
	m := newPromTestClient(Metrics{})

	samples, _, err := parseExposition(strings.NewReader(
		`# TYPE myapp_requests_total counter
myapp_requests_total{method="post",code="200"} 1027
myapp_queue_depth 17
myapp_d_bucket{le="0.005"} 24054
`))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	pts, errMsg := m.promPoints(samples)
	if errMsg != "" {
		t.Errorf("Expected no error message, got %q", errMsg)
	}

	for _, exp := range []struct {
		typ string
		key string
		val float64
	}{
		{"myapp_requests_total", "code=200,method=post", 1027},
		{"myapp_queue_depth", "", 17},
		{"myapp_d_bucket", "le=0_005", 24054},
	} {
		p, ok := findPoint(pts, exp.typ, exp.key)
		if !ok {
			t.Errorf("Expected a point %q key %q, got %+v", exp.typ, exp.key, pts)
			continue
		}

		if p.Val() != exp.val {
			t.Errorf("Expected %q value %v, got %v", exp.typ, exp.val, p.Val())
		}
	}

	// no delta on the first scrape of a counter, and never for a gauge
	if _, ok := findPoint(pts, "myapp_requests_total"+data.CounterDeltaSuffix,
		"code=200,method=post"); ok {
		t.Error("Expected no delta point on the first scrape")
	}
}

// Every point a scrape publishes has to survive the store, which rejects a
// type or key holding a period, whitespace, or a NATS wildcard.
func TestPromPointsArePublishable(t *testing.T) {
	m := newPromTestClient(Metrics{})

	for _, name := range []string{
		"testdata/prom-client-golang.txt",
		"testdata/prom-node-exporter.txt",
	} {
		f := mustOpen(t, name)
		defer f.Close()

		samples, _, err := parseExposition(f)
		if err != nil {
			t.Fatal("Error parsing fixture:", err)
		}

		pts, _ := m.promPoints(samples)

		if len(pts) == 0 {
			t.Fatalf("Expected %v to produce points", name)
		}

		for _, p := range pts {
			if err := p.CheckSubjectTokens(); err != nil {
				t.Errorf("Point from %v is not publishable: %v", name, err)
			}
		}
	}
}

func TestPromPointsPrefix(t *testing.T) {
	m := newPromTestClient(Metrics{Prefix: "myapp_"})

	f := mustOpen(t, "testdata/prom-client-golang.txt")
	defer f.Close()

	samples, _, err := parseExposition(f)
	if err != nil {
		t.Fatal("Error parsing fixture:", err)
	}

	pts, _ := m.promPoints(samples)

	if len(pts) == 0 {
		t.Fatal("Expected the prefix to keep the application's own metrics")
	}

	for _, p := range pts {
		if !strings.HasPrefix(p.Type, "myapp_") {
			t.Errorf("Expected only myapp_ metrics, got %q", p.Type)
		}
	}

	if _, ok := findPoint(pts, "myapp_queue_depth", ""); !ok {
		t.Error("Expected the prefix to keep myapp_queue_depth")
	}
}

// A metric whose name matches a node configuration point type would be merged
// into the client config by Run, rewriting the node's own settings.
func TestPromPointsReservedNames(t *testing.T) {
	m := newPromTestClient(Metrics{})

	samples, _, err := parseExposition(strings.NewReader(
		`period 42
description 1
disabled 1
uri 3
myapp_hits 7
`))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	pts, _ := m.promPoints(samples)

	if len(pts) != 1 || pts[0].Type != "myapp_hits" {
		t.Fatalf("Expected only myapp_hits to be published, got %+v", pts)
	}

	// the skip is logged once per name, not once per scrape
	if !m.promSkipped["period"] {
		t.Error("Expected the skipped name to be recorded")
	}
}

func TestPromPointsCounterDelta(t *testing.T) {
	m := newPromTestClient(Metrics{CounterDelta: true})

	const key = "code=200,method=post"

	scrape := func(val float64) data.Points {
		t.Helper()

		in := fmt.Sprintf(
			"# TYPE myapp_requests_total counter\n"+
				"myapp_requests_total{method=\"post\",code=\"200\"} %v\n", val)

		samples, _, err := parseExposition(strings.NewReader(in))
		if err != nil {
			t.Fatal("Error parsing:", err)
		}

		pts, _ := m.promPoints(samples)

		return pts
	}

	deltaType := "myapp_requests_total" + data.CounterDeltaSuffix

	// scrape 1: nothing to subtract from yet
	pts := scrape(1027)
	if _, ok := findPoint(pts, deltaType, key); ok {
		t.Error("Expected no delta on the first scrape")
	}

	if p, ok := findPoint(pts, "myapp_requests_total", key); !ok || p.Val() != 1027 {
		t.Errorf("Expected the raw counter to be published, got %+v", pts)
	}

	// scrape 2 and 3: the difference
	for _, test := range []struct{ val, delta float64 }{
		{1053, 26},
		{1061, 8},
	} {
		pts = scrape(test.val)

		p, ok := findPoint(pts, deltaType, key)
		if !ok {
			t.Fatalf("Expected a delta point, got %+v", pts)
		}

		if p.Val() != test.delta {
			t.Errorf("Expected delta %v, got %v", test.delta, p.Val())
		}
	}

	// scrape 4: the application restarted, so the counter went backwards and
	// the current value is the count since that restart
	pts = scrape(12)

	p, ok := findPoint(pts, deltaType, key)
	if !ok {
		t.Fatalf("Expected a delta point after a restart, got %+v", pts)
	}

	if p.Val() != 12 {
		t.Errorf("Expected the delta after a restart to be 12, got %v", p.Val())
	}
}

func TestPromPointsNoDeltaForGauges(t *testing.T) {
	m := newPromTestClient(Metrics{CounterDelta: true})

	in := "# TYPE myapp_queue_depth gauge\nmyapp_queue_depth %v\n"

	for _, val := range []int{17, 23} {
		samples, _, err := parseExposition(
			strings.NewReader(fmt.Sprintf(in, val)))
		if err != nil {
			t.Fatal("Error parsing:", err)
		}

		pts, _ := m.promPoints(samples)

		if _, ok := findPoint(pts,
			"myapp_queue_depth"+data.CounterDeltaSuffix, ""); ok {
			t.Error("Expected no delta point for a gauge")
		}
	}
}

func TestPromPointsCounterDeltaDisabled(t *testing.T) {
	m := newPromTestClient(Metrics{CounterDelta: false})

	in := "# TYPE myapp_requests_total counter\nmyapp_requests_total %v\n"

	for _, val := range []int{5, 9} {
		samples, _, err := parseExposition(
			strings.NewReader(fmt.Sprintf(in, val)))
		if err != nil {
			t.Fatal("Error parsing:", err)
		}

		pts, _ := m.promPoints(samples)

		if _, ok := findPoint(pts,
			"myapp_requests_total"+data.CounterDeltaSuffix, ""); ok {
			t.Error("Expected no delta point when counterDelta is off")
		}
	}
}

// A counter no longer served should not be remembered forever, or an
// application that drops a label value leaks an entry per value.
func TestPromPointsForgetsMissingCounters(t *testing.T) {
	m := newPromTestClient(Metrics{CounterDelta: true})

	samples, _, err := parseExposition(strings.NewReader(
		"# TYPE myapp_a counter\nmyapp_a 1\n# TYPE myapp_b counter\nmyapp_b 2\n"))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	m.promPoints(samples)

	if len(m.promCounters) != 2 {
		t.Fatalf("Expected 2 remembered counters, got %v", len(m.promCounters))
	}

	samples, _, err = parseExposition(strings.NewReader(
		"# TYPE myapp_a counter\nmyapp_a 3\n"))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	m.promPoints(samples)

	if len(m.promCounters) != 1 {
		t.Errorf("Expected the dropped counter to be forgotten, got %v",
			m.promCounters)
	}
}

func TestPromPointsMaxSeries(t *testing.T) {
	var b strings.Builder

	for n := 0; n < 50; n++ {
		fmt.Fprintf(&b, "myapp_hits{n=\"%03d\"} %v\n", n, n)
	}

	samples, _, err := parseExposition(strings.NewReader(b.String()))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	m := newPromTestClient(Metrics{MaxSeries: 10})

	pts, errMsg := m.promPoints(samples)

	if len(pts) != 10 {
		t.Fatalf("Expected 10 points, got %v", len(pts))
	}

	if errMsg == "" {
		t.Error("Expected a truncated scrape to report an error")
	}

	// the same series must survive a repeat scrape rather than flapping
	pts2, _ := m.promPoints(samples)

	for i := range pts {
		if pts[i].Key != pts2[i].Key {
			t.Fatalf("Expected the same series to survive, got %q then %q",
				pts[i].Key, pts2[i].Key)
		}
	}

	if pts[0].Key != "n=000" || pts[9].Key != "n=009" {
		t.Errorf("Expected the cap to keep a deterministic set, got %q..%q",
			pts[0].Key, pts[9].Key)
	}
}

func TestPromResetForgetsCounters(t *testing.T) {
	m := newPromTestClient(Metrics{CounterDelta: true})

	samples, _, err := parseExposition(strings.NewReader(
		"# TYPE myapp_a counter\nmyapp_a 100\n"))
	if err != nil {
		t.Fatal("Error parsing:", err)
	}

	m.promPoints(samples)
	m.promReset()

	pts, _ := m.promPoints(samples)

	if _, ok := findPoint(pts, "myapp_a"+data.CounterDeltaSuffix, ""); ok {
		t.Error("Expected no delta after a reset, since the endpoint changed")
	}
}
