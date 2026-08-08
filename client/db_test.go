package client_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// The db client speaks the InfluxDB v2 write API. VictoriaMetrics
// accepts that protocol on /api/v2/write and runs as a single binary
// with no setup, so the test starts a throwaway instance itself and
// verifies through the Prometheus-style query API. Install
// victoria-metrics to enable this test; it is skipped otherwise.

const vmAddr = "127.0.0.1:18428"

// startVictoriaMetrics runs a VictoriaMetrics instance on a temporary
// data directory, returning a stop function. The test is skipped if
// the binary is not installed.
func startVictoriaMetrics(t *testing.T) func() {
	t.Helper()

	bin, err := exec.LookPath("victoria-metrics")
	if err != nil {
		bin, err = exec.LookPath("victoria-metrics-prod")
	}
	if err != nil {
		t.Skip("victoria-metrics binary not found, skipping db test")
	}

	cmd := exec.Command(bin,
		"-storageDataPath", t.TempDir(),
		"-httpListenAddr", vmAddr,
		// freshly written samples are visible to instant queries
		// immediately instead of after the default 30s offset
		"-search.latencyOffset", "0s",
	)
	if err := cmd.Start(); err != nil {
		t.Fatal("Error starting victoria-metrics:", err)
	}

	stop := func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}

	start := time.Now()
	for {
		resp, err := http.Get("http://" + vmAddr + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Since(start) > 10*time.Second {
			stop()
			t.Fatal("victoria-metrics did not become ready")
		}
		time.Sleep(50 * time.Millisecond)
	}

	return stop
}

// vmQueryValue runs an instant query and returns the first result
// value. ok is false when there is no result yet.
func vmQueryValue(query string) (float64, bool) {
	// recently ingested data sits in memory buffers; force a flush so
	// the query sees it (an endpoint VictoriaMetrics provides for
	// exactly this kind of use)
	resp, err := http.Get("http://" + vmAddr + "/internal/force_flush")
	if err == nil {
		_ = resp.Body.Close()
	}

	resp, err = http.Get("http://" + vmAddr + "/api/v1/query?query=" +
		url.QueryEscape(query))
	if err != nil {
		return 0, false
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false
	}

	var r struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Value []any `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &r) != nil {
		return 0, false
	}
	if r.Status != "success" || len(r.Data.Result) == 0 ||
		len(r.Data.Result[0].Value) != 2 {
		return 0, false
	}
	s, ok := r.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, false
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%g", &v); err != nil {
		return 0, false
	}
	return v, true
}

func TestDb(t *testing.T) {
	stopVM := startVictoriaMetrics(t)
	defer stopVM()

	// Start up a SIOT test server for this test
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	dbConfig := client.Db{
		ID:          "ID-db",
		Parent:      root.ID,
		Description: "vm test db",
		URI:         "http://" + vmAddr,
		Org:         "siot-test",
		Bucket:      "test",
		AuthToken:   "not-used",
	}

	// set up Db client
	err = client.SendNodeType(nc, dbConfig, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// let the client start and create its durable stream consumers, so
	// the point below is delivered to them
	time.Sleep(time.Second)

	// write a numeric point and verify it lands in the external store.
	// The Influx line protocol maps measurement "points" field "value"
	// to metric points_value, and the node.id tag becomes a label of
	// the same name (quoted in the query, since it is not a standard
	// identifier). String fields (text) are not stored by
	// VictoriaMetrics, so the assertion uses a float point.
	p := data.NewPointFloat(data.PointTypeValue, "", 42)
	p.Origin = "test"
	err = client.SendNodePoint(nc, dbConfig.ID, p, true)
	if err != nil {
		t.Fatal("Error sending point:", err)
	}

	// the influx client batches writes (~1s); poll until the value is
	// queryable. last_over_time is used because a bare instant query
	// only looks back one step, which can miss a just-written sample.
	query := `last_over_time(points_value{"node.id"="ID-db", type="value"}[5m])`
	waitFor(t, 15*time.Second, "point in VictoriaMetrics", func() bool {
		v, ok := vmQueryValue(query)
		return ok && v == 42
	})
}
