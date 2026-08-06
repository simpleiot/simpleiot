package client_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
	"github.com/simpleiot/simpleiot/test"
)

// shellTestRig stands in for an MCU on the other end of the serial link. It
// speaks lines over a unix fifo the same way a Zephyr console would.
type shellTestRig struct {
	t       *testing.T
	fifo    *client.LineWrapper
	lines   chan string
	getNode func() client.SerialDev
	nc      *nats.Conn
	nodeID  string
}

// setPoint writes a node point the way the UI does. An origin is required:
// the client manager drops points with no origin on the client's own node,
// on the grounds that the client published them itself.
func (r *shellTestRig) setPoint(typ string, val float64) {
	r.t.Helper()
	p := data.NewPointFloat(typ, "", val)
	p.Origin = "test"
	if err := client.SendNodePoint(r.nc, r.nodeID, p, true); err != nil {
		r.t.Fatalf("error sending %v point: %v", typ, err)
	}
}

// readLine returns the next line the client wrote, or fails the test.
func (r *shellTestRig) readLine(timeout time.Duration) string {
	r.t.Helper()
	select {
	case l := <-r.lines:
		return l
	case <-time.After(timeout):
		r.t.Fatal("timeout waiting for a line from the serial client")
		return ""
	}
}

// send writes a line to the client as the MCU would.
func (r *shellTestRig) send(line string) {
	r.t.Helper()
	if _, err := r.fifo.Write([]byte(line + "\r\n")); err != nil {
		r.t.Fatalf("error writing %q to fifo: %v", line, err)
	}
}

// waitFor polls the node until cond is satisfied.
func (r *shellTestRig) waitFor(what string, timeout time.Duration, cond func(client.SerialDev) bool) {
	r.t.Helper()
	start := time.Now()
	for {
		if cond(r.getNode()) {
			return
		}
		if time.Since(start) > timeout {
			r.t.Fatalf("timeout waiting for %v (node: %+v)", what, r.getNode())
		}
		<-time.After(time.Millisecond * 20)
	}
}

func setupShellRig(t *testing.T, cfg func(*client.SerialDev)) (*shellTestRig, func()) {
	t.Helper()

	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	fifo, err := test.NewFifoA("serialfifo")
	if err != nil {
		stop()
		t.Fatal("Error starting fifo: ", err)
	}

	fifoW := client.NewLineWrapper(fifo, 500)

	serialTest := client.SerialDev{
		ID:          "ID-serial-shell",
		Parent:      root.ID,
		Description: "test shell",
		Port:        "serialfifo",
		Protocol:    data.PointValueProtocolShell,
		Debug:       4,
	}
	if cfg != nil {
		cfg(&serialTest)
	}

	if err := client.SendNodeType(nc, serialTest, "test"); err != nil {
		stop()
		t.Fatal("Error sending node: ", err)
	}

	getNode, stopWatcher, err := client.NodeWatcher[client.SerialDev](nc, serialTest.ID, serialTest.Parent)
	if err != nil {
		stop()
		t.Fatal("Error setting up node watcher")
	}

	rig := &shellTestRig{
		t:       t,
		fifo:    fifoW,
		lines:   make(chan string, 64),
		getNode: getNode,
		nc:      nc,
		nodeID:  serialTest.ID,
	}

	// pump lines the client writes into a channel
	go func() {
		for {
			buf := make([]byte, 512)
			c, err := fifoW.Read(buf)
			if err != nil {
				return
			}
			rig.lines <- string(buf[:c])
		}
	}()

	rig.waitFor("node to populate", time.Second, func(n client.SerialDev) bool {
		return n.ID == serialTest.ID
	})

	return rig, func() {
		stopWatcher()
		fifoW.Close()
		stop()
	}
}

func TestSerialShellHandshake(t *testing.T) {
	rig, teardown := setupShellRig(t, nil)
	defer teardown()

	// The connect sequence turns off echo and colors so the console carries
	// only firmware output, then starts streaming and asks for the cache.
	expected := []string{
		"shell echo off",
		"shell colors off",
		"siot stream on",
		"siot dump",
	}

	for _, exp := range expected {
		got := rig.readLine(2 * time.Second)
		if got != exp {
			t.Fatalf("handshake: got %q, expected %q", got, exp)
		}
	}
}

func TestSerialShellReceivePoint(t *testing.T) {
	rig, teardown := setupShellRig(t, nil)
	defer teardown()

	// drain the handshake
	for range 4 {
		rig.readLine(2 * time.Second)
	}

	rig.send("pt uptime 0 INT 5523")

	rig.waitFor("uptime point", 2*time.Second, func(n client.SerialDev) bool {
		return n.Uptime == 5523
	})

	// any line is evidence the MCU is alive
	rig.waitFor("connected", 2*time.Second, func(n client.SerialDev) bool {
		return n.Connected
	})
}

// A disable/enable cycle has to leave the node reporting connected again once
// the MCU speaks. The state is published only when it changes, so a cycle that
// loses the transition leaves the node reported as not connected for as long
// as the link stays up.
func TestSerialShellDisableEnableReconnects(t *testing.T) {
	rig, teardown := setupShellRig(t, nil)
	defer teardown()

	for range 4 {
		rig.readLine(2 * time.Second)
	}

	rig.send("pt uptime 0 INT 1")
	rig.waitFor("connected", 2*time.Second, func(n client.SerialDev) bool {
		return n.Connected
	})

	rig.setPoint(data.PointTypeDisabled, 1)
	rig.waitFor("disconnected after disable", 2*time.Second, func(n client.SerialDev) bool {
		return !n.Connected
	})

	rig.setPoint(data.PointTypeDisabled, 0)

	// the port reopens, so the handshake goes out a second time
	for range 4 {
		rig.readLine(2 * time.Second)
	}

	rig.send("pt uptime 0 INT 2")
	rig.waitFor("connected after enable", 5*time.Second, func(n client.SerialDev) bool {
		return n.Connected
	})
}

// Same cycle, but with the MCU streaming the whole time and a short silence
// timeout, which is how a board in the field behaves.
func TestSerialShellDisableEnableWhileStreaming(t *testing.T) {
	rig, teardown := setupShellRig(t, func(c *client.SerialDev) {
		c.Timeout = 1
	})
	defer teardown()

	streaming := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; ; i++ {
			select {
			case <-streaming:
				return
			case <-time.After(100 * time.Millisecond):
			}
			_, _ = rig.fifo.Write(fmt.Appendf(nil,
				"[00:00:%02d.000,000] <inf> app: tick %v\r\n", i%60, i))
		}
	}()
	defer func() {
		close(streaming)
		<-done
	}()

	rig.waitFor("connected", 3*time.Second, func(n client.SerialDev) bool {
		return n.Connected
	})

	rig.setPoint(data.PointTypeDisabled, 1)
	rig.waitFor("disconnected after disable", 3*time.Second, func(n client.SerialDev) bool {
		return !n.Connected
	})

	rig.setPoint(data.PointTypeDisabled, 0)
	rig.waitFor("connected after enable", 20*time.Second, func(n client.SerialDev) bool {
		return n.Connected
	})
}

func TestSerialShellConsoleNoiseIsNotAnError(t *testing.T) {
	rig, teardown := setupShellRig(t, nil)
	defer teardown()

	for range 4 {
		rig.readLine(2 * time.Second)
	}

	// a realistic slice of a Nucleo-H743ZI boot
	for _, l := range []string{
		"*** Booting Zephyr OS build v4.0.0-rc1 ***",
		"[00:00:00.310,000] <inf> siot: Network connected",
		"[00:00:00.311,000] <inf> net_config: IPv4 address: 192.168.1.50",
		"uart:~$ ",
		"Interface eth0 (0x24000f00)",
		"pt uptime 0 INT 42",
	} {
		rig.send(l)
	}

	rig.waitFor("point through the noise", 2*time.Second, func(n client.SerialDev) bool {
		return n.Uptime == 42
	})

	if n := rig.getNode(); n.ErrorCount != 0 {
		t.Errorf("got errorCount %v, expected 0 -- console noise must not count as errors", n.ErrorCount)
	}
}

func TestSerialShellMalformedPointCountsError(t *testing.T) {
	rig, teardown := setupShellRig(t, nil)
	defer teardown()

	for range 4 {
		rig.readLine(2 * time.Second)
	}

	rig.send("pt uptime 0 XXX 42")

	rig.waitFor("error count", 2*time.Second, func(n client.SerialDev) bool {
		return n.ErrorCount > 0
	})
}

// TestSerialShellEchoLoop is the regression test for the echo loop. The MCU's
// `p` handler publishes to the same channel its emitter subscribes to, so
// every point SIOT writes comes straight back. Without suppression the two
// sides trade the same point forever.
func TestSerialShellEchoLoop(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}
	defer stop()

	fifo, err := test.NewFifoA("serialfifo")
	if err != nil {
		t.Fatal("Error starting fifo: ", err)
	}
	fifoW := client.NewLineWrapper(fifo, 500)
	defer fifoW.Close()

	serialTest := client.SerialDev{
		ID:          "ID-serial-echo",
		Parent:      root.ID,
		Description: "test echo",
		Port:        "serialfifo",
		Protocol:    data.PointValueProtocolShell,
		Debug:       4,
	}

	if err := client.SendNodeType(nc, serialTest, "test"); err != nil {
		t.Fatal("Error sending node: ", err)
	}

	getNode, stopWatcher, err := client.NodeWatcher[client.SerialDev](nc, serialTest.ID, serialTest.Parent)
	if err != nil {
		t.Fatal("Error setting up node watcher")
	}
	defer stopWatcher()

	lines := make(chan string, 64)
	go func() {
		for {
			buf := make([]byte, 512)
			c, err := fifoW.Read(buf)
			if err != nil {
				return
			}
			lines <- string(buf[:c])
		}
	}()

	start := time.Now()
	for getNode().ID != serialTest.ID {
		if time.Since(start) > time.Second {
			t.Fatal("timeout waiting for node")
		}
		<-time.After(10 * time.Millisecond)
	}

	readLine := func(d time.Duration) string {
		t.Helper()
		select {
		case l := <-lines:
			return l
		case <-time.After(d):
			t.Fatal("timeout waiting for a line")
			return ""
		}
	}

	// drain handshake
	for range 4 {
		readLine(2 * time.Second)
	}

	// Write an uptime point to the node; the client must forward it to the
	// MCU. (description, port, and baud are deliberately not forwarded, so
	// they cannot be used here.)
	uptimePt := data.NewPointInt(data.PointTypeUptime, "", 1234)
	// a point with no origin is treated as coming from the node itself and
	// is not delivered back to its own client
	uptimePt.Origin = root.ID
	err = client.SendNodePoint(nc, serialTest.ID, uptimePt, true)
	if err != nil {
		t.Fatal("Error sending point: ", err)
	}

	var written string
	deadline := time.After(3 * time.Second)
	for written == "" {
		select {
		case l := <-lines:
			if strings.HasPrefix(l, "p uptime ") {
				written = l
			}
		case <-deadline:
			t.Fatal("timeout waiting for the uptime write")
		}
	}

	if !strings.Contains(written, "1234") {
		t.Fatalf("got %q, expected the uptime value", written)
	}
	// SIOT stamps what it writes, which is what makes the echo identifiable
	fields := strings.Fields(written)
	if len(fields) < 6 {
		t.Fatalf("got %q, expected a trailing timestamp field", written)
	}

	// echo it back exactly as the MCU would
	if _, err := fifoW.Write([]byte(strings.Replace(written, "p ", "pt ", 1) + "\r\n")); err != nil {
		t.Fatal("error echoing: ", err)
	}

	// the client must not write it again
	select {
	case l := <-lines:
		if strings.HasPrefix(l, "p uptime ") {
			t.Fatalf("echo was not suppressed, client wrote %q again", l)
		}
	case <-time.After(time.Second):
		// quiet, which is what we want
	}

	// The same point with a DIFFERENT timestamp is a real MCU-side report and
	// must be accepted rather than swallowed. This is the case a value-only
	// match within a time window gets wrong.
	newStamp := time.Now().UTC().Add(time.Second).Format("2006-01-02T15:04:05.000000000Z07:00")
	line := fmt.Sprintf(`pt uptime 0 INT 9999 %v`, newStamp)
	if _, err := fifoW.Write([]byte(line + "\r\n")); err != nil {
		t.Fatal("error sending: ", err)
	}

	start = time.Now()
	for getNode().Uptime != 9999 {
		if time.Since(start) > 3*time.Second {
			t.Fatalf("timeout: a genuine MCU change was not accepted (uptime=%v)",
				getNode().Uptime)
		}
		<-time.After(20 * time.Millisecond)
	}
}
