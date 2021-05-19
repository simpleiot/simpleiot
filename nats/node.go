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
		return data.Node{}, nil
	}

	node, err := data.PbDecodeNode(nodeMsg.Data)

	if err != nil {
		return data.Node{}, nil
	}

	return node, nil

}
