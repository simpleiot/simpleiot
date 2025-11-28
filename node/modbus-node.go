package node

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/simpleiot/simpleiot/data"
)

// ModbusNode is the node data from the database
type ModbusNode struct {
	nodeID             string
	busType            string
	protocol           string
	uri                string
	id                 int // only used for server
	portName           string
	debugLevel         int
	baud               int
	pollPeriod         int
	timeout            int // response timeout in milliseconds
	disabled           bool
	errorCount         int
	errorCountCRC      int
	errorCountEOF      int
	errorCountReset    bool
	errorCountCRCReset bool
	errorCountEOFReset bool
}

// ModbusNodeResult contains the result of creating a ModbusNode and any corrections made
type ModbusNodeResult struct {
	Node             *ModbusNode
	TimeoutCorrected bool
}

// NewModbusNode converts a node to ModbusNode data structure
func NewModbusNode(node data.NodeEdge) (*ModbusNode, error) {
	result, err := NewModbusNodeWithCorrections(node)
	if err != nil {
		return nil, err
	}
	return result.Node, nil
}

// NewModbusNodeWithCorrections converts a node to ModbusNode data structure and reports corrections
func NewModbusNodeWithCorrections(node data.NodeEdge) (*ModbusNodeResult, error) {
	ret := ModbusNode{
		nodeID: node.ID,
	}

	var ok bool

	ret.busType, ok = node.Points.Text(data.PointTypeClientServer, "")
	if !ok {
		return nil, errors.New("must define modbus client/server")
	}

	ret.protocol, ok = node.Points.Text(data.PointTypeProtocol, "")
	if !ok {
		return nil, errors.New("must define modbus protocol")
	}

	if ret.protocol == data.PointValueRTU {
		ret.portName, ok = node.Points.Text(data.PointTypePort, "")
		if !ok {
			return nil, errors.New("must define modbus port name")
		}

		baud, ok := node.Points.Text(data.PointTypeBaud, "")
		if !ok {
			return nil, errors.New("must define modbus baud")
		}

		var err error
		ret.baud, err = strconv.Atoi(baud)

		if err != nil {
			return nil, fmt.Errorf("invalid baud: %v", baud)
		}
	}

	if ret.protocol == data.PointValueTCP {
		switch ret.busType {
		case data.PointValueClient:
			ret.uri, ok = node.Points.Text(data.PointTypeURI, "")
			if !ok {
				return nil, errors.New("must define modbus URI")
			}
		case data.PointValueServer:
			ret.portName, ok = node.Points.Text(data.PointTypePort, "")
			if !ok {
				return nil, errors.New("must define modbus port name")
			}
		default:
			return nil, fmt.Errorf("invalid bus type: %v", ret.busType)
		}
	}

	ret.pollPeriod, ok = node.Points.ValueInt(data.PointTypePollPeriod, "")
	if ret.busType == data.PointValueClient && !ok {
		return nil, errors.New("must define modbus polling period for client devices")
	}

	ret.debugLevel, _ = node.Points.ValueInt(data.PointTypeDebug, "")

	var timeoutCorrected bool
	ret.timeout, ok = node.Points.ValueInt(data.PointTypeTimeout, "")
	if !ok || ret.timeout <= 0 {
		ret.timeout = 100     // default timeout is 100ms
		timeoutCorrected = ok // only mark as corrected if timeout was explicitly set to invalid value
	}
	ret.disabled, _ = node.Points.ValueBool(data.PointTypeDisabled, "")
	ret.errorCount, _ = node.Points.ValueInt(data.PointTypeErrorCount, "")
	ret.errorCountCRC, _ = node.Points.ValueInt(data.PointTypeErrorCountCRC, "")
	ret.errorCountEOF, _ = node.Points.ValueInt(data.PointTypeErrorCountEOF, "")
	ret.errorCountReset, _ = node.Points.ValueBool(data.PointTypeErrorCountReset, "")
	ret.errorCountCRCReset, _ = node.Points.ValueBool(data.PointTypeErrorCountCRCReset, "")
	ret.errorCountEOFReset, _ = node.Points.ValueBool(data.PointTypeErrorCountEOFReset, "")

	if ret.busType == data.PointValueServer {
		var ok bool
		ret.id, ok = node.Points.ValueInt(data.PointTypeID, "")
		if !ok {
			return nil, errors.New("must define modbus ID for server bus")
		}
	}

	return &ModbusNodeResult{
		Node:             &ret,
		TimeoutCorrected: timeoutCorrected,
	}, nil
}
