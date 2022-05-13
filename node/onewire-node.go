package node

import (
	"github.com/simpleiot/simpleiot/data"
)

type oneWireNode struct {
	nodeID          string
	description     string
	index           int
	debugLevel      int
	pollPeriod      int
	disable         bool
	errorCount      int
	errorCountReset bool
}

func newOneWireNode(node data.NodeEdge) (*oneWireNode, error) {
	ret := oneWireNode{
		nodeID: node.ID,
	}

	ret.description, _ = node.Points.Text(data.PointTypeDescription, "")
	ret.index, _ = node.Points.ValueInt(data.PointTypeIndex, "")
	ret.debugLevel, _ = node.Points.ValueInt(data.PointTypeDebug, "")
	ret.disable, _ = node.Points.ValueBool(data.PointTypeDisable, "")
	ret.pollPeriod, _ = node.Points.ValueInt(data.PointTypePollPeriod, "")
	ret.errorCount, _ = node.Points.ValueInt(data.PointTypeErrorCount, "")
	ret.errorCountReset, _ = node.Points.ValueBool(data.PointTypeErrorCountReset, "")

	return &ret, nil
}
