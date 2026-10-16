package server

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/system"
)

const (
	// how long to wait before the first retry of a version write, and the
	// longest wait between later ones
	versionRetryFirst = 2 * time.Second
	versionRetryMax   = time.Minute
)

// reportVersions records the application and OS versions on the root node, so
// that the version an instance is running can be seen in the UI and by an
// upstream instance. It updates only what changed, and retries until it
// succeeds or stop is closed: an instance that has just started may still be
// recovering its streams, and a version reported once and then dropped would
// leave the stored version wrong until the next restart.
func reportVersions(stop <-chan struct{}, nc *nats.Conn, appVersion, osVersionField string) {
	wait := versionRetryFirst

	for {
		err := setVersions(nc, appVersion, osVersionField)
		if err == nil {
			return
		}

		log.Printf("Error reporting versions, retrying in %v: %v", wait, err)

		select {
		case <-stop:
			return
		case <-time.After(wait):
		}

		wait *= 2
		if wait > versionRetryMax {
			wait = versionRetryMax
		}
	}
}

// setVersions makes one attempt to record the versions. It reads the root node
// each time it is called, so a retry sees what an earlier attempt stored.
func setVersions(nc *nats.Conn, appVersion, osVersionField string) error {
	rootNode, err := client.GetRootNode(nc)
	if err != nil {
		return fmt.Errorf("error getting root node to report versions: %w", err)
	}

	appVer, ok := rootNode.Points.Find(data.PointTypeVersionApp, "")
	if !ok || appVer.Txt() != appVersion {
		log.Println("Setting app version:", appVersion)
		err := client.SendNodePoint(nc, rootNode.ID,
			data.NewPointString(data.PointTypeVersionApp, "", appVersion), true)
		if err != nil {
			return fmt.Errorf("error setting app version: %w", err)
		}
	}

	osVer, err := system.ReadOSVersion(osVersionField)
	if err != nil {
		// the OS does not report a version this way, which a retry will not
		// change, so the app version above is as far as this goes
		log.Println("Error reading OS version:", err)
		return nil
	}

	log.Println("OS version:", osVer)

	osVerStored, ok := rootNode.Points.Find(data.PointTypeVersionOS, "")
	if !ok || osVer.String() != osVerStored.Txt() {
		log.Println("Setting os version:", osVer)
		err := client.SendNodePoint(nc, rootNode.ID,
			data.NewPointString(data.PointTypeVersionOS, "", osVer.String()), true)
		if err != nil {
			return fmt.Errorf("error setting OS version: %w", err)
		}
	}

	return nil
}
