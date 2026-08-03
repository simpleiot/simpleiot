package server

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/system"
)

// reportVersions records the application and OS versions on the root node, so
// that the version an instance is running can be seen in the UI and by an
// upstream instance. It runs once at startup and updates only what changed.
func reportVersions(nc *nats.Conn, appVersion, osVersionField string) {
	rootNode, err := client.GetRootNode(nc)
	if err != nil {
		log.Println("Error getting root node to report versions:", err)
		return
	}

	appVer, ok := rootNode.Points.Find(data.PointTypeVersionApp, "")
	if !ok || appVer.Txt() != appVersion {
		log.Println("Setting app version:", appVersion)
		err := client.SendNodePoint(nc, rootNode.ID,
			data.NewPointString(data.PointTypeVersionApp, "", appVersion), true)
		if err != nil {
			log.Println("Error setting app version:", err)
		}
	}

	osVer, err := system.ReadOSVersion(osVersionField)
	if err != nil {
		log.Println("Error reading OS version:", err)
		return
	}

	log.Println("OS version:", osVer)

	osVerStored, ok := rootNode.Points.Find(data.PointTypeVersionOS, "")
	if !ok || osVer.String() != osVerStored.Txt() {
		log.Println("Setting os version:", osVer)
		err := client.SendNodePoint(nc, rootNode.ID,
			data.NewPointString(data.PointTypeVersionOS, "", osVer.String()), true)
		if err != nil {
			log.Println("Error setting OS version:", err)
		}
	}
}
