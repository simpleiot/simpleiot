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
	disable            bool
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

	ret.id, ok = node.Points.ValueInt(data.PointTypeID, "")
	if busType == data.PointValueClient && !ok {
		if busType == data.PointValueServer {
			return nil, errors.New("Must define modbus ID")
		}
	}

	ret.description, _ = node.Points.Text(data.PointTypeDescription, "")

	ret.address, ok = node.Points.ValueInt(data.PointTypeAddress, "")
	if !ok {
		return nil, errors.New("Must define modbus address")
	}

	ret.modbusIOType, ok = node.Points.Text(data.PointTypeModbusIOType, "")
	if !ok {
		return nil, errors.New("Must define modbus IO type")
	}

	ret.readOnly, _ = node.Points.ValueBool(data.PointTypeReadOnly, "")

	if ret.modbusIOType == data.PointValueModbusInputRegister ||
		ret.modbusIOType == data.PointValueModbusHoldingRegister {
		ret.modbusDataType, ok = node.Points.Text(data.PointTypeDataFormat, "")
		if !ok {
			return nil, errors.New("Data format must be specified")
		}
		ret.scale, ok = node.Points.Value(data.PointTypeScale, "")
		if !ok {
			return nil, errors.New("Must define modbus scale")
		}
		ret.offset, ok = node.Points.Value(data.PointTypeOffset, "")
		if !ok {
			return nil, errors.New("Must define modbus offset")
		}
	}

	ret.value, _ = node.Points.Value(data.PointTypeValue, "")
	ret.valueSet, _ = node.Points.Value(data.PointTypeValueSet, "")
	ret.disable, _ = node.Points.ValueBool(data.PointTypeDisable, "")
	ret.errorCount, _ = node.Points.ValueInt(data.PointTypeErrorCount, "")
	ret.errorCountCRC, _ = node.Points.ValueInt(data.PointTypeErrorCountCRC, "")
	ret.errorCountEOF, _ = node.Points.ValueInt(data.PointTypeErrorCountEOF, "")
	ret.errorCountReset, _ = node.Points.ValueBool(data.PointTypeErrorCountReset, "")
	ret.errorCountCRCReset, _ = node.Points.ValueBool(data.PointTypeErrorCountCRCReset, "")
	ret.errorCountEOFReset, _ = node.Points.ValueBool(data.PointTypeErrorCountEOFReset, "")

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
