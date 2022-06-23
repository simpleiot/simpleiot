package node

import (
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// Init is used to create initial root node and admin
// user
func Init(nc *nats.Conn) error {
	rootID := uuid.New().String()

	pRoot := data.Point{
		Type: data.PointTypeNodeType,
		Text: data.NodeTypeDevice,
		Time: time.Now(),
	}

	err := client.SendNodePoint(nc, rootID, pRoot, true)

	if err != nil {
		return err
	}

	admin := data.User{
		ID:        uuid.New().String(),
		FirstName: "admin",
		LastName:  "user",
		Email:     "admin@admin.com",
		Pass:      "admin",
	}

	return client.SendNodePoints(nc, admin.ID, admin.ToPoints(), true)
}
