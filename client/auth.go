package client

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// UserCheck sends a nats message to check auth of user
// This function returns user nodes and a JWT node which includes a token
func UserCheck(nc *nats.Conn, email, pass string) ([]data.NodeEdge, error) {
	points := data.Points{
		data.NewPointString(data.PointTypeEmail, "0", email),
		data.NewPointString(data.PointTypePass, "0", pass),
	}

	pointsData := points.Encode()

	nodeMsg, err := nc.Request("auth.user", pointsData, time.Second*20)
	if err != nil {
		return []data.NodeEdge{}, err
	}

	nodes, err := data.DecodeNodes(nodeMsg.Data)

	if err != nil {
		return []data.NodeEdge{}, err
	}

	return nodes, nil
}
