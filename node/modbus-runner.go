package node

import (
	"errors"
	"fmt"
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

// ModbusRunner is a type that actually does the modbus IO
type ModbusRunner struct {
	db     *genji.Db
	nc     *natsgo.Conn
	bus    *ModbusNode
	ios    map[string]*ModbusIO
	chDone chan bool
	client *modbus.Client
	server *modbus.Server
	chBus  <-chan *ModbusNode
	chIO   <-chan *ModbusIO
}

// NewModbusRunner creates a new modbus runner
func NewModbusRunner(db *genji.Db, nc *natsgo.Conn, bus *ModbusNode,
	ios map[string]*ModbusIO) *ModbusRunner {
	return &ModbusRunner{
		db:  db,
		nc:  nc,
		bus: bus,
		ios: ios,
	}
}

// Close stop the runner
func (r *ModbusRunner) Close() {
	// only try to close if running
	if r.chDone != nil {
		r.chDone <- true
	}
}

// SendPoint sends a point over nats
func (r *ModbusRunner) SendPoint(nodeID, pointType string, value float64) error {
	// send the point
	p := data.Point{
		Type:  pointType,
		Value: value,
	}

	return nats.SendPoint(r.nc, nodeID, &p, true)
}

// WriteBusHoldingReg used to write register values to bus
// should only be used by client
func (r *ModbusRunner) WriteBusHoldingReg(io *ModbusIO) error {
	unscaledValue := (io.valueSet - io.offset) / io.scale
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		err := r.client.WriteSingleReg(byte(io.id),
			uint16(io.address), uint16(unscaledValue))
		if err != nil {
			return err
		}
	case data.PointValueUINT32:
		regs := modbus.Uint32ToRegs([]uint32{uint32(unscaledValue)})
		err := r.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = r.client.WriteSingleReg(byte(io.id),
			uint16(io.address+1), regs[1])
		if err != nil {
			return err
		}

	case data.PointValueINT32:
		regs := modbus.Int32ToRegs([]int32{int32(unscaledValue)})
		err := r.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = r.client.WriteSingleReg(byte(io.id),
			uint16(io.address+1), regs[1])
		if err != nil {
			return err
		}

	case data.PointValueFLOAT32:
		regs := modbus.Float32ToRegs([]float32{float32(unscaledValue)})
		err := r.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = r.client.WriteSingleReg(byte(io.id),
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
func (r *ModbusRunner) ReadBusReg(io *ModbusIO) error {
	readFunc := r.client.ReadHoldingRegs
	switch io.modbusIOType {
	case data.PointValueModbusHoldingRegister:
	case data.PointValueModbusInputRegister:
		readFunc = r.client.ReadInputRegs
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
	err := r.SendPoint(io.nodeID, data.PointTypeValue, io.value)
	if err != nil {
		return err
	}

	return nil
}

// ReadBusBit is used to read coil of discrete input values from bus
// this function modifies io.value. This should only be called from client.
func (r *ModbusRunner) ReadBusBit(io *ModbusIO) error {
	readFunc := r.client.ReadCoils
	switch io.modbusIOType {
	case data.PointValueModbusCoil:
	case data.PointValueModbusDiscreteInput:
		readFunc = r.client.ReadDiscreteInputs
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

	err = r.SendPoint(io.nodeID, data.PointTypeValue, io.value)
	if err != nil {
		return err
	}

	return nil
}

// ClientIO processes an IO on a client bus
func (r *ModbusRunner) ClientIO(io *ModbusIO) error {

	// read value from remote device and update regs
	switch io.modbusIOType {
	case data.PointValueModbusCoil:
		err := r.ReadBusBit(io)
		if err != nil {
			return err
		}

		if io.valueSet != io.value {
			vBool := data.FloatToBool(io.valueSet)
			// we need set the remote value
			err := r.client.WriteSingleCoil(byte(io.id), uint16(io.address),
				vBool)

			if err != nil {
				return err
			}

			err = r.SendPoint(io.nodeID, data.PointTypeValue, io.valueSet)
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusDiscreteInput:
		err := r.ReadBusBit(io)
		if err != nil {
			return err
		}

	case data.PointValueModbusHoldingRegister:
		err := r.ReadBusReg(io)
		if err != nil {
			return err
		}

		if io.valueSet != io.value {
			// we need set the remote value
			err := r.WriteBusHoldingReg(io)

			if err != nil {
				return err
			}

			err = r.SendPoint(io.nodeID, data.PointTypeValue, io.valueSet)
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusInputRegister:
		err := r.ReadBusReg(io)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unhandled modbus io type, io: %+v", io)
	}

	return nil
}

// ServerIO processes an IO on a server bus
func (r *ModbusRunner) ServerIO(io *ModbusIO) error {
	// update regs with db value
	switch io.modbusIOType {
	case data.PointValueModbusDiscreteInput:
		r.server.Regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusCoil:
		regValue, err := r.server.Regs.ReadCoil(io.address)
		if err != nil {
			return err
		}

		dbValue := data.FloatToBool(io.value)

		if regValue != dbValue {
			err = r.SendPoint(io.nodeID, data.PointTypeValue, data.BoolToFloat(regValue))
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusInputRegister:
		r.WriteReg(io)

	case data.PointValueModbusHoldingRegister:
		v, err := r.ReadReg(io)
		if err != nil {
			return err
		}

		if io.value != v {
			err = r.SendPoint(io.nodeID, data.PointTypeValue, v)
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
func (r *ModbusRunner) InitRegs(io *ModbusIO) {
	if r.server == nil {
		return
	}
	switch io.modbusIOType {
	case data.PointValueModbusDiscreteInput:
		r.server.Regs.AddCoil(io.address)
	case data.PointValueModbusCoil:
		r.server.Regs.AddCoil(io.address)
		r.server.Regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusInputRegister:
		r.server.Regs.AddReg(io.address, regCount(io.modbusDataType))
	case data.PointValueModbusHoldingRegister:
		r.server.Regs.AddReg(io.address, regCount(io.modbusDataType))
		r.WriteReg(io)
	}
}

// ReadReg reads an value from a reg (internal, not bus)
// This should only be used on server
func (r *ModbusRunner) ReadReg(io *ModbusIO) (float64, error) {
	var valueUnscaled float64
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		v, err := r.server.Regs.ReadReg(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueUINT32:
		v, err := r.server.Regs.ReadRegUint32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueINT32:
		v, err := r.server.Regs.ReadRegInt32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueFLOAT32:
		v, err := r.server.Regs.ReadRegFloat32(io.address)
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
func (r *ModbusRunner) WriteReg(io *ModbusIO) error {
	unscaledValue := (io.value - io.offset) / io.scale
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		r.server.Regs.WriteReg(io.address, uint16(unscaledValue))
	case data.PointValueUINT32:
		r.server.Regs.WriteRegUint32(io.address,
			uint32(unscaledValue))
	case data.PointValueINT32:
		r.server.Regs.WriteRegInt32(io.address,
			int32(unscaledValue))
	case data.PointValueFLOAT32:
		r.server.Regs.WriteRegFloat32(io.address,
			float32(unscaledValue))
	default:
		return fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)
	}
	return nil
}

// LogError ...
func (r *ModbusRunner) LogError(io *ModbusIO, typ string) error {
	busCount := 0
	ioCount := 0
	switch typ {
	case data.PointTypeErrorCount:
		busCount = r.bus.errorCount
		ioCount = io.errorCount
		r.bus.errorCount++
		io.errorCount++
	case data.PointTypeErrorCountEOF:
		busCount = r.bus.errorCountEOF
		ioCount = io.errorCountEOF
		r.bus.errorCountEOF++
		io.errorCountEOF++
	case data.PointTypeErrorCountCRC:
		busCount = r.bus.errorCountCRC
		ioCount = io.errorCountCRC
		r.bus.errorCountCRC++
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

	err := nats.SendPoint(r.nc, r.bus.nodeID, &p, true)
	if err != nil {
		return err
	}

	p.Value = float64(ioCount)
	return nats.SendPoint(r.nc, io.nodeID, &p, true)
}

// Run is routine that runs the logic for a bus. Intended to be run as
// a goroutine and all communication is with channels.
// this routine may need to run fast scan times, so it should be doing
// slow things like reading the database.
func (r *ModbusRunner) Run() <-chan error {
	chError := make(chan error)

	go func() {
		// if we reset any error count, we set this to avoid continually resetting
		timer := time.NewTicker(time.Millisecond * time.Duration(r.bus.pollPeriod))
		log.Println("initializing modbus port: ", r.bus.portName)
		// need to init serial port
		mode := &serial.Mode{
			BaudRate: r.bus.baud,
		}

		serialPort, err := serial.Open(r.bus.portName, mode)
		if err != nil {
			chError <- fmt.Errorf("Error opening serial port: %w", err)
			close(chError)
			return
		}

		port := respreader.NewReadWriteCloser(serialPort, time.Millisecond*200, time.Millisecond*30)

		if r.bus.busType == data.PointValueServer {
			r.server = modbus.NewServer(byte(r.bus.id), port)
			go r.server.Listen(r.bus.debugLevel, func(err error) {
				log.Println("Modbus server error: ", err)
			}, func(changes []modbus.RegChange) {
				log.Println("Modbus reg change")
			})
		} else if r.bus.busType == data.PointValueClient {
			r.client = modbus.NewClient(port, r.bus.debugLevel)
		}

		// init all ios
		for _, io := range r.ios {
			r.InitRegs(io)
		}

		for {
			select {
			case <-timer.C:
				for _, io := range r.ios {
					switch r.bus.busType {
					case data.PointValueClient:
						err := r.ClientIO(io)
						if err != nil {
							log.Printf("Modbus client %v:%v, error: %v\n",
								r.bus.portName, io.description, err)
							err := r.LogError(io, modbusErrorToPointType(err))
							if err != nil {
								log.Println("Error logging modbus error: ", err)
							}
						}
					case data.PointValueServer:
						err := r.ServerIO(io)
						if err != nil {
							log.Printf("Modbus server %v:%v, error: %v\n",
								r.bus.portName, io.description, err)
							err := r.LogError(io, modbusErrorToPointType(err))
							if err != nil {
								log.Println("Error logging modbus error: ", err)
							}
						}
					default:
						log.Println("Uknown modbus bus type: ",
							r.bus.busType)
					}

				}
			case <-r.chDone:
				log.Println("Stopping client IO for: ", r.bus.portName)
				port.Close()
				if r.server != nil {
					r.server.Close()
				}
				return
			}
		}
	}()

	return chError
}
