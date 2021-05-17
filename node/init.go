package node

import (
	"time"

	"github.com/google/uuid"
	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
)

// Init is used to create initial root node and admin
// user
func Init(nc *natsgo.Conn) error {
	rootID := uuid.New().String()

	pRoot := data.Point{
		Type: data.PointTypeNodeType,
		Text: data.NodeTypeDevice,
		Time: time.Now(),
	}

	err := nats.SendPoint(nc, rootID, pRoot, true)

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

	return nats.SendPoints(nc, admin.ID, admin.ToPoints(), true)
}
