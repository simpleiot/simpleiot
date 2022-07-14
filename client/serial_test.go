package client_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/server"
	"github.com/simpleiot/simpleiot/test"
)

func TestSerial(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	_ = nc

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	fifo, err := test.NewFifoA("serialfifo")
	if err != nil {
		t.Fatal("Error starting fifo: ", err)
	}

	serialTest := client.SerialDev{
		ID:          uuid.New().String(),
		Parent:      root.ID,
		Description: "test serial",
		Port:        "serialfifo",
	}

	// hydrate database with test data
	err = client.SendNodeType(nc, serialTest)
	if err != nil {
		t.Fatal("Error sending node: ", err)
	}

	// set up watcher for node
	getNode, stopWatcher, err := client.NodeWatcher[client.SerialDev](nc, serialTest.ID, serialTest.Parent)
	if err != nil {
		t.Fatal("Error setting up node watcher")
	}

	defer stopWatcher()

	start := time.Now()

	// wait for node to be populated
	for {
		cur := getNode()
		if cur.ID == serialTest.ID {
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for serial node")
		}
		<-time.After(time.Millisecond * 10)
	}

	// send a packet to the serial client
	testLog := "Hi there"
	_, err = fifo.Write([]byte(testLog))

	if err != nil {
		t.Error("Error sending packet to fifo: ", err)
	}

	// wait for a packet to be received
	start = time.Now()
	for {
		cur := getNode()
		if cur.Rx > 0 && cur.Log == testLog {
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for log packet")
		}
		<-time.After(time.Millisecond * 100)
	}
}
