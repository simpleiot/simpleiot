package node

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/modbus"
	"github.com/simpleiot/simpleiot/respreader"
	"go.bug.st/serial"
)

type pointWID struct {
	id    string
	point data.Point
}

type server interface {
	Close() error
	Listen(func(error), func(), func())
}

// Modbus describes a modbus bus
type Modbus struct {
	// node data should only be changed through NATS, so that it is only changed in one place
	node    data.NodeEdge
	busNode *ModbusNode
	ios     map[string]*ModbusIO

	// data associated with running the bus
	nc           *nats.Conn
	sub          *nats.Subscription
	regs         *modbus.Regs
	client       *modbus.Client
	server       server
	serialPort   serial.Port
	ioErrorCount int

	chDone      chan bool
	chPoint     chan pointWID
	chError     <-chan error
	chRegChange chan bool
}

// NewModbus creates a new bus from a node
func NewModbus(nc *nats.Conn, node data.NodeEdge) (*Modbus, error) {
	bus := &Modbus{
		nc:          nc,
		node:        node,
		ios:         make(map[string]*ModbusIO),
		chDone:      make(chan bool),
		chPoint:     make(chan pointWID),
		chRegChange: make(chan bool),
	}

	modbusNode, err := NewModbusNode(node)
	if err != nil {
		return nil, err
	}

	bus.busNode = modbusNode

	// closure is required so we don't get races accessing bus.busNode
	func(id string) {
		bus.sub, err = nc.Subscribe("node."+bus.busNode.nodeID+".points", func(msg *nats.Msg) {
			points, err := data.PbDecodePoints(msg.Data)
			if err != nil {
				// FIXME, send over channel
				log.Println("Error decoding node data: ", err)
				return
			}

			for _, p := range points {
				bus.chPoint <- pointWID{id, p}
			}
		})
	}(bus.busNode.nodeID)

	if err != nil {
		return nil, err
	}

	go bus.Run()

	return bus, nil
}

// Stop stops the bus and resets various fields
func (b *Modbus) Stop() {
	if b.sub != nil {
		err := b.sub.Unsubscribe()
		if err != nil {
			log.Println("Error unsubscribing from bus: ", err)
		}
	}
	for _, io := range b.ios {
		io.Stop()
	}
	b.chDone <- true
}

// CheckIOs goes through ios on the bus and handles any config changes
func (b *Modbus) CheckIOs() error {
	nodes, err := client.GetNodes(b.nc, b.busNode.nodeID, "all", data.NodeTypeModbusIO, false)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, node := range nodes {
		found[node.ID] = true
		io, ok := b.ios[node.ID]
		if !ok {
			// add ios
			var err error
			ioNode, err := NewModbusIONode(b.busNode.busType, &node)
			if err != nil {
				log.Println("Error with IO node: ", err)
				continue
			}
			io, err = NewModbusIO(b.nc, ioNode, b.chPoint)
			if err != nil {
				log.Println("Error creating new modbus IO: ", err)
				continue
			}
			b.ios[node.ID] = io
			b.InitRegs(io.ioNode)
		}
	}

	// remove ios that have been deleted
	for id, io := range b.ios {
		_, ok := found[id]
		if !ok {
			// io was deleted so close and clear it
			log.Println("modbus io removed: ", io.ioNode.description)
			io.Stop()
			delete(b.ios, id)
		}
	}

	return nil
}

// SendPoint sends a point over nats
func (b *Modbus) SendPoint(nodeID, pointType string, value float64) error {
	// send the point
	p := data.Point{
		Time:  time.Now(),
		Type:  pointType,
		Value: value,
	}

	return client.SendNodePoint(b.nc, nodeID, p, true)
}

// WriteBusHoldingReg used to write register values to bus
// should only be used by client
func (b *Modbus) WriteBusHoldingReg(io *ModbusIONode) error {
	unscaledValue := (io.valueSet - io.offset) / io.scale
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		err := b.client.WriteSingleReg(byte(io.id),
			uint16(io.address), uint16(unscaledValue))
		if err != nil {
			return err
		}
	case data.PointValueUINT32:
		regs := modbus.Uint32ToRegs([]uint32{uint32(unscaledValue)})
		err := b.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = b.client.WriteSingleReg(byte(io.id),
			uint16(io.address+1), regs[1])
		if err != nil {
			return err
		}

	case data.PointValueINT32:
		regs := modbus.Int32ToRegs([]int32{int32(unscaledValue)})
		err := b.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = b.client.WriteSingleReg(byte(io.id),
			uint16(io.address+1), regs[1])
		if err != nil {
			return err
		}

	case data.PointValueFLOAT32:
		regs := modbus.Float32ToRegs([]float32{float32(unscaledValue)})
		err := b.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = b.client.WriteSingleReg(byte(io.id),
			uint16(io.address+1), regs[1])
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)

	}

	return nil
}

// ReadBusReg reads an io value from a reg from bus
// this function modifies io.value
func (b *Modbus) ReadBusReg(io *ModbusIO) error {
	readFunc := b.client.ReadHoldingRegs
	switch io.ioNode.modbusIOType {
	case data.PointValueModbusHoldingRegister:
	case data.PointValueModbusInputRegister:
		readFunc = b.client.ReadInputRegs
	default:
		return fmt.Errorf("ReadBusReg: unsupported modbus IO type: %v",
			io.ioNode.modbusIOType)
	}
	var valueUnscaled float64
	switch io.ioNode.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		regs, err := readFunc(byte(io.ioNode.id), uint16(io.ioNode.address), 1)
		if err != nil {
			return err
		}
		if len(regs) < 1 {
			return errors.New("Did not receive enough data")
		}
		valueUnscaled = float64(regs[0])

	case data.PointValueUINT32:
		regs, err := readFunc(byte(io.ioNode.id), uint16(io.ioNode.address), 2)
		if err != nil {
			return err
		}
		if len(regs) < 2 {
			return errors.New("Did not receive enough data")
		}
		v := modbus.RegsToUint32(regs)

		valueUnscaled = float64(v[0])

	case data.PointValueINT32:
		regs, err := readFunc(byte(io.ioNode.id), uint16(io.ioNode.address), 2)
		if err != nil {
			return err
		}
		if len(regs) < 2 {
			return errors.New("Did not receive enough data")
		}
		v := modbus.RegsToInt32(regs)

		valueUnscaled = float64(v[0])

	case data.PointValueFLOAT32:
		regs, err := readFunc(byte(io.ioNode.id), uint16(io.ioNode.address), 2)
		if err != nil {
			return err
		}
		if len(regs) < 2 {
			return errors.New("Did not receive enough data")
		}
		valueUnscaled = float64(modbus.RegsToFloat32(regs)[0])

	default:
		return fmt.Errorf("unhandled data type: %v",
			io.ioNode.modbusDataType)
	}

	value := valueUnscaled*io.ioNode.scale + io.ioNode.offset

	if value != io.ioNode.value || time.Since(io.lastSent) > time.Minute*10 {
		io.ioNode.value = value
		err := b.SendPoint(io.ioNode.nodeID, data.PointTypeValue, value)
		if err != nil {
			return err
		}
		io.lastSent = time.Now()
	}

	return nil
}

// ReadBusBit is used to read coil of discrete input values from bus
// this function modifies io.value. This should only be called from client.
func (b *Modbus) ReadBusBit(io *ModbusIO) error {
	readFunc := b.client.ReadCoils
	switch io.ioNode.modbusIOType {
	case data.PointValueModbusCoil:
	case data.PointValueModbusDiscreteInput:
		readFunc = b.client.ReadDiscreteInputs
	default:
		return fmt.Errorf("ReadBusBit: unhandled modbusIOType: %v",
			io.ioNode.modbusIOType)
	}
	bits, err := readFunc(byte(io.ioNode.id), uint16(io.ioNode.address), 1)
	if err != nil {
		return err
	}
	if len(bits) < 1 {
		return errors.New("Did not receive enough data")
	}

	value := data.BoolToFloat(bits[0])

	if value != io.ioNode.value || time.Since(io.lastSent) > time.Minute*10 {
		io.ioNode.value = value
		err := b.SendPoint(io.ioNode.nodeID, data.PointTypeValue, value)
		if err != nil {
			return err
		}

		io.lastSent = time.Now()
	}

	io.ioNode.value = value

	return nil
}

// ClientIO processes an IO on a client bus
func (b *Modbus) ClientIO(io *ModbusIO) error {

	if b.client == nil {
		return errors.New("client is not set up")
	}

	// read value from remote device and update regs
	switch io.ioNode.modbusIOType {
	case data.PointValueModbusCoil:
		err := b.ReadBusBit(io)
		if err != nil {
			return err
		}

		if !io.ioNode.readOnly && io.ioNode.valueSet != io.ioNode.value {
			vBool := data.FloatToBool(io.ioNode.valueSet)
			// we need set the remote value
			err := b.client.WriteSingleCoil(byte(io.ioNode.id), uint16(io.ioNode.address),
				vBool)

			if err != nil {
				return err
			}

			err = b.SendPoint(io.ioNode.nodeID, data.PointTypeValue, io.ioNode.valueSet)
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusDiscreteInput:
		err := b.ReadBusBit(io)
		if err != nil {
			return err
		}

	case data.PointValueModbusHoldingRegister:
		err := b.ReadBusReg(io)
		if err != nil {
			return err
		}

		if !io.ioNode.readOnly && io.ioNode.valueSet != io.ioNode.value {
			// we need set the remote value
			err := b.WriteBusHoldingReg(io.ioNode)

			if err != nil {
				return err
			}

			err = b.SendPoint(io.ioNode.nodeID, data.PointTypeValue, io.ioNode.valueSet)
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusInputRegister:
		err := b.ReadBusReg(io)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unhandled modbus io type, io: %+v", io)
	}

	return nil
}

// ServerIO processes an IO on a server bus
func (b *Modbus) ServerIO(io *ModbusIONode) error {
	// update regs with db value
	switch io.modbusIOType {
	case data.PointValueModbusDiscreteInput:
		b.regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusCoil:
		regValue, err := b.regs.ReadCoil(io.address)
		if err != nil {
			return err
		}

		dbValue := data.FloatToBool(io.value)

		if regValue != dbValue {
			err = b.SendPoint(io.nodeID, data.PointTypeValue, data.BoolToFloat(regValue))
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusInputRegister:
		b.WriteReg(io)

	case data.PointValueModbusHoldingRegister:
		v, err := b.ReadReg(io)
		if err != nil {
			return err
		}

		if io.value != v {
			err = b.SendPoint(io.nodeID, data.PointTypeValue, v)
			if err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unhandled modbus io type: %v", io.modbusIOType)
	}

	return nil
}

func regCount(regType string) int {
	switch regType {
	case data.PointValueUINT16, data.PointValueINT16:
		return 1
	case data.PointValueUINT32, data.PointValueINT32,
		data.PointValueFLOAT32:
		return 2
	default:
		log.Println("regCount, unknown data type: ", regType)
		// be conservative
		return 2
	}
}

// InitRegs is used in server mode to initilize the internal modbus regs when a IO changes
func (b *Modbus) InitRegs(io *ModbusIONode) {
	if b.server == nil {
		return
	}

	// we initialize all values from database, even if they are written from
	// another device so that we preserve the last known state
	switch io.modbusIOType {
	case data.PointValueModbusDiscreteInput:
		b.regs.AddCoil(io.address)
		b.regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusCoil:
		b.regs.AddCoil(io.address)
		b.regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusInputRegister:
		b.regs.AddReg(io.address, regCount(io.modbusDataType))
		b.WriteReg(io)
	case data.PointValueModbusHoldingRegister:
		b.regs.AddReg(io.address, regCount(io.modbusDataType))
		b.WriteReg(io)
	}
}

// ReadReg reads an value from a reg (internal, not bus)
// This should only be used on server
func (b *Modbus) ReadReg(io *ModbusIONode) (float64, error) {
	var valueUnscaled float64
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		v, err := b.regs.ReadReg(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueUINT32:
		v, err := b.regs.ReadRegUint32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueINT32:
		v, err := b.regs.ReadRegInt32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueFLOAT32:
		v, err := b.regs.ReadRegFloat32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	default:
		return 0, fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)
	}
	return valueUnscaled*io.scale + io.offset, nil
}

// WriteReg writes an io value to a reg
// This should only be used on server
func (b *Modbus) WriteReg(io *ModbusIONode) error {
	unscaledValue := (io.value - io.offset) / io.scale
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		b.regs.WriteReg(io.address, uint16(unscaledValue))
	case data.PointValueUINT32:
		b.regs.WriteRegUint32(io.address,
			uint32(unscaledValue))
	case data.PointValueINT32:
		b.regs.WriteRegInt32(io.address,
			int32(unscaledValue))
	case data.PointValueFLOAT32:
		b.regs.WriteRegFloat32(io.address,
			float32(unscaledValue))
	default:
		return fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)
	}
	return nil
}

// LogError ...
func (b *Modbus) LogError(io *ModbusIONode, err error) error {
	busCount := 0
	ioCount := 0

	if b.busNode.debugLevel >= 1 {
		log.Printf("Modbus %v:%v, error: %v\n",
			b.busNode.portName, io.description, err)
	}

	// if broken pipe error then close connection
	if errors.Is(err, syscall.EPIPE) {
		if b.busNode.debugLevel >= 1 {
			log.Printf("Broken pipe, closing connection")
		}
		b.ClosePort()
	}

	errType := modbusErrorToPointType(err)
	switch errType {
	case data.PointTypeErrorCountEOF:
		busCount = b.busNode.errorCountEOF
		ioCount = io.errorCountEOF
		b.busNode.errorCountEOF++
		io.errorCountEOF++
	case data.PointTypeErrorCountCRC:
		busCount = b.busNode.errorCountCRC
		ioCount = io.errorCountCRC
		b.busNode.errorCountCRC++
		io.errorCountCRC++
	default:
		// probably a more general serial port error
		b.ioErrorCount++
		errType = data.PointTypeErrorCount
		busCount = b.busNode.errorCount
		ioCount = io.errorCount
		b.busNode.errorCount++
		io.errorCount++
	}

	busCount++
	ioCount++

	p := data.Point{
		Type:  errType,
		Value: float64(busCount),
	}

	err = client.SendNodePoint(b.nc, b.busNode.nodeID, p, false)
	if err != nil {
		return err
	}

	p.Value = float64(ioCount)
	return client.SendNodePoint(b.nc, io.nodeID, p, false)
}

// ClosePort closes both the server and client ports
func (b *Modbus) ClosePort() {
	if b.server != nil {
		err := b.server.Close()
		if err != nil {
			log.Println("Error closing server: ", err)
		}
		b.server = nil
	}

	if b.client != nil {
		err := b.client.Close()
		if err != nil {
			log.Println("Error closing client: ", err)
		}
		b.client = nil
	}
}

// SetupPort sets up io for the bus
func (b *Modbus) SetupPort() error {
	if b.busNode.debugLevel >= 1 {
		log.Println("modbus: setting up modbus transport: ", b.busNode.portName)
	}

	b.ClosePort()

	var transport modbus.Transport

	switch b.busNode.protocol {
	case data.PointValueRTU:
		mode := &serial.Mode{
			BaudRate: b.busNode.baud,
		}

		var err error
		b.serialPort, err = serial.Open(b.busNode.portName, mode)
		if err != nil {
			b.serialPort = nil
			return fmt.Errorf("Error opening serial port: %w", err)
		}

		port := respreader.NewReadWriteCloser(b.serialPort, time.Millisecond*100, time.Millisecond*20)

		transport = modbus.NewRTU(port)
	case data.PointValueTCP:
		switch b.busNode.busType {
		case data.PointValueClient:
			sock, err := net.DialTimeout("tcp", b.busNode.uri, 5*time.Second)
			if err != nil {
				return err
			}
			transport = modbus.NewTCP(sock, 500*time.Millisecond,
				modbus.TransportClient)
		case data.PointValueServer:
			// TCPServer does all the setup
		default:
			log.Println("setting up modbus TCP, invalid bus type: ", b.busNode.busType)
		}

	default:
		return fmt.Errorf("Unsupported modbus protocol: %v", b.busNode.protocol)
	}

	if b.busNode.busType == data.PointValueServer {
		b.regs = &modbus.Regs{}
		if b.busNode.protocol == data.PointValueRTU {
			b.server = modbus.NewServer(byte(b.busNode.id), transport,
				b.regs, b.busNode.debugLevel)
		} else if b.busNode.protocol == data.PointValueTCP {
			var err error
			b.server, err = modbus.NewTCPServer(b.busNode.id, 5,
				b.busNode.portName, b.regs, b.busNode.debugLevel)
			if err != nil {
				b.server = nil
				return err
			}
		} else {
			return errors.New("Modbus protocol not set")
		}

		go b.server.Listen(func(err error) {
			log.Println("Modbus server error: ", err)
		}, func() {
			if b.busNode.debugLevel > 0 {
				log.Println("Modbus reg change")
			}
			b.chRegChange <- true
		}, func() {
			if b.busNode.debugLevel > 0 {
				log.Println("Modbus Listener done")
			}
		})

		for _, io := range b.ios {
			b.InitRegs(io.ioNode)
		}
	} else if b.busNode.busType == data.PointValueClient {
		b.client = modbus.NewClient(transport, b.busNode.debugLevel)
	}

	return nil
}

// Run is routine that runs the logic for a bus. Intended to be run as
// a goroutine
// It assumes an initial dataset is obtained from the database and all updates
// come from NATs
// this routine may need to run fast scan times, so it should be doing
// slow things like reading the database.
func (b *Modbus) Run() {

	// if we reset any error count, we set this to avoid continually resetting
	scanTimer := time.NewTicker(24 * time.Hour)

	setScanTimer := func() {
		if b.busNode.busType == data.PointValueClient {
			scanTimer.Reset(time.Millisecond * time.Duration(b.busNode.pollPeriod))
		} else {
			scanTimer.Stop()
		}
	}

	setScanTimer()

	checkIoTimer := time.NewTicker(time.Second * 10)

	log.Println("initializing modbus port: ", b.busNode.portName)

	for {
		select {
		case point := <-b.chPoint:
			p := point.point
			if point.id == b.busNode.nodeID {
				b.node.AddPoint(p)
				var err error
				b.busNode, err = NewModbusNode(b.node)
				if err != nil {
					log.Println("Error updating bus node: ", err)
				}

				switch point.point.Type {
				case data.PointTypeClientServer,
					data.PointTypeID,
					data.PointTypeDebug,
					data.PointTypePort,
					data.PointTypeBaud,
					data.PointTypeURI:
					err := b.SetupPort()
					if err != nil {
						log.Println("Error setting up serial port: ", err)
					}
				case data.PointTypePollPeriod:
					setScanTimer()

				case data.PointTypeErrorCountReset:
					if b.busNode.errorCountReset {
						p := data.Point{Type: data.PointTypeErrorCount, Value: 0}
						err := client.SendNodePoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountReset, Value: 0}
						err = client.SendNodePoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}

				case data.PointTypeErrorCountCRCReset:
					if b.busNode.errorCountCRCReset {
						p := data.Point{Type: data.PointTypeErrorCountCRC, Value: 0}
						err := client.SendNodePoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountCRCReset, Value: 0}
						err = client.SendNodePoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}

				case data.PointTypeErrorCountEOFReset:
					if b.busNode.errorCountEOFReset {
						p := data.Point{Type: data.PointTypeErrorCountEOF, Value: 0}
						err := client.SendNodePoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountEOFReset, Value: 0}
						err = client.SendNodePoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}
				}
			} else {
				io, ok := b.ios[point.id]
				if !ok {
					log.Println("modbus received point for unknown node: ", point.id)
					// FIXME, we could create a new IO here
					continue
				}

				valueModified := false
				valueSetModified := false

				// handle IO changes
				switch p.Type {
				case data.PointTypeID:
					io.ioNode.id = int(p.Value)
				case data.PointTypeDescription:
					io.ioNode.description = p.Text
				case data.PointTypeAddress:
					io.ioNode.address = int(p.Value)
					b.InitRegs(io.ioNode)
				case data.PointTypeModbusIOType:
					io.ioNode.modbusIOType = p.Text
				case data.PointTypeDataFormat:
					io.ioNode.modbusDataType = p.Text
				case data.PointTypeReadOnly:
					io.ioNode.readOnly = data.FloatToBool(p.Value)
				case data.PointTypeScale:
					io.ioNode.scale = p.Value
				case data.PointTypeOffset:
					io.ioNode.offset = p.Value
				case data.PointTypeValue:
					valueModified = true
					io.ioNode.value = p.Value
				case data.PointTypeValueSet:
					valueSetModified = true
					io.ioNode.valueSet = p.Value
				case data.PointTypeDisable:
					io.ioNode.disable = data.FloatToBool(p.Value)
				case data.PointTypeErrorCount:
					io.ioNode.errorCount = int(p.Value)
				case data.PointTypeErrorCountEOF:
					io.ioNode.errorCountEOF = int(p.Value)
				case data.PointTypeErrorCountCRC:
					io.ioNode.errorCountCRC = int(p.Value)
				case data.PointTypeErrorCountReset:
					io.ioNode.errorCountReset = data.FloatToBool(p.Value)
					if io.ioNode.errorCountReset {
						p := data.Point{Type: data.PointTypeErrorCount, Value: 0}
						err := client.SendNodePoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountReset, Value: 0}
						err = client.SendNodePoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}

				case data.PointTypeErrorCountEOFReset:
					io.ioNode.errorCountEOFReset = data.FloatToBool(p.Value)
					if io.ioNode.errorCountEOFReset {
						p := data.Point{Type: data.PointTypeErrorCountEOF, Value: 0}
						err := client.SendNodePoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountEOFReset, Value: 0}
						err = client.SendNodePoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}

				case data.PointTypeErrorCountCRCReset:
					io.ioNode.errorCountCRCReset = data.FloatToBool(p.Value)
					if io.ioNode.errorCountCRCReset {
						p := data.Point{Type: data.PointTypeErrorCountCRC, Value: 0}
						err := client.SendNodePoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountCRCReset, Value: 0}
						err = client.SendNodePoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}
				default:
					log.Println("modbus: unhandled io point: ", p)
				}

				if valueModified && b.busNode.busType == data.PointValueServer {
					err := b.ServerIO(io.ioNode)
					if err != nil {
						b.LogError(io.ioNode, err)
					}
				}

				if valueSetModified && (b.busNode.busType == data.PointValueClient) &&
					(io.ioNode.modbusDataType == data.PointValueModbusCoil ||
						io.ioNode.modbusDataType == data.PointValueModbusHoldingRegister) &&
					(io.ioNode.value != io.ioNode.valueSet) {
					err := b.ClientIO(io)
					if err != nil {
						b.LogError(io.ioNode, err)
					}
				}
			}
		case <-b.chRegChange:
			// this only happens on modbus servers
			for _, io := range b.ios {
				err := b.ServerIO(io.ioNode)
				if err != nil {
					err := b.LogError(io.ioNode, err)
					if err != nil {
						log.Println("Error logging modbus error: ", err)
					}
				}
			}

		case <-checkIoTimer.C:
			var portError error
			if b.serialPort != nil {
				// the following handles cases where serial port
				// may have been unplugged and plugged back in
				_, portError = b.serialPort.GetModemStatusBits()
			}

			if b.busNode.disable {
				b.ClosePort()
			} else {
				if (b.client == nil && b.server == nil) ||
					b.ioErrorCount > 10 || portError != nil {
					if b.busNode.debugLevel >= 1 {
						log.Printf("Re-initializing modbus port, err cnt: %v, portError: %v\n", b.ioErrorCount, portError)
					}
					b.ioErrorCount = 0
					// try to set up port
					if err := b.SetupPort(); err != nil {
						log.Println("SetupPort error: ", err)
					}
				}
				if err := b.CheckIOs(); err != nil {
					log.Println("CheckIOs error: ", err)
				}
			}

		case <-scanTimer.C:
			if b.busNode.busType == data.PointValueClient && !b.busNode.disable {
				for _, io := range b.ios {
					if io.ioNode.disable {
						continue
					}
					// for scanning, we only need to process client ios
					err := b.ClientIO(io)
					if err != nil {
						err := b.LogError(io.ioNode, err)
						if err != nil {
							log.Println("Error logging modbus error: ", err)
						}
					}
				}
			}
		case <-b.chDone:
			log.Println("Stopping client IO for: ", b.busNode.portName)
			b.ClosePort()
			return
		}
	}
}

func modbusErrorToPointType(err error) string {
	switch err {
	case io.EOF:
		return data.PointTypeErrorCountEOF
	case modbus.ErrCRC:
		return data.PointTypeErrorCountCRC
	default:
		return ""
	}
}
