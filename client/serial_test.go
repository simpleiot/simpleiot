package client_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
	"github.com/simpleiot/simpleiot/test"
)

func TestSerial(t *testing.T) {
	// Start up a SIOT test server for this test
	nc, root, stop, err := server.TestServer()
	_ = nc

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	// the test.Fifo is used to emulate a serial port
	// channel during this test. The A side is used by the
	// this test, and the B side is used by the serial
	// client.
	fifo, err := test.NewFifoA("serialfifo")
	if err != nil {
		t.Fatal("Error starting fifo: ", err)
	}

	defer fifo.Close()

	serialTest := client.SerialDev{
		ID:          uuid.New().String(),
		Parent:      root.ID,
		Description: "test serial",
		// when Port is set to the magic value of "serialfifo", the serial
		// client opens a unix fifo instead of a real serial port. This allows
		// us to send/receive data to/from serial client during
		// testing without needing real serial hardware.
		Port: "serialfifo",
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

	// send an ascii log message to the serial client
	testLog := "Hi there"
	_, err = fifo.Write([]byte(testLog))
	if err != nil {
		t.Error("Error sending packet to fifo: ", err)
	}

	// wait for a packet to be received
	start = time.Now()
	for {
		cur := getNode()
		if cur.Rx == 1 && cur.Log == testLog {
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for log packet")
		}
		<-time.After(time.Millisecond * 100)
	}

	// send a point to the serial client
	uptimeTest := 5523
	gpioPts := data.Points{
		{Type: data.PointTypeUptime, Value: float64(uptimeTest)},
	}

	gpioPb, err := gpioPts.ToPb()
	if err != nil {
		t.Fatal("Error encoding points to PB: ", err)
	}

	_, err = fifo.Write(gpioPb)
	if err != nil {
		t.Fatal("Error writing pb data to fifo: ", err)
	}

	// wait for point to show up in node
	start = time.Now()
	for {
		cur := getNode()
		if cur.Rx == 2 && cur.Uptime == uptimeTest {
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for uptime to get set")
		}
		<-time.After(time.Millisecond * 100)
	}
}
