package client

import (
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// UserCheck sends a nats message to check auth of user
// This function returns user nodes and a JWT node which includes a token
func UserCheck(nc *nats.Conn, email, pass string) ([]data.NodeEdge, error) {
	points := data.Points{
		{Type: data.PointTypeEmail, Text: email, Key: "0"},
		{Type: data.PointTypePass, Text: pass, Key: "0"},
	}

	pointsData, err := points.ToPb()
	if err != nil {
		return []data.NodeEdge{}, err
	}

	nodeMsg, err := nc.Request("auth.user", pointsData, time.Second*20)
	if err != nil {
		return []data.NodeEdge{}, err
	}

	nodes, err := data.PbDecodeNodesRequest(nodeMsg.Data)

	if err != nil {
		return []data.NodeEdge{}, err
	}

	return nodes, nil
}

// GetNatsURI returns the nats URI and auth token for the SIOT server
// this can be used to set up new NATS connections with different requirements
// (no echo, etc)
// return URI, authToken, error
func GetNatsURI(nc *nats.Conn) (string, string, error) {
	resp, err := nc.Request("auth.getNatsURI", nil, time.Second*1)

	if err != nil {
		return "", "", err
	}

	points, err := data.PbDecodePoints(resp.Data)
	if err != nil {
		return "", "", err
	}

	var uri, token string

	for _, p := range points {
		switch p.Type {
		case data.PointTypeURI:
			uri = p.Text
		case data.PointTypeToken:
			token = p.Text
		}
	}

	if uri == "" {
		return "", "", errors.New("URI not returned")
	}

	return uri, token, nil
}
