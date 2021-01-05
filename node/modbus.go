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

// ModbusManager manages state of modbus
type ModbusManager struct {
	db     *genji.Db
	nc     *natsgo.Conn
	busses map[string]*Modbus
}

// NewModbusManager creates a new modbus manager
func NewModbusManager(db *genji.Db, nc *natsgo.Conn) *ModbusManager {
	return &ModbusManager{
		db:     db,
		nc:     nc,
		busses: make(map[string]*Modbus),
	}
}

func modbusErrorToPointType(err error) string {
	switch err {
	case io.EOF:
		return data.PointTypeErrorCountEOF
	case modbus.ErrCRC:
		return data.PointTypeErrorCountCRC
	default:
		return data.PointTypeErrorCount
	}
}

func copyIos(in map[string]*ModbusIO) map[string]*ModbusIO {
	out := make(map[string]*ModbusIO)
	for k, v := range in {
		io := *v
		out[k] = &io
	}
	return out
}

// Update queries DB for modbus nodes and synchronizes
// with internal structures and updates data
func (mm *ModbusManager) Update() error {
	rootID := mm.db.RootNodeID()
	busNodes, err := mm.db.NodeChildren(rootID, data.NodeTypeModbus)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, busNode := range busNodes {
		found[busNode.ID] = true
		bus, ok := mm.busses[busNode.ID]
		if !ok {
			var err error
			bus, err = NewModbus(mm.db, mm.nc, &busNode)
			if err != nil {
				log.Println("Error creating new modbus: ", err)
				continue
			}
			mm.busses[busNode.ID] = bus
		}

		err := bus.CheckPort(&busNode)

		if err != nil {
			log.Println("Error initializing modbus port: ",
				busNode.ID, err)
			continue
		}

		if !bus.running {
			go bus.Run()
			bus.chIOs <- copyIos(bus.ios)
			bus.running = true
		}

		changed, err := bus.CheckIOs()
		if err != nil {
			log.Println("Error checking modbus IOs: ", err)
			continue
		}

		if changed {
			bus.chIOs <- copyIos(bus.ios)
		}
	}

	// remove busses that have been deleted
	for id, bus := range mm.busses {
		_, ok := found[id]
		if !ok {
			// bus was deleted so close and clear it
			log.Println("Closing modbus on port: ", bus.portName)
			err := bus.port.Close()
			if err != nil {
				log.Println("Error closing modbus port: ", err)
			}

			if bus.running {
				bus.Stop()
			}

			delete(mm.busses, id)
		}
	}

	return nil
}

// Modbus describes a modbus bus
type Modbus struct {
	db                 *genji.Db
	nc                 *natsgo.Conn
	nodeID             string
	busType            string
	id                 int // only used for server
	portName           string
	baud               int
	port               *respreader.ReadWriteCloser
	client             *modbus.Client
	server             *modbus.Server
	debugLevel         int
	chStop             chan bool
	chIOs              chan map[string]*ModbusIO
	running            bool
	pollPeriod         int
	errorCount         int
	errorCountCRC      int
	errorCountEOF      int
	errorCountReset    bool
	errorCountCRCReset bool
	errorCountEOFReset bool
	ios                map[string]*ModbusIO
}

func nodeToModbus(node *data.NodeEdge) (*Modbus, error) {
	ret := Modbus{
		nodeID: node.ID,
	}

	var ok bool

	ret.busType, ok = node.Points.Text("", data.PointTypeClientServer, 0)
	if !ok {
		return nil, errors.New("Must define modbus client/server")
	}
	ret.portName, ok = node.Points.Text("", data.PointTypePort, 0)
	if !ok {
		return nil, errors.New("Must define modbus port name")
	}
	ret.baud, ok = node.Points.ValueInt("", data.PointTypeBaud, 0)
	if !ok {
		return nil, errors.New("Must define modbus baud")
	}

	ret.pollPeriod, ok = node.Points.ValueInt("", data.PointTypePollPeriod, 0)
	if !ok {
		return nil, errors.New("Must define modbus polling period")
	}

	ret.debugLevel, _ = node.Points.ValueInt("", data.PointTypeDebug, 0)
	ret.errorCount, _ = node.Points.ValueInt("", data.PointTypeErrorCount, 0)
	ret.errorCountCRC, _ = node.Points.ValueInt("", data.PointTypeErrorCountCRC, 0)
	ret.errorCountEOF, _ = node.Points.ValueInt("", data.PointTypeErrorCountEOF, 0)
	ret.errorCountReset, _ = node.Points.ValueBool("", data.PointTypeErrorCountReset, 0)
	ret.errorCountCRCReset, _ = node.Points.ValueBool("", data.PointTypeErrorCountCRCReset, 0)
	ret.errorCountEOFReset, _ = node.Points.ValueBool("", data.PointTypeErrorCountEOFReset, 0)

	if ret.busType == data.PointValueServer {
		var ok bool
		ret.id, ok = node.Points.ValueInt("", data.PointTypeID, 0)
		if !ok {
			return nil, errors.New("Must define modbus ID for server bus")
		}
	}

	return &ret, nil
}

// NewModbus creates a new bus from a node
func NewModbus(db *genji.Db, nc *natsgo.Conn, node *data.NodeEdge) (*Modbus, error) {
	ret, err := nodeToModbus(node)
	if err != nil {
		return nil, err
	}

	ret.ios = make(map[string]*ModbusIO)
	ret.db = db
	ret.nc = nc
	ret.chStop = make(chan bool)
	ret.chIOs = make(chan map[string]*ModbusIO)

	return ret, nil
}

// Stop stops the bus and resets various fields
func (bus *Modbus) Stop() {
	bus.chStop <- true
	bus.running = false
}

// CheckPort verifies the serial port setup is correct for bus
func (bus *Modbus) CheckPort(node *data.NodeEdge) error {
	nodeBus, err := nodeToModbus(node)
	if err != nil {
		return err
	}

	if nodeBus.errorCountReset {
		bus.errorCount = 0
		p := data.Point{Type: data.PointTypeErrorCount, Value: 0}
		err := nats.SendPoint(bus.nc, bus.nodeID, &p, true)
		if err != nil {
			return err
		}

		p = data.Point{Type: data.PointTypeErrorCountReset, Value: 0}
		err = nats.SendPoint(bus.nc, bus.nodeID, &p, true)
		if err != nil {
			return err
		}
	}

	if nodeBus.errorCountCRCReset {
		bus.errorCountCRC = 0
		p := data.Point{Type: data.PointTypeErrorCountCRC, Value: 0}
		err := nats.SendPoint(bus.nc, bus.nodeID, &p, true)
		if err != nil {
			return err
		}

		p = data.Point{Type: data.PointTypeErrorCountCRCReset, Value: 0}
		err = nats.SendPoint(bus.nc, bus.nodeID, &p, true)
		if err != nil {
			return err
		}
	}

	if nodeBus.errorCountEOFReset {
		p := data.Point{Type: data.PointTypeErrorCountEOF, Value: 0}
		err := nats.SendPoint(bus.nc, bus.nodeID, &p, true)
		if err != nil {
			return err
		}

		p = data.Point{Type: data.PointTypeErrorCountEOFReset, Value: 0}
		err = nats.SendPoint(bus.nc, bus.nodeID, &p, true)
		if err != nil {
			return err
		}
		bus.errorCountEOF = 0
	}

	if nodeBus.busType != bus.busType ||
		nodeBus.portName != bus.portName ||
		nodeBus.baud != bus.baud ||
		nodeBus.id != bus.id ||
		nodeBus.debugLevel != bus.debugLevel ||
		nodeBus.pollPeriod != bus.pollPeriod {
		// need to re-init port if it is open
		if bus.port != nil {
			bus.port.Close()
			bus.port = nil
		}

		bus.busType = nodeBus.busType
		bus.portName = nodeBus.portName
		bus.baud = nodeBus.baud
		bus.id = nodeBus.id
		bus.debugLevel = nodeBus.debugLevel
		bus.pollPeriod = nodeBus.pollPeriod
	}

	if bus.port == nil {
		if bus.running {
			bus.Stop()
			// wait for bus to stop
			log.Println("Waiting for bus to stop")
		}

		log.Println("initializing modbus port: ", bus.portName)
		// need to init serial port
		mode := &serial.Mode{
			BaudRate: bus.baud,
		}

		port, err := serial.Open(bus.portName, mode)
		if err != nil {
			return err
		}

		bus.port = respreader.NewReadWriteCloser(port, time.Second*1, time.Millisecond*30)

		if bus.busType == data.PointValueServer {
			bus.client = nil
			bus.server = modbus.NewServer(byte(bus.id), bus.port)
			go bus.server.Listen(bus.debugLevel, func(err error) {
				log.Println("Modbus server error: ", err)
			}, func(changes []modbus.RegChange) {
				log.Println("Modbus reg change")
			})
		} else if bus.busType == data.PointValueClient {
			bus.server = nil
			bus.client = modbus.NewClient(bus.port, bus.debugLevel)
		}
	}

	return nil
}

// InitRegs is used in server mode to initilize the internal modbus regs when a IO changes
func (bus *Modbus) InitRegs(io *ModbusIO) {
	if bus.busType != data.PointValueServer {
		return
	}

	switch io.modbusIOType {
	case data.PointValueModbusDiscreteInput:
		bus.server.Regs.AddCoil(io.address)
	case data.PointValueModbusCoil:
		bus.server.Regs.AddCoil(io.address)
		bus.server.Regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusInputRegister:
		bus.server.Regs.AddReg(io.address, regCount(io.modbusDataType))
	case data.PointValueModbusHoldingRegister:
		bus.server.Regs.AddReg(io.address, regCount(io.modbusDataType))
		bus.WriteReg(io)
	}
}

// CheckIOs goes through ios on the bus and handles any config changes
func (bus *Modbus) CheckIOs() (bool, error) {
	ioNodes, err := bus.db.NodeChildren(bus.nodeID, data.NodeTypeModbusIO)
	if err != nil {
		return false, err
	}

	found := make(map[string]bool)

	iosChanged := false

	for _, ioNode := range ioNodes {
		found[ioNode.ID] = true
		io, ok := bus.ios[ioNode.ID]
		if !ok {
			// add ios
			var err error
			io, err = NewModbusIO(bus.busType, &ioNode)
			if err != nil {
				log.Println("Error creating new modbus: ", err)
				continue
			}
			bus.ios[ioNode.ID] = io
			bus.InitRegs(io)
			iosChanged = true
		} else {
			// check if anything has changed
			newIO, err := NewModbusIO(bus.busType, &ioNode)
			if err != nil {
				log.Println("Error with modbus IO: ", err)
				continue
			}
			changed := io.Changed(newIO)
			if changed {
				iosChanged = true
				bus.ios[ioNode.ID] = newIO
				bus.InitRegs(newIO)
			}
		}
	}

	// remove ios that have been deleted
	for id, io := range bus.ios {
		_, ok := found[id]
		if !ok {
			// bus was deleted so close and clear it
			log.Println("modbus io removed: ", io.description)
			// FIXME, do we need to do anything here
			delete(bus.ios, id)
			iosChanged = true
		}
	}

	return iosChanged, nil
}

// Run is routine that runs the logic for a bus. Intended to be run as
// a goroutine and all communication is with channels
func (bus *Modbus) Run() {
	timer := time.NewTicker(time.Millisecond * time.Duration(bus.pollPeriod))
	ios := make(map[string]*ModbusIO)
	for {
		select {
		case <-timer.C:
			for _, io := range ios {
				switch bus.busType {
				case data.PointValueClient:
					err := bus.ClientIO(io)
					if err != nil {
						log.Printf("Modbus client %v:%v, error: %v\n",
							bus.portName, io.description, err)
						err := bus.LogError(io, modbusErrorToPointType(err))
						if err != nil {
							log.Println("Error logging modbus error: ", err)
						}
					}
				case data.PointValueServer:
					err := bus.ServerIO(io)
					if err != nil {
						log.Printf("Modbus server %v:%v, error: %v\n",
							bus.portName, io.description, err)
						err := bus.LogError(io, modbusErrorToPointType(err))
						if err != nil {
							log.Println("Error logging modbus error: ", err)
						}
					}
				default:
					log.Println("Uknown modbus bus type: ",
						bus.busType)
				}
			}
		case <-bus.chStop:
			log.Println("Stopping client IO for: ", bus.portName)
			return
		case newIOs := <-bus.chIOs:
			ios = newIOs
		}
	}
}

// SendPoint sends a point over nats
func (bus *Modbus) SendPoint(nodeID, pointType string, value float64) error {
	// send the point
	p := data.Point{
		Type:  pointType,
		Value: value,
	}

	return nats.SendPoint(bus.nc, nodeID, &p, true)
}

// LogError logs any errors encountered
func (bus *Modbus) LogError(io *ModbusIO, typ string) error {
	busCount := 0
	ioCount := 0
	switch typ {
	case data.PointTypeErrorCount:
		busCount = bus.errorCount
		ioCount = io.errorCount
		bus.errorCount++
		io.errorCount++
	case data.PointTypeErrorCountEOF:
		busCount = bus.errorCountEOF
		ioCount = bus.errorCountEOF
		bus.errorCountEOF++
		io.errorCountEOF++
	case data.PointTypeErrorCountCRC:
		busCount = bus.errorCountCRC
		ioCount = bus.errorCountCRC
		bus.errorCountCRC++
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

	err := nats.SendPoint(bus.nc, bus.nodeID, &p, true)
	if err != nil {
		return err
	}

	p.Value = float64(ioCount)
	return nats.SendPoint(bus.nc, io.nodeID, &p, true)
}

// ReadReg reads an value from a reg (internal, not bus)
// This should only be used on server
func (bus *Modbus) ReadReg(io *ModbusIO) (float64, error) {
	var valueUnscaled float64
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		v, err := bus.server.Regs.ReadReg(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueUINT32:
		v, err := bus.server.Regs.ReadRegUint32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueINT32:
		v, err := bus.server.Regs.ReadRegInt32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueFLOAT32:
		v, err := bus.server.Regs.ReadRegFloat32(io.address)
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
func (bus *Modbus) WriteReg(io *ModbusIO) error {
	unscaledValue := (io.value - io.offset) / io.scale
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		bus.server.Regs.WriteReg(io.address, uint16(unscaledValue))
	case data.PointValueUINT32:
		bus.server.Regs.WriteRegUint32(io.address,
			uint32(unscaledValue))
	case data.PointValueINT32:
		bus.server.Regs.WriteRegInt32(io.address,
			int32(unscaledValue))
	case data.PointValueFLOAT32:
		bus.server.Regs.WriteRegFloat32(io.address,
			float32(unscaledValue))
	default:
		return fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)
	}
	return nil
}

// WriteBusHoldingReg used to write register values to bus
// should only be used by client
func (bus *Modbus) WriteBusHoldingReg(io *ModbusIO) error {
	unscaledValue := (io.valueSet - io.offset) / io.scale
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		err := bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address), uint16(unscaledValue))
		if err != nil {
			return err
		}
	case data.PointValueUINT32:
		regs := modbus.Uint32ToRegs([]uint32{uint32(unscaledValue)})
		err := bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address+1), regs[1])
		if err != nil {
			return err
		}

	case data.PointValueINT32:
		regs := modbus.Int32ToRegs([]int32{int32(unscaledValue)})
		err := bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address+1), regs[1])
		if err != nil {
			return err
		}

	case data.PointValueFLOAT32:
		regs := modbus.Float32ToRegs([]float32{float32(unscaledValue)})
		err := bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = bus.client.WriteSingleReg(byte(io.id),
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
func (bus *Modbus) ReadBusReg(io *ModbusIO) error {
	readFunc := bus.client.ReadHoldingRegs
	switch io.modbusIOType {
	case data.PointValueModbusHoldingRegister:
	case data.PointValueModbusInputRegister:
		readFunc = bus.client.ReadInputRegs
	default:
		return fmt.Errorf("ReadBusReg: unsupported modbus IO type: %v",
			io.modbusIOType)
	}
	var valueUnscaled float64
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		regs, err := readFunc(byte(io.id), uint16(io.address), 1)
		if err != nil {
			return err
		}
		if len(regs) < 1 {
			return errors.New("Did not receive enough data")
		}
		valueUnscaled = float64(regs[0])

	case data.PointValueUINT32:
		regs, err := readFunc(byte(io.id), uint16(io.address), 2)
		if err != nil {
			return err
		}
		if len(regs) < 2 {
			return errors.New("Did not receive enough data")
		}
		v := modbus.RegsToUint32(regs)

		valueUnscaled = float64(v[0])

	case data.PointValueINT32:
		regs, err := readFunc(byte(io.id), uint16(io.address), 2)
		if err != nil {
			return err
		}
		if len(regs) < 2 {
			return errors.New("Did not receive enough data")
		}
		v := modbus.RegsToInt32(regs)

		valueUnscaled = float64(v[0])

	case data.PointValueFLOAT32:
		regs, err := readFunc(byte(io.id), uint16(io.address), 2)
		if err != nil {
			return err
		}
		if len(regs) < 2 {
			return errors.New("Did not receive enough data")
		}
		valueUnscaled = float64(modbus.RegsToFloat32(regs)[0])

	default:
		return fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)
	}

	io.value = valueUnscaled*io.scale + io.offset
	// send the point
	err := bus.SendPoint(io.nodeID, data.PointTypeValue, io.value)
	if err != nil {
		return err
	}

	return nil
}

// ReadBusBit is used to read coil of discrete input values from bus
// this function modifies io.value. This should only be called from client.
func (bus *Modbus) ReadBusBit(io *ModbusIO) error {
	readFunc := bus.client.ReadCoils
	switch io.modbusIOType {
	case data.PointValueModbusCoil:
	case data.PointValueModbusDiscreteInput:
		readFunc = bus.client.ReadDiscreteInputs
	default:
		return fmt.Errorf("ReadBusBit: unhandled modbusIOType: %v",
			io.modbusIOType)
	}
	bits, err := readFunc(byte(io.id), uint16(io.address), 1)
	if err != nil {
		return err
	}
	if len(bits) < 1 {
		return errors.New("Did not receive enough data")
	}
	io.value = data.BoolToFloat(bits[0])

	err = bus.SendPoint(io.nodeID, data.PointTypeValue, io.value)
	if err != nil {
		return err
	}

	return nil
}

// ClientIO processes an IO on a client bus
func (bus *Modbus) ClientIO(io *ModbusIO) error {

	// read value from remote device and update regs
	switch io.modbusIOType {
	case data.PointValueModbusCoil:
		err := bus.ReadBusBit(io)
		if err != nil {
			return err
		}

		if io.valueSet != io.value {
			vBool := data.FloatToBool(io.valueSet)
			// we need set the remote value
			err := bus.client.WriteSingleCoil(byte(io.id), uint16(io.address),
				vBool)

			if err != nil {
				return err
			}

			err = bus.SendPoint(io.nodeID, data.PointTypeValue, io.valueSet)
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusDiscreteInput:
		err := bus.ReadBusBit(io)
		if err != nil {
			return err
		}

	case data.PointValueModbusHoldingRegister:
		err := bus.ReadBusReg(io)
		if err != nil {
			return err
		}

		if io.valueSet != io.value {
			// we need set the remote value
			err := bus.WriteBusHoldingReg(io)

			if err != nil {
				return err
			}

			err = bus.SendPoint(io.nodeID, data.PointTypeValue, io.valueSet)
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusInputRegister:
		err := bus.ReadBusReg(io)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unhandled modbus io type, io: %+v", io)
	}

	return nil
}

// ServerIO processes an IO on a server bus
func (bus *Modbus) ServerIO(io *ModbusIO) error {
	// update regs with db value
	switch io.modbusIOType {
	case data.PointValueModbusDiscreteInput:
		bus.server.Regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusCoil:
		regValue, err := bus.server.Regs.ReadCoil(io.address)
		if err != nil {
			return err
		}

		dbValue := data.FloatToBool(io.value)

		if regValue != dbValue {
			err = bus.SendPoint(io.nodeID, data.PointTypeValue, data.BoolToFloat(regValue))
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusInputRegister:
		bus.WriteReg(io)

	case data.PointValueModbusHoldingRegister:
		v, err := bus.ReadReg(io)
		if err != nil {
			return err
		}

		if io.value != v {
			err = bus.SendPoint(io.nodeID, data.PointTypeValue, v)
			if err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unhandled modbus io type: %v", io.modbusIOType)
	}

	return nil
}

// ModbusIO describes a modbus IO
type ModbusIO struct {
	nodeID             string
	description        string
	id                 int
	address            int
	modbusIOType       string
	modbusDataType     string
	scale              float64
	offset             float64
	value              float64
	valueSet           float64
	errorCount         int
	errorCountCRC      int
	errorCountEOF      int
	errorCountReset    bool
	errorCountCRCReset bool
	errorCountEOFReset bool
}

// NewModbusIO Convert node to modbus IO
func NewModbusIO(busType string, node *data.NodeEdge) (*ModbusIO, error) {
	ret := ModbusIO{
		nodeID: node.ID,
	}

	var ok bool

	ret.id, ok = node.Points.ValueInt("", data.PointTypeID, 0)
	if busType == data.PointValueClient && !ok {
		if busType == data.PointValueServer {
			return nil, errors.New("Must define modbus ID")
		}
	}

	ret.description, _ = node.Points.Text("", data.PointTypeDescription, 0)

	ret.address, ok = node.Points.ValueInt("", data.PointTypeAddress, 0)
	if !ok {
		return nil, errors.New("Must define modbus address")
	}
	ret.modbusIOType, ok = node.Points.Text("", data.PointTypeModbusIOType, 0)
	if !ok {
		return nil, errors.New("Must define modbus IO type")
	}

	if ret.modbusIOType == data.PointValueModbusInputRegister ||
		ret.modbusIOType == data.PointValueModbusHoldingRegister {
		ret.modbusDataType, ok = node.Points.Text("", data.PointTypeDataFormat, 0)
		if !ok {
			return nil, errors.New("Data format must be specified")
		}
		ret.scale, ok = node.Points.Value("", data.PointTypeScale, 0)
		if !ok {
			return nil, errors.New("Must define modbus scale")
		}
		ret.offset, ok = node.Points.Value("", data.PointTypeOffset, 0)
		if !ok {
			return nil, errors.New("Must define modbus offset")
		}
	}

	ret.value, _ = node.Points.Value("", data.PointTypeValue, 0)
	ret.valueSet, _ = node.Points.Value("", data.PointTypeValueSet, 0)
	ret.errorCount, _ = node.Points.ValueInt("", data.PointTypeErrorCount, 0)
	ret.errorCountCRC, _ = node.Points.ValueInt("", data.PointTypeErrorCountCRC, 0)
	ret.errorCountEOF, _ = node.Points.ValueInt("", data.PointTypeErrorCountEOF, 0)
	ret.errorCountReset, _ = node.Points.ValueBool("", data.PointTypeErrorCountReset, 0)
	ret.errorCountCRCReset, _ = node.Points.ValueBool("", data.PointTypeErrorCountCRCReset, 0)
	ret.errorCountEOFReset, _ = node.Points.ValueBool("", data.PointTypeErrorCountEOFReset, 0)

	return &ret, nil
}

// Changed returns true if the config of the IO has changed
func (io *ModbusIO) Changed(newIO *ModbusIO) bool {
	if io.id != newIO.id ||
		io.address != newIO.address ||
		io.modbusIOType != newIO.modbusIOType ||
		io.modbusDataType != newIO.modbusDataType ||
		io.scale != newIO.scale ||
		io.offset != newIO.offset ||
		io.value != newIO.value ||
		io.valueSet != newIO.valueSet ||
		io.errorCountReset != newIO.errorCountReset ||
		io.errorCountCRCReset != newIO.errorCountCRCReset ||
		io.errorCountEOFReset != newIO.errorCountEOFReset {
		return true
	}

	return false
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
