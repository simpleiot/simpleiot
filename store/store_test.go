package store_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/server"
)

func TestStoreUp(t *testing.T) {
	nc, root, stop, err := server.TestServer()
	_ = root

	if err != nil {
		t.Fatal("Error starting test server: ", err)
	}

	defer stop()

	chUpPoints := make(chan data.Points)

	sub, err := nc.Subscribe("up.none.>", func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			fmt.Println("Error decoding points")
			return
		}

		chUpPoints <- points
	})

	if err != nil {
		t.Fatal("sub error: ", err)
	}

	defer sub.Unsubscribe()

	err = client.SendNodePoint(nc, root.ID, data.Point{Type: data.PointTypeDescription,
		Text: "rootly"}, false)

	if err != nil {
		t.Fatal("Error sending point: ", err)
	}

stopFor:
	for {
		select {
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for description change")
		case p := <-chUpPoints:
			if p[0].Type != data.PointTypeDescription {
				continue
			}
			break stopFor // all is well
		}
	}
}
