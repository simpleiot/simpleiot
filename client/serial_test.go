package client_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/test"
)

func TestSerial(t *testing.T) {
	nc, root, stop, err := test.Server()
	_ = nc

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	serialTest := client.SerialDev{
		Parent:      root.ID,
		Description: "test serial",
		Port:        "fifo",
	}

	ne, err := data.Encode(serialTest)
	if err != nil {
		t.Fatal("Error encoding node: ", err)
	}

	// hydrate database with test data
	err = client.SendNode(nc, ne)

	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	start := time.Now()

	// wait for node to be populated
	for {
		nodes, err := client.GetNodeChildrenType[client.SerialDev](nc, root.ID)
		if err != nil {
			t.Fatal("Error getting node children: ", err)
		}
		if len(nodes) > 0 {
			serialTest.ID = nodes[0].ID
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for serial node")
		}
		<-time.After(time.Millisecond * 10)
	}

	// wait for a packet to be received
	start = time.Now()
	for {
		nodes, err := client.GetNodeChildrenType[client.SerialDev](nc, root.ID)
		if err != nil {
			t.Fatal("Error getting node children: ", err)
		}
		if len(nodes) > 0 {
			fmt.Printf("CLIFF: serial nodes: %+v\n", nodes)
			if nodes[0].Rx > 0 {
				break
			}
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for rx packet")
		}
		<-time.After(time.Millisecond * 100)
	}

}
