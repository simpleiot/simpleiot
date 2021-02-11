package node

import (
	"errors"

	"github.com/simpleiot/simpleiot/data"
)

// ModbusIONode describes a modbus IO db node
type ModbusIONode struct {
	nodeID             string
	description        string
	id                 int
	address            int
	modbusIOType       string
	modbusDataType     string
	readOnly           bool
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

// NewModbusIONode Convert node to modbus IO node
func NewModbusIONode(busType string, node *data.NodeEdge) (*ModbusIONode, error) {
	ret := ModbusIONode{
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

	ret.readOnly, _ = node.Points.ValueBool("", data.PointTypeReadOnly, 0)

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
// FIXME, we should not need this once we get NATS wired
func (io *ModbusIONode) Changed(newIO *ModbusIONode) bool {
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
