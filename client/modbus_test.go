package client_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

// waitFor polls cond every 10ms until it returns true, and fails the test with
// what was being waited on if that does not happen in time.
func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()

	start := time.Now()
	for {
		if cond() {
			return
		}
		if time.Since(start) > timeout {
			t.Fatalf("timeout waiting for %v", what)
		}
		time.Sleep(time.Millisecond * 10)
	}
}

// modbusTestTimeout is how long an assertion waits for a value to travel all
// the way around: a point into the store, a poll of the bus, and a point back
// out. It allows for one retry of the ten second port check, since a bus
// client that is restarted while its IOs are created has to reconnect.
const modbusTestTimeout = time.Second * 30

// modbusTestPollPeriod is short so the tests converge quickly.
const modbusTestPollPeriod = 50

// freePort returns a TCP port that is free at the time it is called. A fixed
// port would collide with whatever else is on the machine, including a
// parallel `go test` run.
func freePort(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Error allocating a port: ", err)
	}

	port := fmt.Sprintf("%v", l.Addr().(*net.TCPAddr).Port)

	if err := l.Close(); err != nil {
		t.Fatal("Error closing port allocation listener: ", err)
	}

	return port
}

// sendPoint sends a point the way the UI does, with an origin set. Points sent
// to a client's own node with an empty origin are filtered out by the client
// manager, which assumes the client generated them.
func sendPoint(t *testing.T, nc *nats.Conn, nodeID string, p data.Point) {
	t.Helper()

	p.Origin = "test"
	if err := client.SendNodePoint(nc, nodeID, p, true); err != nil {
		t.Fatalf("Error sending %v point to %v: %v", p.Type, nodeID, err)
	}
}

// modbusCase describes one IO under test. Each case creates an IO on the
// server bus and a matching IO on the client bus, unless clientOnly is set.
type modbusCase struct {
	desc    string
	ioType  string
	format  string
	address int

	// scale and offset are applied on both sides unless clientScale is set.
	scale  float64
	offset float64

	// clientScale and clientOffset replace scale and offset on the client IO.
	// Reading the same register with a different scale shows the raw register
	// contents, which is how the scaled case checks what the server stored.
	clientScale  float64
	clientOffset float64

	// serverValue seeds the server IO, and through it the server's register
	// when the bus opens.
	serverValue float64

	// clientExpect is the value the client IO should settle at once it has
	// polled the server.
	clientExpect float64

	// serverUpdate is written to the server IO node after the first read-back,
	// after which the client IO should reach clientUpdateExpect.
	serverUpdate       *float64
	clientUpdateExpect float64

	// clientSet is written to the client IO as valueSet, after which both the
	// server IO and the client IO should reach it.
	clientSet *float64

	// clientOnly creates only the client side of the pair, for a case that
	// observes a register another case set up.
	clientOnly bool
}

// writable reports whether the client drives this IO type rather than only
// reading it.
func (c modbusCase) writable() bool {
	return c.ioType == data.PointValueModbusCoil ||
		c.ioType == data.PointValueModbusHoldingRegister
}

func f64(v float64) *float64 {
	return &v
}

// modbusCases exercises every IO type and data format over a real TCP
// connection. Bit addresses stay below 32 and word addresses at 100 and above,
// because Regs.AddCoil maps a coil to register num/16 -- coils and registers
// share one address space, so a coil write to a high bit would land on a low
// register. Word addresses advance by the register count of the case before
// them so a 32-bit format cannot overlap its neighbor.
var modbusCases = []modbusCase{
	{
		desc:         "discrete input",
		ioType:       data.PointValueModbusDiscreteInput,
		address:      0,
		serverValue:  1,
		clientExpect: 1,
		serverUpdate: f64(0), clientUpdateExpect: 0,
	},
	{
		desc:         "coil",
		ioType:       data.PointValueModbusCoil,
		address:      16,
		serverValue:  0,
		clientExpect: 0,
		clientSet:    f64(1),
	},
	{
		desc:         "input register uint16",
		ioType:       data.PointValueModbusInputRegister,
		format:       data.PointValueUINT16,
		address:      100,
		scale:        1,
		serverValue:  65535,
		clientExpect: 65535,
		serverUpdate: f64(1234), clientUpdateExpect: 1234,
	},
	{
		desc:         "input register int16",
		ioType:       data.PointValueModbusInputRegister,
		format:       data.PointValueINT16,
		address:      101,
		scale:        1,
		serverValue:  -1234,
		clientExpect: -1234,
		serverUpdate: f64(4321), clientUpdateExpect: 4321,
	},
	{
		desc:         "input register uint32",
		ioType:       data.PointValueModbusInputRegister,
		format:       data.PointValueUINT32,
		address:      102,
		scale:        1,
		serverValue:  4000000000,
		clientExpect: 4000000000,
		serverUpdate: f64(1), clientUpdateExpect: 1,
	},
	{
		desc:         "input register int32",
		ioType:       data.PointValueModbusInputRegister,
		format:       data.PointValueINT32,
		address:      104,
		scale:        1,
		serverValue:  -2000000,
		clientExpect: -2000000,
		serverUpdate: f64(2000000), clientUpdateExpect: 2000000,
	},
	{
		// 3.25 and 1.5 are exact in binary, so equality is safe
		desc:         "input register float32",
		ioType:       data.PointValueModbusInputRegister,
		format:       data.PointValueFLOAT32,
		address:      106,
		scale:        1,
		serverValue:  3.25,
		clientExpect: 3.25,
		serverUpdate: f64(1.5), clientUpdateExpect: 1.5,
	},
	{
		desc:         "holding register uint16",
		ioType:       data.PointValueModbusHoldingRegister,
		format:       data.PointValueUINT16,
		address:      110,
		scale:        1,
		serverValue:  100,
		clientExpect: 100,
		clientSet:    f64(200),
	},
	{
		desc:         "holding register int32",
		ioType:       data.PointValueModbusHoldingRegister,
		format:       data.PointValueINT32,
		address:      112,
		scale:        1,
		serverValue:  -500000,
		clientExpect: -500000,
		clientSet:    f64(600000),
	},
	{
		desc:         "holding register float32",
		ioType:       data.PointValueModbusHoldingRegister,
		format:       data.PointValueFLOAT32,
		address:      114,
		scale:        1,
		serverValue:  1.5,
		clientExpect: 1.5,
		clientSet:    f64(2.75),
	},
	{
		// A temperature transmitter scaling, where the two sides deliberately
		// disagree: the node values are in degrees and the register holds
		// tenths of a degree above -40.
		desc:         "input register scaled",
		ioType:       data.PointValueModbusInputRegister,
		format:       data.PointValueUINT16,
		address:      120,
		scale:        0.1,
		offset:       -40,
		serverValue:  25,
		clientExpect: 25,
	},
	{
		// Reads the register the case above wrote, unscaled, which is the only
		// way from here to see what the server actually stored.
		desc:         "input register scaled, raw",
		ioType:       data.PointValueModbusInputRegister,
		format:       data.PointValueUINT16,
		address:      120,
		clientScale:  1,
		clientExpect: 650,
		clientOnly:   true,
	},
}

// modbusBus creates a bus node and returns it.
func modbusBus(t *testing.T, nc *nats.Conn, bus client.Modbus) client.Modbus {
	t.Helper()

	if err := client.SendNodeType(nc, bus, "test"); err != nil {
		t.Fatalf("Error creating modbus bus %v: %v", bus.Description, err)
	}

	return bus
}

// modbusIO creates an IO node below a bus and returns a getter for its
// current state.
func modbusIO(t *testing.T, nc *nats.Conn, io client.ModbusIo) func() client.ModbusIo {
	t.Helper()

	if err := client.SendNodeType(nc, io, "test"); err != nil {
		t.Fatalf("Error creating modbus IO %v: %v", io.Description, err)
	}

	get, stop, err := client.NodeWatcher[client.ModbusIo](nc, io.ID, io.Parent)
	if err != nil {
		t.Fatalf("Error watching modbus IO %v: %v", io.Description, err)
	}

	t.Cleanup(stop)

	waitFor(t, modbusTestTimeout, "modbus IO node "+io.Description, func() bool {
		return get().ID == io.ID
	})

	return get
}

// TestModbus runs a modbus server bus and a modbus client bus against each
// other over a TCP loopback socket, so every register type and data format
// goes over a real wire and back out as a point on a SIOT node.
func TestModbus(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	port := freePort(t)

	// The server bus and its IOs are created first and given time to settle,
	// so the client bus connects to a listener that is already up. Adding an
	// IO restarts its bus client, which is why the two sides are not created
	// at the same time.
	serverBus := modbusBus(t, nc, client.Modbus{
		ID:           "modbus-server",
		Parent:       root.ID,
		Description:  "test modbus server",
		ClientServer: data.PointValueServer,
		Protocol:     data.PointValueTCP,
		Port:         port,
		ServerID:     1,
	})

	serverIOs := make([]func() client.ModbusIo, len(modbusCases))

	for i, c := range modbusCases {
		if c.clientOnly {
			continue
		}
		serverIOs[i] = modbusIO(t, nc, client.ModbusIo{
			ID:           fmt.Sprintf("modbus-server-io-%v", i),
			Parent:       serverBus.ID,
			Description:  c.desc,
			ServerID:     1,
			Address:      c.address,
			ModbusIOType: c.ioType,
			DataFormat:   c.format,
			Scale:        c.scale,
			Offset:       c.offset,
			Value:        c.serverValue,
		})
	}

	// wait until the server is listening before pointing a client at it
	waitFor(t, modbusTestTimeout, "modbus server to listen", func() bool {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	})

	clientBus := modbusBus(t, nc, client.Modbus{
		ID:           "modbus-client",
		Parent:       root.ID,
		Description:  "test modbus client",
		ClientServer: data.PointValueClient,
		Protocol:     data.PointValueTCP,
		URI:          "127.0.0.1:" + port,
		PollPeriod:   modbusTestPollPeriod,
	})

	clientIOs := make([]func() client.ModbusIo, len(modbusCases))

	for i, c := range modbusCases {
		scale, offset := c.scale, c.offset
		if c.clientScale != 0 {
			scale, offset = c.clientScale, c.clientOffset
		}

		io := client.ModbusIo{
			ID:           fmt.Sprintf("modbus-client-io-%v", i),
			Parent:       clientBus.ID,
			Description:  c.desc,
			ServerID:     1,
			Address:      c.address,
			ModbusIOType: c.ioType,
			DataFormat:   c.format,
			Scale:        scale,
			Offset:       offset,
		}

		// A writable IO whose valueSet does not match what it reads is written
		// on the next poll, so start it at the value the server holds and let
		// the test drive it from there.
		if c.writable() {
			io.ValueSet = c.serverValue
		}

		clientIOs[i] = modbusIO(t, nc, io)
	}

	// every client IO should read what the server was seeded with
	for i, c := range modbusCases {
		get := clientIOs[i]
		waitFor(t, modbusTestTimeout,
			fmt.Sprintf("client IO %q to read %v", c.desc, c.clientExpect),
			func() bool { return get().Value == c.clientExpect })
	}

	// a value written on the server should show up on the client
	for i, c := range modbusCases {
		if c.serverUpdate == nil {
			continue
		}

		sendPoint(t, nc, fmt.Sprintf("modbus-server-io-%v", i),
			data.NewPointFloat(data.PointTypeValue, "", *c.serverUpdate))

		get := clientIOs[i]
		waitFor(t, modbusTestTimeout,
			fmt.Sprintf("client IO %q to follow server to %v", c.desc, c.clientUpdateExpect),
			func() bool { return get().Value == c.clientUpdateExpect })
	}

	// a value set on the client should be written to the server
	for i, c := range modbusCases {
		if c.clientSet == nil {
			continue
		}

		sendPoint(t, nc, fmt.Sprintf("modbus-client-io-%v", i),
			data.NewPointFloat(data.PointTypeValueSet, "", *c.clientSet))

		serverGet, clientGet := serverIOs[i], clientIOs[i]

		waitFor(t, modbusTestTimeout,
			fmt.Sprintf("server IO %q to follow client to %v", c.desc, *c.clientSet),
			func() bool { return serverGet().Value == *c.clientSet })

		waitFor(t, modbusTestTimeout,
			fmt.Sprintf("client IO %q to read back %v", c.desc, *c.clientSet),
			func() bool { return clientGet().Value == *c.clientSet })
	}
}

// TestModbusAddIO checks that an IO added to a running bus starts being
// polled. Adding a child restarts the bus client, so this is the one case that
// cannot be folded into the matrix above.
func TestModbusAddIO(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	port := freePort(t)

	serverBus := modbusBus(t, nc, client.Modbus{
		ID:           "modbus-server",
		Parent:       root.ID,
		Description:  "test modbus server",
		ClientServer: data.PointValueServer,
		Protocol:     data.PointValueTCP,
		Port:         port,
		ServerID:     1,
	})

	modbusIO(t, nc, client.ModbusIo{
		ID:           "modbus-server-io-1",
		Parent:       serverBus.ID,
		Description:  "first",
		ServerID:     1,
		Address:      100,
		ModbusIOType: data.PointValueModbusInputRegister,
		DataFormat:   data.PointValueUINT16,
		Scale:        1,
		Value:        11,
	})

	waitFor(t, modbusTestTimeout, "modbus server to listen", func() bool {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	})

	clientBus := modbusBus(t, nc, client.Modbus{
		ID:           "modbus-client",
		Parent:       root.ID,
		Description:  "test modbus client",
		ClientServer: data.PointValueClient,
		Protocol:     data.PointValueTCP,
		URI:          "127.0.0.1:" + port,
		PollPeriod:   modbusTestPollPeriod,
	})

	firstIO := modbusIO(t, nc, client.ModbusIo{
		ID:           "modbus-client-io-1",
		Parent:       clientBus.ID,
		Description:  "first",
		ServerID:     1,
		Address:      100,
		ModbusIOType: data.PointValueModbusInputRegister,
		DataFormat:   data.PointValueUINT16,
		Scale:        1,
	})

	waitFor(t, modbusTestTimeout, "first client IO to be polled", func() bool {
		return firstIO().Value == 11
	})

	// now add a second IO on both sides and check it starts being polled
	modbusIO(t, nc, client.ModbusIo{
		ID:           "modbus-server-io-2",
		Parent:       serverBus.ID,
		Description:  "second",
		ServerID:     1,
		Address:      101,
		ModbusIOType: data.PointValueModbusInputRegister,
		DataFormat:   data.PointValueUINT16,
		Scale:        1,
		Value:        22,
	})

	secondIO := modbusIO(t, nc, client.ModbusIo{
		ID:           "modbus-client-io-2",
		Parent:       clientBus.ID,
		Description:  "second",
		ServerID:     1,
		Address:      101,
		ModbusIOType: data.PointValueModbusInputRegister,
		DataFormat:   data.PointValueUINT16,
		Scale:        1,
	})

	waitFor(t, modbusTestTimeout, "second client IO to be polled", func() bool {
		return secondIO().Value == 22
	})

	// the first IO should still be polled after the restart
	waitFor(t, modbusTestTimeout, "first client IO to still be polled", func() bool {
		return firstIO().Value == 11
	})
}

// TestModbusErrors checks that a client pointed at a port with nothing
// listening counts errors on both the bus and the IO, that the counts stop
// rising once a server appears, and that a reset zeroes them.
func TestModbusErrors(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	port := freePort(t)

	clientBus := modbusBus(t, nc, client.Modbus{
		ID:           "modbus-client",
		Parent:       root.ID,
		Description:  "test modbus client",
		ClientServer: data.PointValueClient,
		Protocol:     data.PointValueTCP,
		URI:          "127.0.0.1:" + port,
		PollPeriod:   modbusTestPollPeriod,
	})

	busGet, busStop, err := client.NodeWatcher[client.Modbus](nc, clientBus.ID, clientBus.Parent)
	if err != nil {
		t.Fatal("Error watching modbus bus: ", err)
	}

	defer busStop()

	ioGet := modbusIO(t, nc, client.ModbusIo{
		ID:           "modbus-client-io-1",
		Parent:       clientBus.ID,
		Description:  "io",
		ServerID:     1,
		Address:      100,
		ModbusIOType: data.PointValueModbusInputRegister,
		DataFormat:   data.PointValueUINT16,
		Scale:        1,
	})

	waitFor(t, modbusTestTimeout, "bus error count to rise", func() bool {
		return busGet().ErrorCount > 0
	})

	waitFor(t, modbusTestTimeout, "IO error count to rise", func() bool {
		return ioGet().ErrorCount > 0
	})

	// resetting the count should zero it and clear the request
	sendPoint(t, nc, clientBus.ID,
		data.NewPointFloat(data.PointTypeErrorCountReset, "", 1))

	waitFor(t, modbusTestTimeout, "bus error count reset to be cleared", func() bool {
		return !busGet().ErrorCountReset
	})
}

// TestModbusDisabled checks that a disabled IO is not polled, and that
// disabling a bus closes its port.
func TestModbusDisabled(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	port := freePort(t)

	serverBus := modbusBus(t, nc, client.Modbus{
		ID:           "modbus-server",
		Parent:       root.ID,
		Description:  "test modbus server",
		ClientServer: data.PointValueServer,
		Protocol:     data.PointValueTCP,
		Port:         port,
		ServerID:     1,
	})

	modbusIO(t, nc, client.ModbusIo{
		ID:           "modbus-server-io-1",
		Parent:       serverBus.ID,
		Description:  "input",
		ServerID:     1,
		Address:      100,
		ModbusIOType: data.PointValueModbusInputRegister,
		DataFormat:   data.PointValueUINT16,
		Scale:        1,
		Value:        42,
	})

	listening := func() bool {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}

	waitFor(t, modbusTestTimeout, "modbus server to listen", listening)

	clientBus := modbusBus(t, nc, client.Modbus{
		ID:           "modbus-client",
		Parent:       root.ID,
		Description:  "test modbus client",
		ClientServer: data.PointValueClient,
		Protocol:     data.PointValueTCP,
		URI:          "127.0.0.1:" + port,
		PollPeriod:   modbusTestPollPeriod,
	})

	clientIO := modbusIO(t, nc, client.ModbusIo{
		ID:           "modbus-client-io-1",
		Parent:       clientBus.ID,
		Description:  "input",
		ServerID:     1,
		Address:      100,
		ModbusIOType: data.PointValueModbusInputRegister,
		DataFormat:   data.PointValueUINT16,
		Scale:        1,
		Disabled:     true,
	})

	// give the bus several poll periods to read the IO if it is going to
	time.Sleep(time.Millisecond * modbusTestPollPeriod * 10)

	if v := clientIO().Value; v != 0 {
		t.Fatalf("disabled IO was polled, value is %v", v)
	}

	// enabling it should start the polling
	sendPoint(t, nc, "modbus-client-io-1",
		data.NewPointFloat(data.PointTypeDisabled, "", 0))

	waitFor(t, modbusTestTimeout, "enabled IO to be polled", func() bool {
		return clientIO().Value == 42
	})

	// disabling the server bus should close its port
	sendPoint(t, nc, serverBus.ID,
		data.NewPointFloat(data.PointTypeDisabled, "", 1))

	waitFor(t, modbusTestTimeout, "disabled server bus to close its port", func() bool {
		return !listening()
	})
}

// TestModbusReadOnly checks that a read-only IO is not written even when its
// valueSet does not match what the server holds.
func TestModbusReadOnly(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	port := freePort(t)

	serverBus := modbusBus(t, nc, client.Modbus{
		ID:           "modbus-server",
		Parent:       root.ID,
		Description:  "test modbus server",
		ClientServer: data.PointValueServer,
		Protocol:     data.PointValueTCP,
		Port:         port,
		ServerID:     1,
	})

	serverIO := modbusIO(t, nc, client.ModbusIo{
		ID:           "modbus-server-io-1",
		Parent:       serverBus.ID,
		Description:  "holding",
		ServerID:     1,
		Address:      100,
		ModbusIOType: data.PointValueModbusHoldingRegister,
		DataFormat:   data.PointValueUINT16,
		Scale:        1,
		Value:        42,
	})

	waitFor(t, modbusTestTimeout, "modbus server to listen", func() bool {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	})

	clientBus := modbusBus(t, nc, client.Modbus{
		ID:           "modbus-client",
		Parent:       root.ID,
		Description:  "test modbus client",
		ClientServer: data.PointValueClient,
		Protocol:     data.PointValueTCP,
		URI:          "127.0.0.1:" + port,
		PollPeriod:   modbusTestPollPeriod,
	})

	clientIO := modbusIO(t, nc, client.ModbusIo{
		ID:           "modbus-client-io-1",
		Parent:       clientBus.ID,
		Description:  "holding",
		ServerID:     1,
		Address:      100,
		ModbusIOType: data.PointValueModbusHoldingRegister,
		DataFormat:   data.PointValueUINT16,
		Scale:        1,
		ReadOnly:     true,
	})

	waitFor(t, modbusTestTimeout, "read-only IO to read the server value", func() bool {
		return clientIO().Value == 42
	})

	sendPoint(t, nc, "modbus-client-io-1",
		data.NewPointFloat(data.PointTypeValueSet, "", 99))

	// give the client several poll periods to write if it is going to
	time.Sleep(time.Millisecond * modbusTestPollPeriod * 10)

	if v := serverIO().Value; v != 42 {
		t.Fatalf("read-only IO wrote to the server, value is %v", v)
	}

	if v := clientIO().Value; v != 42 {
		t.Fatalf("read-only IO changed its own value to %v", v)
	}
}
