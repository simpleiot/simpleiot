package client

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// DecodeNodePointsMsg decodes NATS message into node ID and points
func DecodeNodePointsMsg(msg *nats.Msg) (string, []data.Point, error) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 3 {
		return "", []data.Point{}, errors.New("Error decoding node points subject")
	}
	nodeID := chunks[1]
	points, err := data.PbDecodePoints(msg.Data)
	if err != nil {
		log.Println("Error decoding Pb points: ", err)
		return "", []data.Point{}, fmt.Errorf("Error decoding Pb points: %w", err)
	}

	return nodeID, points, nil
}

// DecodeEdgePointsMsg decodes NATS message into node ID and points
func DecodeEdgePointsMsg(msg *nats.Msg) (string, string, []data.Point, error) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 4 {
		return "", "", []data.Point{}, errors.New("Error decoding edge points subject")
	}
	nodeID := chunks[1]
	parentID := chunks[2]
	points, err := data.PbDecodePoints(msg.Data)
	if err != nil {
		log.Println("Error decoding Pb points: ", err)
		return "", "", []data.Point{}, fmt.Errorf("Error decoding Pb points: %w", err)
	}

	return nodeID, parentID, points, nil
}
