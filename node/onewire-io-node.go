package node

import (
	"errors"

	"github.com/simpleiot/simpleiot/data"
)

type oneWireIONode struct {
	nodeID          string
	description     string
	id              int
	value           float64
	disable         bool
	errorCount      int
	errorCountReset bool
}

func newOneWireIONode(node *data.NodeEdge) (*oneWireIONode, error) {
	ret := oneWireIONode{
		nodeID: node.ID,
	}

	var ok bool

	ret.id, ok = node.Points.ValueInt(data.PointTypeID, "")
	if !ok {
		return nil, errors.New("Must define onewire ID")
	}

	ret.description, _ = node.Points.Text(data.PointTypeDescription, "")

	ret.value, _ = node.Points.Value(data.PointTypeValue, "")
	ret.disable, _ = node.Points.ValueBool(data.PointTypeDisable, "")
	ret.errorCount, _ = node.Points.ValueInt(data.PointTypeErrorCount, "")
	ret.errorCountReset, _ = node.Points.ValueBool(data.PointTypeErrorCountReset, "")

	return &ret, nil
}

// Changed returns true if the config of the IO has changed
// FIXME, we should not need this once we get NATS wired
func (io *oneWireIONode) Changed(newIO *oneWireIONode) bool {
	if io.id != newIO.id ||
		io.value != newIO.value ||
		io.errorCountReset != newIO.errorCountReset {
		return true
	}

	return false
}
