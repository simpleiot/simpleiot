package client

import (
	"fmt"
	"log"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// DecodeNodePointsMsg decodes NATS message into node ID and points
func DecodeNodePointsMsg(msg *nats.Msg) (string, []data.Point, error) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 2 {
		return "", []data.Point{}, fmt.Errorf("Invalid NodePoints subject: %v", msg.Subject)
	}
	nodeID := chunks[1]
	points, err := data.PbDecodePoints(msg.Data)
	if err != nil {
		log.Println("Error decoding Pb points:", err)
		return "", []data.Point{}, fmt.Errorf("Error decoding Pb points: %w", err)
	}

	return nodeID, points, nil
}

// DecodeEdgePointsMsg decodes NATS message into node ID and points
// returns nodeID, parentID, points, error
func DecodeEdgePointsMsg(msg *nats.Msg) (string, string, []data.Point, error) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 3 {
		return "", "", []data.Point{}, fmt.Errorf("Invalid EdgePoints subject: %v", msg.Subject)
	}
	nodeID := chunks[1]
	parentID := chunks[2]
	points, err := data.PbDecodePoints(msg.Data)
	if err != nil {
		log.Println("Error decoding Pb points:", err)
		return "", "", []data.Point{}, fmt.Errorf("Error decoding Pb points: %w", err)
	}

	return nodeID, parentID, points, nil
}

// DecodeUpNodePointsMsg decodes NATS message into node ID and points
// returns upID, nodeID, points, error
func DecodeUpNodePointsMsg(msg *nats.Msg) (string, string, []data.Point, error) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 3 {
		return "", "", []data.Point{}, fmt.Errorf("Invalid UpNode subject: %v", msg.Subject)
	}
	upID := chunks[1]
	nodeID := chunks[2]
	points, err := data.PbDecodePoints(msg.Data)
	if err != nil {
		log.Println("Error decoding Pb points:", err)
		return "", "", []data.Point{}, fmt.Errorf("Error decoding Pb points: %w", err)
	}

	return upID, nodeID, points, nil
}

// DecodeUpEdgePointsMsg decodes NATS message into node ID and points
// returns upID, nodeID, parentID, points, error
func DecodeUpEdgePointsMsg(msg *nats.Msg) (string, string, string, []data.Point, error) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 4 {
		return "", "", "", []data.Point{}, fmt.Errorf("Invalid UpEdge subject: %v", msg.Subject)
	}
	upID := chunks[1]
	nodeID := chunks[2]
	parentID := chunks[3]
	points, err := data.PbDecodePoints(msg.Data)
	if err != nil {
		log.Println("Error decoding Pb points:", err)
		return "", "", "", []data.Point{}, fmt.Errorf("Error decoding Pb points: %w", err)
	}

	return upID, nodeID, parentID, points, nil
}
