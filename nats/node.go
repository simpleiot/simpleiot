package nats

import (
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// GetNode over NATS
func GetNode(nc *natsgo.Conn, id string) (data.Node, error) {
	nodeMsg, err := nc.Request("node."+id, nil, time.Second*20)
	if err != nil {
		return data.Node{}, err
	}

	node, err := data.PbDecodeNode(nodeMsg.Data)

	if err != nil {
		return data.Node{}, err
	}

	return node, nil
}

// GetNodeChildren over NATS (immediate children only, not recursive)
func GetNodeChildren(nc *natsgo.Conn, id string) ([]data.Node, error) {
	nodeMsg, err := nc.Request("node."+id+".children", nil, time.Second*20)
	if err != nil {
		return nil, err
	}

	nodes, err := data.PbDecodeNodes(nodeMsg.Data)

	if err != nil {
		return nil, err
	}

	return nodes, nil
}
