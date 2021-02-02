package node

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
	"github.com/simpleiot/simpleiot/modbus"
	"github.com/simpleiot/simpleiot/nats"
	"github.com/simpleiot/simpleiot/respreader"
	"go.bug.st/serial"
)

type pointWID struct {
	id    string
	point data.Point
}

// Modbus describes a modbus bus
type Modbus struct {
	// node data should only be changed through NATS, so that it is only changed in one place
	node    *data.NodeEdge
	busNode *ModbusNode
	ios     map[string]*ModbusIO

	// data associated with running the bus
	db     *genji.Db
	nc     *natsgo.Conn
	sub    *natsgo.Subscription
	client *modbus.Client
	server *modbus.Server
	port   io.ReadWriteCloser

	chDone      chan bool
	chPoint     chan pointWID
	chError     <-chan error
	chRegChange chan bool
}

// NewModbus creates a new bus from a node
func NewModbus(db *genji.Db, nc *natsgo.Conn, node *data.NodeEdge) (*Modbus, error) {
	bus := &Modbus{
		db:          db,
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
		bus.sub, err = nc.Subscribe("node."+bus.busNode.nodeID+".points", func(msg *natsgo.Msg) {
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
	nodes, err := b.db.NodeChildren(b.busNode.nodeID, data.NodeTypeModbusIO)
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

		}
	}

	// remove ios that have been deleted
	for id, io := range b.ios {
		_, ok := found[id]
		if !ok {
			// bus was deleted so close and clear it
			log.Println("modbus io removed: ", io.ioNode.description)
			// FIXME, do we need to do anything here
			delete(b.ios, id)
			b.Stop()
		}
	}

	return nil
}

// SendPoint sends a point over nats
func (b *Modbus) SendPoint(nodeID, pointType string, value float64) error {
	// send the point
	p := data.Point{
		Type:  pointType,
		Value: value,
	}

	return nats.SendPoint(b.nc, nodeID, p, true)
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

		if io.ioNode.valueSet != io.ioNode.value {
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

		if io.ioNode.valueSet != io.ioNode.value {
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
		b.server.Regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusCoil:
		regValue, err := b.server.Regs.ReadCoil(io.address)
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
		b.server.Regs.AddCoil(io.address)
		b.server.Regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusCoil:
		b.server.Regs.AddCoil(io.address)
		b.server.Regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusInputRegister:
		b.server.Regs.AddReg(io.address, regCount(io.modbusDataType))
		b.WriteReg(io)
	case data.PointValueModbusHoldingRegister:
		b.server.Regs.AddReg(io.address, regCount(io.modbusDataType))
		b.WriteReg(io)
	}
}

// ReadReg reads an value from a reg (internal, not bus)
// This should only be used on server
func (b *Modbus) ReadReg(io *ModbusIONode) (float64, error) {
	var valueUnscaled float64
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		v, err := b.server.Regs.ReadReg(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueUINT32:
		v, err := b.server.Regs.ReadRegUint32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueINT32:
		v, err := b.server.Regs.ReadRegInt32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueFLOAT32:
		v, err := b.server.Regs.ReadRegFloat32(io.address)
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
		b.server.Regs.WriteReg(io.address, uint16(unscaledValue))
	case data.PointValueUINT32:
		b.server.Regs.WriteRegUint32(io.address,
			uint32(unscaledValue))
	case data.PointValueINT32:
		b.server.Regs.WriteRegInt32(io.address,
			int32(unscaledValue))
	case data.PointValueFLOAT32:
		b.server.Regs.WriteRegFloat32(io.address,
			float32(unscaledValue))
	default:
		return fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)
	}
	return nil
}

// LogError ...
func (b *Modbus) LogError(io *ModbusIONode, typ string) error {
	busCount := 0
	ioCount := 0
	switch typ {
	case data.PointTypeErrorCount:
		busCount = b.busNode.errorCount
		ioCount = io.errorCount
		b.busNode.errorCount++
		io.errorCount++
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
		return fmt.Errorf("Unknown error type to log: %v", typ)
	}

	busCount++
	ioCount++

	p := data.Point{
		Type:  typ,
		Value: float64(busCount),
	}

	err := nats.SendPoint(b.nc, b.busNode.nodeID, p, true)
	if err != nil {
		return err
	}

	p.Value = float64(ioCount)
	return nats.SendPoint(b.nc, io.nodeID, p, true)
}

// SetupPort sets up io for the bus
func (b *Modbus) SetupPort() error {
	if b.busNode.debugLevel >= 1 {
		log.Println("modbus: setting up serial port: ", b.busNode.portName)
	}
	if b.server != nil {
		b.server.Close()
		b.server = nil
	}

	if b.port != nil {
		b.port.Close()
		b.port = nil
	}

	mode := &serial.Mode{
		BaudRate: b.busNode.baud,
	}

	serialPort, err := serial.Open(b.busNode.portName, mode)
	if err != nil {
		return fmt.Errorf("Error opening serial port: %w", err)
	}

	b.port = respreader.NewReadWriteCloser(serialPort, time.Millisecond*200, time.Millisecond*30)

	if b.busNode.busType == data.PointValueServer {
		b.server = modbus.NewServer(byte(b.busNode.id), b.port)
		go b.server.Listen(b.busNode.debugLevel, func(err error) {
			log.Println("Modbus server error: ", err)
		}, func() {
			log.Println("Modbus reg change")
			b.chRegChange <- true
		})

		for _, io := range b.ios {
			b.InitRegs(io.ioNode)
		}
	} else if b.busNode.busType == data.PointValueClient {
		b.client = modbus.NewClient(b.port, b.busNode.debugLevel)
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
	scanTimer := time.NewTicker(time.Millisecond * time.Duration(b.busNode.pollPeriod))
	checkIoTimer := time.NewTicker(time.Second * 10)

	b.CheckIOs()
	b.SetupPort()

	log.Println("initializing modbus port: ", b.busNode.portName)

	for {
		select {
		case point := <-b.chPoint:
			p := point.point
			if point.id == b.busNode.nodeID {
				b.node.ProcessPoint(p)
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
					data.PointTypeBaud:
					err := b.SetupPort()
					if err != nil {
						log.Println("Error setting up serial port: ", err)
					}
				case data.PointTypePollPeriod:
					scanTimer = time.NewTicker(time.Millisecond * time.Duration(b.busNode.pollPeriod))

				case data.PointTypeErrorCountReset:
					if b.busNode.errorCountReset {
						p := data.Point{Type: data.PointTypeErrorCount, Value: 0}
						err := nats.SendPoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountReset, Value: 0}
						err = nats.SendPoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}

				case data.PointTypeErrorCountCRCReset:
					if b.busNode.errorCountCRCReset {
						p := data.Point{Type: data.PointTypeErrorCountCRC, Value: 0}
						err := nats.SendPoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountCRCReset, Value: 0}
						err = nats.SendPoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}

				case data.PointTypeErrorCountEOFReset:
					if b.busNode.errorCountEOFReset {
						p := data.Point{Type: data.PointTypeErrorCountEOF, Value: 0}
						err := nats.SendPoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountEOFReset, Value: 0}
						err = nats.SendPoint(b.nc, b.busNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}
				}
			} else {
				io, ok := b.ios[point.id]
				if !ok {
					log.Println("received point for unknown IO: ", point.id)
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
						err := nats.SendPoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountReset, Value: 0}
						err = nats.SendPoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}

				case data.PointTypeErrorCountEOFReset:
					io.ioNode.errorCountEOFReset = data.FloatToBool(p.Value)
					if io.ioNode.errorCountEOFReset {
						p := data.Point{Type: data.PointTypeErrorCountEOF, Value: 0}
						err := nats.SendPoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountEOFReset, Value: 0}
						err = nats.SendPoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}
					}

				case data.PointTypeErrorCountCRCReset:
					io.ioNode.errorCountCRCReset = data.FloatToBool(p.Value)
					if io.ioNode.errorCountCRCReset {
						p := data.Point{Type: data.PointTypeErrorCountCRC, Value: 0}
						err := nats.SendPoint(b.nc, io.ioNode.nodeID, p, true)
						if err != nil {
							log.Println("Send point error: ", err)
						}

						p = data.Point{Type: data.PointTypeErrorCountCRCReset, Value: 0}
						err = nats.SendPoint(b.nc, io.ioNode.nodeID, p, true)
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
						b.LogError(io.ioNode, modbusErrorToPointType(err))
					}
				}

				if valueSetModified && (b.busNode.busType == data.PointValueClient) &&
					(io.ioNode.modbusDataType == data.PointValueModbusCoil ||
						io.ioNode.modbusDataType == data.PointValueModbusHoldingRegister) &&
					(io.ioNode.value != io.ioNode.valueSet) {
					err := b.ClientIO(io)
					if err != nil {
						b.LogError(io.ioNode, modbusErrorToPointType(err))
					}
				}
			}
		case <-b.chRegChange:
			// this only happens on modbus servers
			for _, io := range b.ios {
				err := b.ServerIO(io.ioNode)
				if err != nil {
					log.Printf("Modbus server %v:%v, error: %v\n",
						b.busNode.portName, io.ioNode.description, err)
					err := b.LogError(io.ioNode, modbusErrorToPointType(err))
					if err != nil {
						log.Println("Error logging modbus error: ", err)
					}
				}
			}

		case <-checkIoTimer.C:
			b.CheckIOs()

		case <-scanTimer.C:
			for _, io := range b.ios {
				// for scanning, we only need to process client ios
				if b.busNode.busType == data.PointValueClient {
					err := b.ClientIO(io)
					if err != nil {
						if b.busNode.debugLevel >= 1 {
							log.Printf("Modbus client %v:%v, error: %v\n",
								b.busNode.portName, io.ioNode.description, err)
						}
						err := b.LogError(io.ioNode, modbusErrorToPointType(err))
						if err != nil {
							log.Println("Error logging modbus error: ", err)
						}
					}
				}
			}
		case <-b.chDone:
			log.Println("Stopping client IO for: ", b.busNode.portName)
			b.port.Close()
			if b.server != nil {
				b.server.Close()
			}
			return
		}
	}
}
