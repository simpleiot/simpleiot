package client

import (
	"errors"
	"fmt"
	goio "io"
	"log"
	"net"
	"strconv"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/modbus"
	"github.com/simpleiot/simpleiot/respreader"
	"go.bug.st/serial"
)

// defaultModbusTimeout is the response timeout in milliseconds used when a bus
// does not have a valid one configured.
const defaultModbusTimeout = 100

// modbusValueRefresh is how often a value point is published even when the
// value has not changed, so that a reading is never older than this.
const modbusValueRefresh = time.Minute * 10

// Modbus describes a modbus bus. The Go type name maps to the "modbus" node
// type, so the client manager starts a ModbusClient for every modbus node.
type Modbus struct {
	ID           string `node:"id"`
	Parent       string `node:"parent"`
	Description  string `point:"description"`
	ClientServer string `point:"clientServer"`
	Protocol     string `point:"protocol"`
	URI          string `point:"uri"`
	// Port is the serial port for RTU, and the TCP port to listen on when
	// serving over TCP.
	Port string `point:"port"`
	Baud string `point:"baud"`
	// ServerID is the modbus server (unit) address used when this bus is a
	// server. It is unrelated to the SIOT node ID above.
	ServerID           int        `point:"id"`
	PollPeriod         int        `point:"pollPeriod"`
	Timeout            int        `point:"timeout"`
	Debug              int        `point:"debug"`
	Disabled           bool       `point:"disabled"`
	ErrorCount         int        `point:"errorCount"`
	ErrorCountCRC      int        `point:"errorCountCRC"`
	ErrorCountEOF      int        `point:"errorCountEOF"`
	ErrorCountReset    bool       `point:"errorCountReset"`
	ErrorCountCRCReset bool       `point:"errorCountCRCReset"`
	ErrorCountEOFReset bool       `point:"errorCountEOFReset"`
	IOs                []ModbusIo `child:"modbusIo"`
}

// modbusServer is the subset of a modbus server used by this client. Both
// modbus.Server (RTU) and modbus.TCPServer satisfy it.
type modbusServer interface {
	Close() error
	Listen(func(error), func(), func())
}

// ModbusClient is a SIOT client that runs a modbus bus, either as a client
// polling remote devices or as a server responding to them. The IOs configured
// below the bus node are managed by this client rather than by clients of
// their own, because they share the port, the register map, and the poll cycle.
type ModbusClient struct {
	nc            *nats.Conn
	config        Modbus
	stop          chan struct{}
	newPoints     chan NewPoints
	newEdgePoints chan NewPoints
	chRegChange   chan bool

	// state that only exists while the client is running
	regs         *modbus.Regs
	client       *modbus.Client
	server       modbusServer
	serialPort   serial.Port
	ioErrorCount int
	// portClosed is closed when the port is closed, which releases the server
	// listener goroutine if it is waiting to report a register change.
	portClosed chan struct{}
	// lastSent records when a value point was last published for each IO node
	// so that an unchanging value is still refreshed periodically.
	lastSent map[string]time.Time
	// configErr is the last configuration problem reported. An incomplete
	// configuration is logged when it changes rather than on every retry.
	configErr string
}

// NewModbusClient returns a new modbus client for the given bus node
func NewModbusClient(nc *nats.Conn, config Modbus) Client {
	return &ModbusClient{
		nc:            nc,
		config:        config,
		stop:          make(chan struct{}),
		newPoints:     make(chan NewPoints),
		newEdgePoints: make(chan NewPoints),
		chRegChange:   make(chan bool),
		lastSent:      make(map[string]time.Time),
	}
}

// Run runs the main logic for this client and blocks until stopped
func (c *ModbusClient) Run() error {
	log.Println("Starting modbus client:", c.config.Description)

	scanTimer := time.NewTicker(time.Hour)
	scanTimer.Stop()

	setScanTimer := func() {
		if c.config.ClientServer == data.PointValueClient &&
			c.config.PollPeriod > 0 && !c.config.Disabled {
			scanTimer.Reset(time.Millisecond * time.Duration(c.config.PollPeriod))
		} else {
			scanTimer.Stop()
		}
	}

	// The port check timer detects a serial port that was unplugged and
	// plugged back in, and retries a port that could not be opened.
	checkPortTimer := time.NewTicker(time.Second * 10)

	c.checkTimeout()
	c.setupPort()
	setScanTimer()

done:
	for {
		select {
		case <-c.stop:
			break done

		case pts := <-c.newPoints:
			err := data.MergePoints(pts.ID, pts.Points, &c.config)
			if err != nil {
				log.Println("Modbus: error merging new points:", err)
			}

			if pts.ID == c.config.ID {
				c.busPoints(pts.Points, setScanTimer)
			} else {
				c.ioPoints(pts.ID, pts.Points)
			}

		case pts := <-c.newEdgePoints:
			err := data.MergeEdgePoints(pts.ID, pts.Parent, pts.Points, &c.config)
			if err != nil {
				log.Println("Modbus: error merging new edge points:", err)
			}

		case <-c.chRegChange:
			// this only happens on modbus servers
			for i := range c.config.IOs {
				io := &c.config.IOs[i]
				if err := c.ServerIO(io); err != nil {
					c.LogError(io, err)
				}
			}

		case <-checkPortTimer.C:
			var portError error
			if c.serialPort != nil {
				// the following handles cases where the serial port may have
				// been unplugged and plugged back in
				_, portError = c.serialPort.GetModemStatusBits()
			}

			if c.config.Disabled {
				c.ClosePort()
				break
			}

			if (c.client == nil && c.server == nil) ||
				c.ioErrorCount > 10 || portError != nil {
				if c.config.Debug >= 1 {
					log.Printf("Re-initializing modbus port, err cnt: %v, portError: %v\n",
						c.ioErrorCount, portError)
				}
				c.ioErrorCount = 0
				c.setupPort()
			}

		case <-scanTimer.C:
			if c.config.ClientServer != data.PointValueClient || c.config.Disabled {
				break
			}

			for i := range c.config.IOs {
				io := &c.config.IOs[i]
				if io.Disabled {
					continue
				}
				if err := c.ClientIO(io); err != nil {
					c.LogError(io, err)
				}
			}
		}
	}

	log.Println("Stopping modbus client:", c.config.Description)
	scanTimer.Stop()
	checkPortTimer.Stop()
	c.ClosePort()

	return nil
}

// Stop sends a signal to the Run function to exit
func (c *ModbusClient) Stop(_ error) {
	close(c.stop)
}

// Points is called by the Manager when new points for this node are received.
func (c *ModbusClient) Points(nodeID string, points []data.Point) {
	c.newPoints <- NewPoints{nodeID, "", points}
}

// EdgePoints is called by the Manager when new edge points for this node are
// received.
func (c *ModbusClient) EdgePoints(nodeID, parentID string, points []data.Point) {
	c.newEdgePoints <- NewPoints{nodeID, parentID, points}
}

// busPoints reacts to points on the bus node that have a side effect beyond
// updating the config, which has already been done by the caller.
func (c *ModbusClient) busPoints(pts data.Points, setScanTimer func()) {
	reopen := false

	for _, p := range pts {
		switch p.Type {
		case data.PointTypeClientServer,
			data.PointTypeID,
			data.PointTypeDebug,
			data.PointTypePort,
			data.PointTypeBaud,
			data.PointTypeURI,
			data.PointTypeProtocol,
			data.PointTypeDisabled:
			reopen = true

		case data.PointTypeTimeout:
			c.checkTimeout()
			reopen = true

		case data.PointTypePollPeriod:
			setScanTimer()

		case data.PointTypeErrorCountReset:
			if c.config.ErrorCountReset {
				c.config.ErrorCount = 0
				c.config.ErrorCountReset = false
				c.sendResetPoints(c.config.ID, data.PointTypeErrorCount,
					data.PointTypeErrorCountReset)
			}

		case data.PointTypeErrorCountCRCReset:
			if c.config.ErrorCountCRCReset {
				c.config.ErrorCountCRC = 0
				c.config.ErrorCountCRCReset = false
				c.sendResetPoints(c.config.ID, data.PointTypeErrorCountCRC,
					data.PointTypeErrorCountCRCReset)
			}

		case data.PointTypeErrorCountEOFReset:
			if c.config.ErrorCountEOFReset {
				c.config.ErrorCountEOF = 0
				c.config.ErrorCountEOFReset = false
				c.sendResetPoints(c.config.ID, data.PointTypeErrorCountEOF,
					data.PointTypeErrorCountEOFReset)
			}
		}
	}

	if reopen {
		c.setupPort()
		setScanTimer()
	}
}

// ioPoints reacts to points on one of the IO nodes below the bus.
func (c *ModbusClient) ioPoints(id string, pts data.Points) {
	index := -1
	for i := range c.config.IOs {
		if c.config.IOs[i].ID == id {
			index = i
			break
		}
	}

	if index < 0 {
		// Adding or removing an IO restarts this client, so a point for an
		// unknown node simply means the restart has not happened yet.
		return
	}

	io := &c.config.IOs[index]

	initRegs := false
	valueModified := false
	valueSetModified := false

	for _, p := range pts {
		switch p.Type {
		case data.PointTypeAddress,
			data.PointTypeModbusIOType,
			data.PointTypeDataFormat:
			initRegs = true

		case data.PointTypeValue:
			valueModified = true

		case data.PointTypeValueSet:
			valueSetModified = true

		case data.PointTypeErrorCountReset:
			if io.ErrorCountReset {
				io.ErrorCount = 0
				io.ErrorCountReset = false
				c.sendResetPoints(io.ID, data.PointTypeErrorCount,
					data.PointTypeErrorCountReset)
			}

		case data.PointTypeErrorCountCRCReset:
			if io.ErrorCountCRCReset {
				io.ErrorCountCRC = 0
				io.ErrorCountCRCReset = false
				c.sendResetPoints(io.ID, data.PointTypeErrorCountCRC,
					data.PointTypeErrorCountCRCReset)
			}

		case data.PointTypeErrorCountEOFReset:
			if io.ErrorCountEOFReset {
				io.ErrorCountEOF = 0
				io.ErrorCountEOFReset = false
				c.sendResetPoints(io.ID, data.PointTypeErrorCountEOF,
					data.PointTypeErrorCountEOFReset)
			}
		}
	}

	if initRegs {
		c.InitRegs(io)
	}

	if io.Disabled {
		return
	}

	if valueModified && c.config.ClientServer == data.PointValueServer {
		if err := c.ServerIO(io); err != nil {
			c.LogError(io, err)
		}
	}

	if valueSetModified && c.config.ClientServer == data.PointValueClient &&
		(io.ModbusIOType == data.PointValueModbusCoil ||
			io.ModbusIOType == data.PointValueModbusHoldingRegister) &&
		io.Value != io.ValueSet {
		if err := c.ClientIO(io); err != nil {
			c.LogError(io, err)
		}
	}
}

// checkTimeout replaces a missing or non-positive response timeout with the
// default and publishes the corrected value, so what the bus is using is what
// the configuration shows.
func (c *ModbusClient) checkTimeout() {
	if c.config.Timeout > 0 {
		return
	}

	c.config.Timeout = defaultModbusTimeout

	err := SendNodePoint(c.nc, c.config.ID,
		data.NewPointFloat(data.PointTypeTimeout, "", float64(c.config.Timeout)), true)
	if err != nil {
		log.Println("Modbus: error sending corrected timeout:", err)
	}
}

// validate returns a description of what keeps the bus from being started, or
// an empty string when the configuration is usable. A client cannot refuse to
// start the way the old bus manager could, so an incomplete configuration
// leaves the port closed until a point arrives that completes it.
func (c *ModbusClient) validate() string {
	switch c.config.ClientServer {
	case data.PointValueClient, data.PointValueServer:
	case "":
		return "modbus client/server is not set"
	default:
		return fmt.Sprintf("invalid modbus client/server: %v", c.config.ClientServer)
	}

	switch c.config.Protocol {
	case data.PointValueRTU:
		if c.config.Port == "" {
			return "modbus port is not set"
		}
		if _, err := strconv.Atoi(c.config.Baud); err != nil {
			return fmt.Sprintf("invalid modbus baud: %v", c.config.Baud)
		}
	case data.PointValueTCP:
		if c.config.ClientServer == data.PointValueClient && c.config.URI == "" {
			return "modbus URI is not set"
		}
		if c.config.ClientServer == data.PointValueServer && c.config.Port == "" {
			return "modbus port is not set"
		}
	case "":
		return "modbus protocol is not set"
	default:
		return fmt.Sprintf("unsupported modbus protocol: %v", c.config.Protocol)
	}

	if c.config.ClientServer == data.PointValueClient && c.config.PollPeriod <= 0 {
		return "modbus poll period is not set"
	}

	return ""
}

// setupPort validates the configuration and opens the port, leaving it closed
// when the bus is disabled or not configured yet.
func (c *ModbusClient) setupPort() {
	if c.config.Disabled {
		c.ClosePort()
		return
	}

	if problem := c.validate(); problem != "" {
		if problem != c.configErr {
			log.Printf("Modbus %v: %v\n", c.config.Description, problem)
			c.configErr = problem
		}
		c.ClosePort()
		return
	}

	c.configErr = ""

	if err := c.SetupPort(); err != nil {
		log.Println("Modbus: error setting up port:", err)
	}
}

// SetupPort sets up the transport for the bus
func (c *ModbusClient) SetupPort() error {
	if c.config.Debug >= 1 {
		log.Println("modbus: setting up modbus transport:", c.config.Port)
	}

	c.ClosePort()

	var transport modbus.Transport

	switch c.config.Protocol {
	case data.PointValueRTU:
		baud, err := strconv.Atoi(c.config.Baud)
		if err != nil {
			return fmt.Errorf("invalid baud: %v", c.config.Baud)
		}

		mode := &serial.Mode{
			BaudRate: baud,
		}

		c.serialPort, err = serial.Open(c.config.Port, mode)
		if err != nil {
			c.serialPort = nil
			return fmt.Errorf("error opening serial port: %w", err)
		}

		port := respreader.NewReadWriteCloser(c.serialPort,
			time.Millisecond*time.Duration(c.config.Timeout), time.Millisecond*20)

		transport = modbus.NewRTU(port)

	case data.PointValueTCP:
		switch c.config.ClientServer {
		case data.PointValueClient:
			sock, err := net.DialTimeout("tcp", c.config.URI, 5*time.Second)
			if err != nil {
				return err
			}
			transport = modbus.NewTCP(sock, 500*time.Millisecond, modbus.TransportClient)
		case data.PointValueServer:
			// TCPServer does all the setup
		}

	default:
		return fmt.Errorf("unsupported modbus protocol: %v", c.config.Protocol)
	}

	switch c.config.ClientServer {
	case data.PointValueServer:
		c.regs = &modbus.Regs{}

		switch c.config.Protocol {
		case data.PointValueRTU:
			c.server = modbus.NewServer(byte(c.config.ServerID), transport,
				c.regs, c.config.Debug)
		case data.PointValueTCP:
			var err error
			c.server, err = modbus.NewTCPServer(c.config.ServerID, 5,
				c.config.Port, c.regs, c.config.Debug)
			if err != nil {
				c.server = nil
				return err
			}
		}

		// copied so the listener goroutine does not read fields the main loop
		// may be writing
		debug := c.config.Debug
		closed := make(chan struct{})
		c.portClosed = closed

		go c.server.Listen(func(err error) {
			log.Println("Modbus server error:", err)
		}, func() {
			if debug > 0 {
				log.Println("Modbus reg change")
			}
			select {
			case c.chRegChange <- true:
			case <-closed:
			case <-c.stop:
			}
		}, func() {
			if debug > 0 {
				log.Println("Modbus listener done")
			}
		})

		for i := range c.config.IOs {
			c.InitRegs(&c.config.IOs[i])
		}

	case data.PointValueClient:
		c.client = modbus.NewClient(transport, c.config.Debug)
	}

	return nil
}

// ClosePort closes both the server and client ports
func (c *ModbusClient) ClosePort() {
	if c.portClosed != nil {
		close(c.portClosed)
		c.portClosed = nil
	}

	if c.server != nil {
		if err := c.server.Close(); err != nil {
			log.Println("Modbus: error closing server:", err)
		}
		c.server = nil
	}

	if c.client != nil {
		if err := c.client.Close(); err != nil {
			log.Println("Modbus: error closing client:", err)
		}
		c.client = nil
	}

	c.serialPort = nil
}

// SendPoint sends a point over NATS. Points sent to an IO node carry the bus
// node ID as their origin so the client manager does not feed them back to
// this client as new input.
func (c *ModbusClient) SendPoint(nodeID, pointType string, value float64) error {
	p := data.NewPointFloat(pointType, "", value)
	p.Time = time.Now()
	if nodeID != c.config.ID {
		p.Origin = c.config.ID
	}

	return SendNodePoint(c.nc, nodeID, p, true)
}

// sendResetPoints zeroes an error count and clears the reset request that
// triggered it.
func (c *ModbusClient) sendResetPoints(nodeID, countType, resetType string) {
	pts := data.Points{
		data.NewPointFloat(countType, "", 0),
		data.NewPointFloat(resetType, "", 0),
	}

	if nodeID != c.config.ID {
		for i := range pts {
			pts[i].Origin = c.config.ID
		}
	}

	if err := SendNodePoints(c.nc, nodeID, pts, true); err != nil {
		log.Println("Modbus: error sending reset points:", err)
	}
}

// sendValue publishes a value point for an IO if it changed, or if the last
// one is old enough that it is worth repeating.
func (c *ModbusClient) sendValue(io *ModbusIo, value float64) error {
	if value == io.Value && time.Since(c.lastSent[io.ID]) <= modbusValueRefresh {
		return nil
	}

	io.Value = value

	if err := c.SendPoint(io.ID, data.PointTypeValue, value); err != nil {
		return err
	}

	c.lastSent[io.ID] = time.Now()

	return nil
}

// WriteBusHoldingReg writes a register value to the bus. Client mode only.
func (c *ModbusClient) WriteBusHoldingReg(io *ModbusIo) error {
	unscaledValue := (io.ValueSet - io.Offset) / io.scaleFactor()

	writePair := func(regs []uint16) error {
		if err := c.client.WriteSingleReg(byte(io.ServerID),
			uint16(io.Address), regs[0]); err != nil {
			return err
		}
		return c.client.WriteSingleReg(byte(io.ServerID),
			uint16(io.Address+1), regs[1])
	}

	switch io.DataFormat {
	case data.PointValueUINT16, data.PointValueINT16:
		return c.client.WriteSingleReg(byte(io.ServerID),
			uint16(io.Address), uint16(unscaledValue))
	case data.PointValueUINT32:
		return writePair(modbus.Uint32ToRegs([]uint32{uint32(unscaledValue)}))
	case data.PointValueINT32:
		return writePair(modbus.Int32ToRegs([]int32{int32(unscaledValue)}))
	case data.PointValueFLOAT32:
		return writePair(modbus.Float32ToRegs([]float32{float32(unscaledValue)}))
	default:
		return fmt.Errorf("unhandled data type: %v", io.DataFormat)
	}
}

// ReadBusReg reads an IO value from a register on the bus and publishes it.
// Client mode only.
func (c *ModbusClient) ReadBusReg(io *ModbusIo) error {
	readFunc := c.client.ReadHoldingRegs
	switch io.ModbusIOType {
	case data.PointValueModbusHoldingRegister:
	case data.PointValueModbusInputRegister:
		readFunc = c.client.ReadInputRegs
	default:
		return fmt.Errorf("ReadBusReg: unsupported modbus IO type: %v",
			io.ModbusIOType)
	}

	read := func(count uint16) ([]uint16, error) {
		regs, err := readFunc(byte(io.ServerID), uint16(io.Address), count)
		if err != nil {
			return nil, err
		}
		if len(regs) < int(count) {
			return nil, errors.New("did not receive enough data")
		}
		return regs, nil
	}

	var valueUnscaled float64

	switch io.DataFormat {
	case data.PointValueUINT16, data.PointValueINT16:
		regs, err := read(1)
		if err != nil {
			return err
		}
		if io.DataFormat == data.PointValueINT16 {
			valueUnscaled = float64(int16(regs[0]))
		} else {
			valueUnscaled = float64(regs[0])
		}

	case data.PointValueUINT32:
		regs, err := read(2)
		if err != nil {
			return err
		}
		valueUnscaled = float64(modbus.RegsToUint32(regs)[0])

	case data.PointValueINT32:
		regs, err := read(2)
		if err != nil {
			return err
		}
		valueUnscaled = float64(modbus.RegsToInt32(regs)[0])

	case data.PointValueFLOAT32:
		regs, err := read(2)
		if err != nil {
			return err
		}
		valueUnscaled = float64(modbus.RegsToFloat32(regs)[0])

	default:
		return fmt.Errorf("unhandled data type: %v", io.DataFormat)
	}

	return c.sendValue(io, valueUnscaled*io.scaleFactor()+io.Offset)
}

// ReadBusBit reads a coil or discrete input value from the bus and publishes
// it. Client mode only.
func (c *ModbusClient) ReadBusBit(io *ModbusIo) error {
	readFunc := c.client.ReadCoils
	switch io.ModbusIOType {
	case data.PointValueModbusCoil:
	case data.PointValueModbusDiscreteInput:
		readFunc = c.client.ReadDiscreteInputs
	default:
		return fmt.Errorf("ReadBusBit: unhandled modbusIOType: %v",
			io.ModbusIOType)
	}

	bits, err := readFunc(byte(io.ServerID), uint16(io.Address), 1)
	if err != nil {
		return err
	}
	if len(bits) < 1 {
		return errors.New("did not receive enough data")
	}

	return c.sendValue(io, data.BoolToFloat(bits[0]))
}

// ClientIO processes an IO on a client bus
func (c *ModbusClient) ClientIO(io *ModbusIo) error {
	if c.client == nil {
		return errors.New("client is not set up")
	}

	switch io.ModbusIOType {
	case data.PointValueModbusCoil:
		if err := c.ReadBusBit(io); err != nil {
			return err
		}

		if !io.ReadOnly && io.ValueSet != io.Value {
			err := c.client.WriteSingleCoil(byte(io.ServerID), uint16(io.Address),
				data.FloatToBool(io.ValueSet))
			if err != nil {
				return err
			}

			io.Value = io.ValueSet
			if err := c.SendPoint(io.ID, data.PointTypeValue, io.ValueSet); err != nil {
				return err
			}
			c.lastSent[io.ID] = time.Now()
		}

	case data.PointValueModbusDiscreteInput:
		return c.ReadBusBit(io)

	case data.PointValueModbusHoldingRegister:
		if err := c.ReadBusReg(io); err != nil {
			return err
		}

		if !io.ReadOnly && io.ValueSet != io.Value {
			if err := c.WriteBusHoldingReg(io); err != nil {
				return err
			}

			io.Value = io.ValueSet
			if err := c.SendPoint(io.ID, data.PointTypeValue, io.ValueSet); err != nil {
				return err
			}
			c.lastSent[io.ID] = time.Now()
		}

	case data.PointValueModbusInputRegister:
		return c.ReadBusReg(io)

	default:
		return fmt.Errorf("unhandled modbus io type: %v", io.ModbusIOType)
	}

	return nil
}

// ServerIO processes an IO on a server bus
func (c *ModbusClient) ServerIO(io *ModbusIo) error {
	if c.regs == nil || c.server == nil {
		return errors.New("server is not set up")
	}

	switch io.ModbusIOType {
	case data.PointValueModbusDiscreteInput:
		return c.regs.WriteCoil(io.Address, data.FloatToBool(io.Value))

	case data.PointValueModbusCoil:
		regValue, err := c.regs.ReadCoil(io.Address)
		if err != nil {
			return err
		}

		if regValue != data.FloatToBool(io.Value) {
			io.Value = data.BoolToFloat(regValue)
			return c.SendPoint(io.ID, data.PointTypeValue, io.Value)
		}

	case data.PointValueModbusInputRegister:
		return c.WriteReg(io)

	case data.PointValueModbusHoldingRegister:
		v, err := c.ReadReg(io)
		if err != nil {
			return err
		}

		if io.Value != v {
			io.Value = v
			return c.SendPoint(io.ID, data.PointTypeValue, v)
		}

	default:
		return fmt.Errorf("unhandled modbus io type: %v", io.ModbusIOType)
	}

	return nil
}

// modbusRegCount returns how many 16-bit registers a data format occupies
func modbusRegCount(regType string) int {
	switch regType {
	case data.PointValueUINT16, data.PointValueINT16:
		return 1
	case data.PointValueUINT32, data.PointValueINT32, data.PointValueFLOAT32:
		return 2
	default:
		log.Println("modbus: unknown data type:", regType)
		// be conservative
		return 2
	}
}

// InitRegs is used in server mode to initialize the internal modbus registers
// when an IO is set up or changes. Values are seeded from the node so the last
// known state is preserved, even for registers written by another device.
func (c *ModbusClient) InitRegs(io *ModbusIo) {
	if c.server == nil || c.regs == nil {
		return
	}

	switch io.ModbusIOType {
	case data.PointValueModbusDiscreteInput, data.PointValueModbusCoil:
		c.regs.AddCoil(io.Address)
		if err := c.regs.WriteCoil(io.Address, data.FloatToBool(io.Value)); err != nil {
			log.Println("Modbus: error writing coil:", err)
		}

	case data.PointValueModbusInputRegister, data.PointValueModbusHoldingRegister:
		c.regs.AddReg(io.Address, modbusRegCount(io.DataFormat))
		if err := c.WriteReg(io); err != nil {
			log.Println("Modbus: error writing reg:", err)
		}
	}
}

// ReadReg reads a value from an internal register. Server mode only.
func (c *ModbusClient) ReadReg(io *ModbusIo) (float64, error) {
	var valueUnscaled float64

	switch io.DataFormat {
	case data.PointValueUINT16:
		v, err := c.regs.ReadReg(io.Address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueINT16:
		v, err := c.regs.ReadReg(io.Address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(int16(v))
	case data.PointValueUINT32:
		v, err := c.regs.ReadRegUint32(io.Address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueINT32:
		v, err := c.regs.ReadRegInt32(io.Address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueFLOAT32:
		v, err := c.regs.ReadRegFloat32(io.Address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	default:
		return 0, fmt.Errorf("unhandled data type: %v", io.DataFormat)
	}

	return valueUnscaled*io.scaleFactor() + io.Offset, nil
}

// WriteReg writes an IO value to an internal register. Server mode only.
func (c *ModbusClient) WriteReg(io *ModbusIo) error {
	unscaledValue := (io.Value - io.Offset) / io.scaleFactor()

	switch io.DataFormat {
	case data.PointValueUINT16, data.PointValueINT16:
		return c.regs.WriteReg(io.Address, uint16(int32(unscaledValue)))
	case data.PointValueUINT32:
		return c.regs.WriteRegUint32(io.Address, uint32(unscaledValue))
	case data.PointValueINT32:
		return c.regs.WriteRegInt32(io.Address, int32(unscaledValue))
	case data.PointValueFLOAT32:
		return c.regs.WriteRegFloat32(io.Address, float32(unscaledValue))
	default:
		return fmt.Errorf("unhandled data type: %v", io.DataFormat)
	}
}

// LogError counts an error against the bus and the IO that saw it, and
// publishes both counts.
func (c *ModbusClient) LogError(io *ModbusIo, err error) {
	if c.config.Debug >= 1 {
		log.Printf("Modbus %v:%v, error: %v\n",
			c.config.Description, io.Description, err)
	}

	// if broken pipe error then close connection
	if errors.Is(err, syscall.EPIPE) {
		if c.config.Debug >= 1 {
			log.Println("Modbus: broken pipe, closing connection")
		}
		c.ClosePort()
	}

	var busCount, ioCount int

	errType := modbusErrorToPointType(err)
	switch errType {
	case data.PointTypeErrorCountEOF:
		c.config.ErrorCountEOF++
		io.ErrorCountEOF++
		busCount, ioCount = c.config.ErrorCountEOF, io.ErrorCountEOF
	case data.PointTypeErrorCountCRC:
		c.config.ErrorCountCRC++
		io.ErrorCountCRC++
		busCount, ioCount = c.config.ErrorCountCRC, io.ErrorCountCRC
	default:
		// probably a more general port error
		c.ioErrorCount++
		errType = data.PointTypeErrorCount
		c.config.ErrorCount++
		io.ErrorCount++
		busCount, ioCount = c.config.ErrorCount, io.ErrorCount
	}

	busPoint := data.NewPointFloat(errType, "", float64(busCount))
	if err := SendNodePoint(c.nc, c.config.ID, busPoint, false); err != nil {
		log.Println("Modbus: error sending bus error count:", err)
	}

	ioPoint := data.NewPointFloat(errType, "", float64(ioCount))
	ioPoint.Origin = c.config.ID
	if err := SendNodePoint(c.nc, io.ID, ioPoint, false); err != nil {
		log.Println("Modbus: error sending IO error count:", err)
	}
}

func modbusErrorToPointType(err error) string {
	switch {
	case errors.Is(err, goio.EOF):
		return data.PointTypeErrorCountEOF
	case errors.Is(err, modbus.ErrCRC):
		return data.PointTypeErrorCountCRC
	default:
		return ""
	}
}
