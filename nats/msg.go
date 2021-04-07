package nats

import (
	"errors"
	"fmt"
	"log"
	"strings"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// DecodeNodePointsMsg decodes NATS message into node ID and points
func DecodeNodePointsMsg(msg *natsgo.Msg) (string, []data.Point, error) {
	chunks := strings.Split(msg.Subject, ".")
	if len(chunks) < 3 {
		return "", []data.Point{}, errors.New("Error decoding node samples subject")
	}
	nodeID := chunks[1]
	points, err := data.PbDecodePoints(msg.Data)
	if err != nil {
		log.Println("Error decoding Pb points: ", err)
		return "", []data.Point{}, fmt.Errorf("Error decoding Pb points: %w", err)
	}

	return nodeID, points, nil
}
