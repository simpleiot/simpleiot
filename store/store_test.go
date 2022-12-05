package store_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

func TestStoreSimple(t *testing.T) {
	_, _, stop, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()
}

func TestStoreUp(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	_ = root

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	chUpPoints := make(chan data.Points)

	sub, err := nc.Subscribe("up.root.>", func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			fmt.Println("Error decoding points")
			return
		}

		chUpPoints <- points
	})

	if err != nil {
		t.Fatal("sub error: ", err)
	}

	defer sub.Unsubscribe()

	err = client.SendNodePoint(nc, root.ID, data.Point{Type: data.PointTypeDescription,
		Text: "rootly"}, false)

	if err != nil {
		t.Fatal("Error sending point: ", err)
	}

stopFor:
	for {
		select {
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for description change")
		case p := <-chUpPoints:
			if p[0].Type != data.PointTypeDescription {
				continue
			}
			break stopFor // all is well
		}
	}
}

func TestStoreMultiplePoints(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	_ = nc
	_ = root

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	// add client node
	u := client.User{
		ID:        uuid.New().String(),
		Parent:    root.ID,
		FirstName: "cliff",
		LastName:  "brake",
	}

	err = client.SendNodeType(nc, u, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// set up watcher for node
	getNode, stopWatcher, err := client.NodeWatcher[client.User](nc, u.ID, u.Parent)
	if err != nil {
		t.Fatal("Error setting up node watcher")
	}

	defer stopWatcher()

	// wait for node to be populated
	start := time.Now()
	for {
		cur := getNode()
		if cur.ID == u.ID {
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for user node")
		}
		<-time.After(time.Millisecond * 10)
	}

	err = client.SendNodePoints(nc, u.ID, data.Points{
		{Type: data.PointTypeFirstName, Text: "Cliff"},
		{Type: data.PointTypeLastName, Text: "Brake"},
	}, true)

	if err != nil {
		t.Fatal("send points failed: ", err)
	}

	time.Sleep(time.Millisecond * 10)
	updated := getNode()

	if updated.FirstName != "Cliff" {
		t.Fatal("first name not updated: ", updated.FirstName)
	}

	if updated.LastName != "Brake" {
		t.Fatal("last name not updated: ", updated.LastName)
	}
}

func TestGetNatsURI(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	_ = nc
	_ = root

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	uri, _, err := client.GetNatsURI(nc)

	if err != nil {
		t.Fatal(err)
	}

	if uri != server.TestServerOptions.NatsServer {
		t.Fatal("Did not get expected URI: ", uri)
	}
}

func TestDontAllowDeleteRootNode(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	_ = root

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	err = client.SendEdgePoint(nc, root.ID, "root", data.Point{Type: data.PointTypeTombstone,
		Value: 1}, true)

	if err == nil {
		t.Fatal("sending edge point should have returned an error")
	}

	nodes, err := client.GetNodes(nc, "root", root.ID, "", false)

	if err != nil {
		t.Fatal("Error getting node: ", err)
	}

	if len(nodes) < 1 {
		t.Fatal("Root node was deleted")
	}
}
