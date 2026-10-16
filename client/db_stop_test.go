package client_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// TestDbStopDuringWrite checks that a database slow to answer does not
// hold up a shutdown, and that the points in the write are kept. The
// client stops while a write is in flight; the instance is then started
// again on the same store with a database that answers, and the points
// are written there.
func TestDbStopDuringWrite(t *testing.T) {
	// writes block here until the test lets them go, which stands in for
	// a database that has stopped answering mid-request
	release := make(chan struct{})
	var inFlight atomic.Bool

	slow := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			inFlight.Store(true)
			<-release
			w.WriteHeader(http.StatusNoContent)
		}))
	defer slow.Close()

	opts := server.TestServerOptions
	nc, root, stop, err := server.TestServerOpts(opts)
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	dbConfig := client.Db{
		ID:          "ID-db-stop",
		Parent:      root.ID,
		Description: "stop during write test db",
		URI:         slow.URL,
		Org:         "siot-test",
		Bucket:      "test",
		AuthToken:   "not-used",
	}

	if err := client.SendNodeType(nc, dbConfig, "test"); err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// let the client start and create its durable stream consumers
	time.Sleep(time.Second)

	p := data.NewPointFloat(data.PointTypeValue, "stop-test", 42)
	p.Origin = "test"
	p.Time = time.Now()
	if err := client.SendNodePoint(nc, dbConfig.ID, p, true); err != nil {
		t.Fatal("Error sending point:", err)
	}

	waitFor(t, 15*time.Second, "a write to reach the database", func() bool {
		return inFlight.Load()
	})

	// the client manager gives a client five seconds to stop before it
	// gives up and logs, so stopping well inside that shows the write was
	// ended rather than waited out
	start := time.Now()
	stop()
	elapsed := time.Since(start)

	close(release)

	if elapsed > 4*time.Second {
		t.Errorf("stopping took %v, which is long enough for the write to have been waited out", elapsed)
	}

	// the write never completed, so the points must still be in the
	// stream. Start the instance again on the same store, with a database
	// that answers this time.
	var written atomic.Int64

	quick := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			written.Add(1)
			w.WriteHeader(http.StatusNoContent)
		}))
	defer quick.Close()

	nc2, _, stop2, err := server.TestServerOptsKeepStore(opts)
	if err != nil {
		t.Fatal("Error restarting test server: ", err)
	}
	defer stop2()

	dbConfig.URI = quick.URL
	if err := client.SendNodeType(nc2, dbConfig, "test"); err != nil {
		t.Fatal("Error sending node: ", err)
	}

	waitFor(t, 3*time.Minute, "the points to be written after the restart", func() bool {
		return written.Load() > 0
	})
}
