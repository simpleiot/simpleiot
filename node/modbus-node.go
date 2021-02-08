package node

import (
	"errors"

	"github.com/simpleiot/simpleiot/data"
)

// ModbusNode is the node data from the database
type ModbusNode struct {
	nodeID             string
	busType            string
	id                 int // only used for server
	portName           string
	debugLevel         int
	baud               int
	pollPeriod         int
	errorCount         int
	errorCountCRC      int
	errorCountEOF      int
	errorCountReset    bool
	errorCountCRCReset bool
	errorCountEOFReset bool
}

// NewModbusNode converts a node to ModbusNode data structure
func NewModbusNode(node data.NodeEdge) (*ModbusNode, error) {
	ret := ModbusNode{
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
