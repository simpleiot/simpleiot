package node

import (
	"errors"

	"github.com/simpleiot/simpleiot/data"
)

type oneWireIONode struct {
	nodeID          string
	description     string
	id              string
	units           string
	value           float64
	disabled        bool
	errorCount      int
	errorCountReset bool
}

func newOneWireIONode(node *data.NodeEdge) (*oneWireIONode, error) {
	ret := oneWireIONode{
		nodeID: node.ID,
	}

	var ok bool

	ret.id, ok = node.Points.Text(data.PointTypeID, "")
	if !ok {
		return nil, errors.New("must define onewire ID")
	}

	ret.description, _ = node.Points.Text(data.PointTypeDescription, "")
	ret.units, _ = node.Points.Text(data.PointTypeUnits, "")

	ret.value, _ = node.Points.Value(data.PointTypeValue, "")
	ret.disabled, _ = node.Points.ValueBool(data.PointTypeDisabled, "")
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
