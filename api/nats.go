package api

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// NatsHandler implements the SIOT NATS api
type NatsHandler struct{}

// NewNatsHandler creates a new NATS client for handling SIOT requests
func NewNatsHandler() *NatsHandler {
	return &NatsHandler{}
}

// Listen for nats requests comming in and process them
// typically run as a goroutine
func (nh *NatsHandler) Listen(server string) {
	nc, err := nats.Connect(server,
		nats.Timeout(10*time.Second),
		nats.PingInterval(60*2*time.Second),
		nats.MaxPingsOutstanding(5),
		nats.ReconnectBufSize(5*1024*1024),
		//nats.Token(authToken),
	)
	if err != nil {
		log.Fatal("Error connecting to nats server: ", err)
	}

	sub, err := nc.SubscribeSync("dev.s.*")
	if err != nil {
		log.Fatal(err)
	}

	for {
		// TODO this is crashing if we get a timeout
		// Wait for a message
		msg, err := sub.NextMsg(10 * time.Minute)
		if err != nil {
			log.Println("error getting NATS message:", err)
		}

		// Use the response
		log.Printf("msg: %s", msg.Data)
	}
}
