package client_test

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"testing"
	"time"

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
	// channel during this test. The A side is written by the
	// this test and simulates MCU writes. The B side is written by the serial
	// client.
	fifo, err := test.NewFifoA("serialfifo")
	if err != nil {
		t.Fatal("Error starting fifo: ", err)
	}

	fifoW := client.NewCobsWrapper(fifo, 500)
	defer fifoW.Close()

	serialTest := client.SerialDev{
		ID:          "ID-serial",
		Parent:      root.ID,
		Description: "test serial",
		// when Port is set to the magic value of "serialfifo", the serial
		// client opens a unix fifo instead of a real serial port. This allows
		// us to send/receive data to/from serial client during
		// testing without needing real serial hardware.
		Port: "serialfifo",
		// You can set debug to increase debugging level
		Debug: 4,
	}

	// hydrate database with test data
	err = client.SendNodeType(nc, serialTest, "test")
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

	readCh := make(chan []byte)
	var readData []byte

	mcuReadSerial := func() {
		buf := make([]byte, 200)
		c, err := fifoW.Read(buf)
		if err != nil {
			fmt.Println("Error reading response from client: ", err)
		}
		buf = buf[:c]
		readCh <- buf
	}

	// dump timeSync package from client
	go mcuReadSerial()

	select {
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for timeSync packet")
	case <-readCh:
		// all is well
	}

	// send an ascii log message to the serial client
	log.Println("Sending log test message")
	buf := bytes.NewBuffer([]byte{})
	_, _ = buf.Write([]byte{1})
	sub := make([]byte, 16)
	copy(sub, []byte("log"))
	_, _ = buf.Write(sub)
	testLog := "Hi there"
	_, _ = buf.Write([]byte(testLog))

	_, err = fifoW.Write(buf.Bytes())
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

	// send a uptime point to the serial client over serial channel
	uptimeTest := 5523
	seq := byte(10)
	uptimePts := data.Points{
		{Type: data.PointTypeUptime, Value: float64(uptimeTest)},
	}

	uptimePacket, err := client.SerialEncode(seq, "", uptimePts)
	if err != nil {
		t.Fatal("Error encoding serial packet: ", err)
	}

	_, err = fifoW.Write(uptimePacket)
	if err != nil {
		t.Fatal("Error writing pb data to fifo: ", err)
	}

	// wait for point to show up in node
	start = time.Now()
	for {
		cur := getNode()
		if cur.Uptime == uptimeTest {
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for uptime to get set")
		}
		<-time.After(time.Millisecond * 100)
	}

	// check for ack response from serial client
	go mcuReadSerial()

	select {
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for serial response")
	case readData = <-readCh:
		// all is well
	}

	seqR, subjectR, payload, err := client.SerialDecode(readData)
	if err != nil {
		t.Error("Error in response: ", err)
	}

	pointsR, err := data.PbDecodeSerialPoints(payload)
	if err != nil {
		t.Errorf("Error decoding serial payload: %v", err)
	}

	if seq != seqR {
		t.Error("Sequence in response did not match: ", seq, seqR)
	}

	if subjectR != "ack" {
		t.Error("Subject in response is not ack, is: ", subjectR)
	}

	if len(pointsR) != 0 {
		t.Error("should be no points in response")
	}

	// test sending points to MCU
	pumpSetting := data.Point{Type: "pumpSetting", Value: 233.5, Origin: root.ID}
	err = client.SendNodePoint(nc, serialTest.ID, pumpSetting, true)
	if err != nil {
		t.Fatal("Error sending pumpSetting point: ", err)
	}

	// the above should trigger a serial packet to get sent to MCU, look for it now
	go mcuReadSerial()

	select {
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for pump setting at MCU")
	case readData = <-readCh:
		// all is well
	}

	_, _, payload, err = client.SerialDecode(readData)
	if err != nil {
		t.Error("Error in response: ", err)
	}

	pointsR, err = data.PbDecodeSerialPoints(payload)
	if err != nil {
		t.Errorf("Error decoding serial payload: %v", err)
	}

	if len(pointsR) < 1 {
		t.Error("Did not receive pointsR point")
	} else {
		if pointsR[0].Value != pumpSetting.Value {
			t.Error("Error in pump setting received by MCU")
		}
	}
}

func TestSerialLargeMessage(t *testing.T) {
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

	fifoW := client.NewCobsWrapper(fifo, 500)
	defer fifoW.Close()

	serialTest := client.SerialDev{
		ID:          "ID-serial",
		Parent:      root.ID,
		Description: "test serial",
		// when Port is set to the magic value of "serialfifo", the serial
		// client opens a unix fifo instead of a real serial port. This allows
		// us to send/receive data to/from serial client during
		// testing without needing real serial hardware.
		Port: "serialfifo",
	}

	// hydrate database with test data
	err = client.SendNodeType(nc, serialTest, "test")
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

	var points data.Points

	for i := 0; i < 10; i++ {
		points = append(points, data.Point{Type: "testPoint",
			Key: strconv.Itoa(i), Value: float64(i * 2)})
	}

	packet, err := client.SerialEncode(1, "", points)
	if err != nil {
		t.Fatal("Error encoding serial packet: ", err)
	}

	fmt.Println("len(packet): ", len(packet))
	fmt.Println("Rx: ", getNode().Rx)

	_, err = fifoW.Write(packet)
	if err != nil {
		t.Fatal("Error writing pb data to fifo: ", err)
	}

	// wait for point to show up in node
	start = time.Now()
	for {
		cur := getNode()
		if cur.Rx >= 1 {
			break
		}
		if time.Since(start) > time.Second {
			t.Fatal("Timeout waiting for packet to get set")
		}
		<-time.After(time.Millisecond * 100)
	}
}
