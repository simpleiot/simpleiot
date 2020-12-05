package node

import (
	"errors"
	"log"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
	"github.com/simpleiot/simpleiot/modbus"
	"github.com/simpleiot/simpleiot/respreader"
	"go.bug.st/serial"
)

// ModbusIO describes a modbus IO
type ModbusIO struct {
	description    string
	id             int
	address        int
	modbusType     string
	modbusDataType string
	scale          float64
	offset         float64
	value          float64
	valueRaw       float64
}

// NewModbusIO Convert node to modbus IO
func NewModbusIO(busType string, node *data.NodeEdge) (*ModbusIO, error) {
	var ret ModbusIO
	var ok bool
	ret.id, ok = node.Points.ValueInt("", data.PointTypeID, 0)
	if busType == data.PointValueClient && !ok {
		return nil, errors.New("Must define modbus ID")
	}

	ret.address, ok = node.Points.ValueInt("", data.PointTypeAddress, 0)
	if !ok {
		return nil, errors.New("Must define modbus address")
	}
	ret.modbusType, ok = node.Points.Text("", data.PointTypeModbusIOType, 0)
	if !ok {
		return nil, errors.New("Must define modbus IO type")
	}

	if ret.modbusType == data.PointValueModbusRegister {
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

	return &ret, nil
}

// Modbus describes a modbus bus
type Modbus struct {
	busType    string
	id         int // only used for server
	ios        map[string]*ModbusIO
	portName   string
	baud       int
	port       *respreader.ReadWriteCloser
	client     *modbus.Client
	server     *modbus.Server
	debugLevel int
}

// NewModbus creates a new bus from a node
func NewModbus(node *data.NodeEdge) (*Modbus, error) {
	busType, ok := node.Points.Text("", data.PointTypeClientServer, 0)
	if !ok {
		return nil, errors.New("Must define modbus client/server")
	}
	portName, ok := node.Points.Text("", data.PointTypePort, 0)
	if !ok {
		return nil, errors.New("Must define modbus port name")
	}
	baud, ok := node.Points.Value("", data.PointTypeBaud, 0)
	if !ok {
		return nil, errors.New("Must define modbus baud")
	}

	return &Modbus{
		busType:  busType,
		portName: portName,
		baud:     int(baud),
		ios:      make(map[string]*ModbusIO),
	}, nil
}

// CheckPort verifies the serial port setup is correct for bus
func (m *Modbus) CheckPort(node *data.NodeEdge) error {
	busType, ok := node.Points.Text("", data.PointTypeClientServer, 0)
	if !ok {
		return errors.New("Must define modbus client/server")
	}
	portName, ok := node.Points.Text("", data.PointTypePort, 0)
	if !ok {
		return errors.New("Must define modbus port name")
	}
	baud, ok := node.Points.Value("", data.PointTypeBaud, 0)
	if !ok {
		return errors.New("Must define modbus baud")
	}

	debugLevel, _ := node.Points.Value("", data.PointTypeDebug, 0)

	id := m.id

	if busType == data.PointValueServer {
		idF, ok := node.Points.Value("", data.PointTypeID, 0)
		if !ok {
			return errors.New("Must define modbus ID for server bus")
		}

		id = int(idF)
	}

	if busType != m.busType || portName != m.portName ||
		int(baud) != m.baud || id != m.id ||
		int(debugLevel) != m.debugLevel {
		// need to re-init port if it is open
		if m.port != nil {
			m.port.Close()
			m.port = nil
		}

		m.busType = busType
		m.portName = portName
		m.baud = int(baud)
		m.id = id
		m.debugLevel = int(debugLevel)
	}

	if m.port == nil {
		log.Println("initializing modbus port: ", m.portName)
		// need to init serial port
		mode := &serial.Mode{
			BaudRate: m.baud,
		}

		port, err := serial.Open(m.portName, mode)
		if err != nil {
			return err
		}

		m.port = respreader.NewReadWriteCloser(port, time.Second*1, time.Millisecond*30)

		if m.busType == data.PointValueServer {
			m.client = nil
			m.server = modbus.NewServer(byte(m.id), m.port)
			go m.server.Listen(m.debugLevel, func(err error) {
				log.Println("Modbus server error: ", err)
			}, func(changes []modbus.RegChange) {
				log.Println("Modbus reg change")
			})
		} else if m.busType == data.PointValueClient {
			m.server = nil
			m.client = modbus.NewClient(m.port, m.debugLevel)
		}
	}

	return nil
}

// ModbusManager manages state of modbus
type ModbusManager struct {
	db     *genji.Db
	busses map[string]*Modbus
}

// NewModbusManager creates a new modbus manager
func NewModbusManager(db *genji.Db) *ModbusManager {
	return &ModbusManager{db: db, busses: make(map[string]*Modbus)}
}

// Update queries DB for modbus nodes and synchronizes
// with internal structures and updates data
func (mm *ModbusManager) Update() error {
	rootID := mm.db.RootNodeID()
	nodes, err := mm.db.NodeChildren(rootID, data.NodeTypeModbus)
	if err != nil {
		return err
	}

	// FIXME remove busses that no longer exist

	for _, ne := range nodes {
		bus, ok := mm.busses[ne.ID]
		if !ok {
			var err error
			bus, err = NewModbus(&ne)
			if err != nil {
				log.Println("Error creating new modbus: ", err)
				continue
			}
			mm.busses[ne.ID] = bus
		}

		err := bus.CheckPort(&ne)
		if err != nil {
			log.Println("Error initializing modbus port: ",
				ne.ID, err)
			continue
		}

		ioNodes, err := mm.db.NodeChildren(ne.ID, data.NodeTypeModbusIO)
		if err != nil {
			log.Println("Error getting modbus IO nodes: ", err)
			continue
		}

		for _, ioNode := range ioNodes {
			io, err := NewModbusIO(bus.busType, &ioNode)
			if err != nil {
				log.Println("Error creating new modbus IO: ", err)
				continue
			}

			if bus.busType == data.PointValueServer {
				// update regs with db value
				switch io.modbusType {
				case data.PointValueModbusRegister:
					switch io.modbusDataType {
					case data.PointValueUINT16:
						bus.server.Regs.AddReg(uint16(io.address))
						bus.server.Regs.WriteReg(
							uint16(io.address),
							uint16(io.value))
					default:
						log.Println("unhandled data type: ",
							io.modbusDataType)
					}
				default:
					log.Println("unhandled modbus io type: ", io.modbusType)
				}
			} else {
				log.Println("unhandled modbus type: ", bus.busType)
			}
		}
	}

	return nil
}
