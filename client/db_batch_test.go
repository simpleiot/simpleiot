package client_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestDbBatching checks that stream-delivered points reach the database
// in batches rather than one request per point.
func TestDbBatching(t *testing.T) {
	var requests, lines atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			n := len(strings.Split(strings.TrimSpace(string(body)), "\n"))
			requests.Add(1)
			lines.Add(int64(n))
			w.WriteHeader(http.StatusNoContent)
		}))
	defer srv.Close()

	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	dbConfig := client.Db{
		ID:          "ID-db-batch",
		Parent:      root.ID,
		Description: "batching test db",
		URI:         srv.URL,
		Org:         "siot-test",
		Bucket:      "test",
		AuthToken:   "not-used",
	}

	if err := client.SendNodeType(nc, dbConfig, "test"); err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// let the client start and create its durable stream consumers
	time.Sleep(time.Second)

	const count = 300
	for i := range count {
		p := data.NewPointFloat(data.PointTypeValue, "", float64(i))
		p.Origin = "test"
		p.Time = time.Now()
		if err := client.SendNodePoint(nc, dbConfig.ID, p, false); err != nil {
			t.Fatal("Error sending point:", err)
		}
	}

	waitFor(t, 30*time.Second, "all points written", func() bool {
		return lines.Load() >= count
	})

	t.Logf("%v points delivered in %v requests", lines.Load(), requests.Load())
	if requests.Load() > count/4 {
		t.Errorf("expected points to be batched, got %v requests for %v points",
			requests.Load(), lines.Load())
	}
}
