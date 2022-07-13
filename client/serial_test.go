package client_test

import (
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
		nodes, err := client.GetNodeChildren(nc, root.ID, data.NodeTypeSerialDev, false, false)
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
		time.After(time.Second)
	}

}
