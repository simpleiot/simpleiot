package client_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

func TestSync(t *testing.T) {
	// Start up a SIOT test servers for this test
	ncU, _, stopU, err := server.TestServer("2")

	if err != nil {
		t.Fatal("Error starting upstream test server: ", err)
	}

	defer stopU()

	ncD, rootD, stopD, err := server.TestServer()

	if err != nil {
		t.Fatal("Error starting upstream test server: ", err)
	}

	defer stopD()

	fmt.Println("**** create sync node")
	sync := client.Sync{
		ID:          "sync-id",
		Parent:      rootD.ID,
		Description: "sync to up",
		URI:         server.TestServerOptions2.NatsServer,
	}

	err = client.SendNodeType(ncD, sync, "test")
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// make sure device node gets sync'd upstream
	start := time.Now()
	for {
		if time.Since(start) > 500*time.Millisecond {
			t.Fatal("device node not synced")
		}

		nodes, err := client.GetNodes(ncU, "none", rootD.ID, "", false)
		if err != nil {
			continue
		}

		if len(nodes) > 0 {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}

	fmt.Println("**** update description down")
	err = client.SendNodePoint(ncD, rootD.ID, data.Point{Type: data.PointTypeDescription, Text: "set down"}, true)
	if err != nil {
		t.Fatal("error sending node point: ", err)
	}

	start = time.Now()
	for {
		if time.Since(start) > 500*time.Millisecond {
			t.Fatal("description not propagated upstream")
		}

		nodes, err := client.GetNodesType[client.Device](ncU, "none", rootD.ID)
		if err != nil {
			continue
		}

		if len(nodes) > 0 {
			if nodes[0].Description == "set down" {
				break
			}
		}

		time.Sleep(time.Millisecond * 10)
	}

	fmt.Println("**** update description up")
	err = client.SendNodePoint(ncU, rootD.ID, data.Point{Type: data.PointTypeDescription, Text: "set up"}, true)
	if err != nil {
		t.Fatal("error sending node point: ", err)
	}

	start = time.Now()
	for {
		if time.Since(start) > 500*time.Millisecond {
			t.Fatal("description not propagated downstream")
		}

		nodes, err := client.GetNodesType[client.Device](ncD, "none", rootD.ID)
		if err != nil {
			continue
		}

		if len(nodes) > 0 {
			if nodes[0].Description == "set up" {
				break
			}
		}

		time.Sleep(time.Millisecond * 10)
	}

	fmt.Println("**** create node down")
	varD := client.Variable{ID: "varDown", Parent: rootD.ID, Description: "varDown"}
	err = client.SendNodeType(ncD, varD, "test")
	if err != nil {
		t.Fatal("Error sending var1: ", err)
	}

	start = time.Now()
	for {
		if time.Since(start) > 500*time.Millisecond {
			t.Fatal("var1 not propagated upstream")
		}

		nodes, err := client.GetNodesType[client.Variable](ncU, "none", "varDown")
		if err != nil {
			continue
		}

		if len(nodes) > 0 {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}

	fmt.Println("**** create node up")
	varU := client.Variable{ID: "varUp", Parent: rootD.ID, Description: "varUp"}
	err = client.SendNodeType(ncU, varU, "test")
	if err != nil {
		t.Fatal("Error sending varU: ", err)
	}

	start = time.Now()
	for {
		if time.Since(start) > 500*time.Millisecond {
			t.Fatal("var2 not propagated downstream")
		}

		nodes, err := client.GetNodesType[client.Variable](ncU, "none", "varUp")
		if err != nil {
			continue
		}

		if len(nodes) > 0 {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}

	fmt.Println("**** delete node down")
	err = client.SendEdgePoint(ncD, varD.ID, rootD.ID, data.Point{Type: data.PointTypeTombstone, Value: 1}, true)
	if err != nil {
		t.Fatal("error sending node point: ", err)
	}

	start = time.Now()
	for {
		if time.Since(start) > 500*time.Millisecond {
			t.Fatal("varD delete not propagated upstream")
		}

		nodes, err := client.GetNodesType[client.Variable](ncU, rootD.ID, varD.ID)
		if err != nil {
			t.Fatal(err)
		}

		if len(nodes) <= 0 {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}

	fmt.Println("**** delete node up")
	err = client.SendEdgePoint(ncU, varU.ID, rootD.ID, data.Point{Type: data.PointTypeTombstone, Value: 1}, true)
	if err != nil {
		t.Fatal("error sending node point: ", err)
	}

	start = time.Now()
	for {
		if time.Since(start) > 500*time.Millisecond {
			t.Fatal("varU not propagated downstream")
		}

		nodes, err := client.GetNodesType[client.Variable](ncD, rootD.ID, varU.ID)
		if err != nil {
			t.Fatal(err)
		}

		if len(nodes) <= 0 {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}

	fmt.Println("sync test finished")
}
