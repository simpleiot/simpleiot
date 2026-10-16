package server

import (
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// TestReportVersionsRecordsAppVersion checks the version an instance runs
// reaches the root node, which is where the UI and an upstream read it.
func TestReportVersionsRecordsAppVersion(t *testing.T) {
	nc, _, stopServer := testServerNoProvisioning(t)
	defer stopServer()

	reportVersions(make(chan struct{}), nc, "1.2.3", "")

	root, err := client.GetRootNode(nc)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	appVer, ok := root.Points.Find(data.PointTypeVersionApp, "")
	if !ok {
		t.Fatal("No app version on the root node")
	}

	if appVer.Txt() != "1.2.3" {
		t.Fatal("App version is: ", appVer.Txt())
	}
}

// TestReportVersionsStopsWhileRetrying checks a server that is shutting down
// is not held up by the retries, which continue until the write succeeds.
func TestReportVersionsStopsWhileRetrying(t *testing.T) {
	nc, _, stopServer := testServerNoProvisioning(t)
	defer stopServer()

	// every attempt now fails as soon as it is made
	nc.Close()

	stop := make(chan struct{})
	returned := make(chan struct{})

	go func() {
		reportVersions(stop, nc, "1.2.3", "")
		close(returned)
	}()

	// long enough for the first attempt to fail and the wait to start
	time.Sleep(100 * time.Millisecond)
	close(stop)

	// shorter than the wait before the second attempt, so returning this
	// quickly means stop was noticed rather than the wait having elapsed
	select {
	case <-returned:
	case <-time.After(versionRetryFirst / 2):
		t.Fatal("reportVersions did not return when asked to stop")
	}
}
