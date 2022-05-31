package node

import (
	"errors"

	"github.com/simpleiot/simpleiot/data"
)

// UpstreamNode represents an upstream connection
type UpstreamNode struct {
	ID          string
	Description string
	URI         string
	AuthToken   string
	Disabled    bool
}

// NewUpstreamNode converts a node to UpstreamNode
func NewUpstreamNode(node data.NodeEdge) (*UpstreamNode, error) {
	var ok bool

	ret := &UpstreamNode{
		ID: node.ID,
	}

	ret.Description, _ = node.Points.Text(data.PointTypeDescription, "")
	ret.AuthToken, _ = node.Points.Text(data.PointTypeAuthToken, "")
	ret.Disabled, _ = node.Points.ValueBool(data.PointTypeDisable, "")

	ret.URI, ok = node.Points.Text(data.PointTypeURI, "")
	if !ok {
		return nil, errors.New("URI must be specified for upstream connection")
	}

	return ret, nil
}
