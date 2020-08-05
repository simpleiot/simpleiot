package api

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
)

// NatsHandler implements the SIOT NATS api
type NatsHandler struct {
	db        *db.Db
	authToken string
}

// NewNatsHandler creates a new NATS client for handling SIOT requests
func NewNatsHandler(db *db.Db, authToken string) *NatsHandler {
	return &NatsHandler{
		db:        db,
		authToken: authToken,
	}
}

// Listen for nats requests comming in and process them
// typically run as a goroutine
func (nh *NatsHandler) Listen(server string) {
	nc, err := nats.Connect(server,
		nats.Timeout(10*time.Second),
		nats.PingInterval(60*5*time.Second),
		nats.MaxPingsOutstanding(5),
		nats.ReconnectBufSize(5*1024*1024),
		nats.SetCustomDialer(&net.Dialer{
			KeepAlive: -1,
		}),
		nats.Token(nh.authToken),
	)

	if err != nil {
		log.Fatal("Error connecting to nats server: ", err)
	}

	for {
		wg := sync.WaitGroup{}
		wg.Add(1)

		if _, err := nc.Subscribe("device.*.samples",
			func(m *nats.Msg) {
				chunks := strings.Split(m.Subject, ".")
				if len(chunks) < 3 {
					log.Println("Error decoding device asmples subject: ", m.Subject)
					return
				}
				deviceID := chunks[1]
				samples, err := data.PbDecodeSamples(m.Data)
				if err != nil {
					log.Println("Error decoding Pb Samples: ", err)
					return
				}

				err = nh.db.DeviceActivity(deviceID)
				if err != nil {
					log.Println("Error updating device activity: ", err)
					return
				}
				for _, s := range samples {
					err = nh.db.DeviceSample(deviceID, s)
					if err != nil {
						log.Println("Error writting sample to Db: ", err)
						return
					}
				}
			}); err != nil {
			log.Println("Subscribe error: ", err)
			// rate limit re-subscribe a little
			time.Sleep(time.Second * 5)
			wg.Done()
		}

		wg.Wait()
	}
}

// NatsEdgeConnect is a function that attempts connections for edge devices with appropriate
// timeouts, backups, etc. Currently set to disconnect if we don't have a connection after 10m,
// and then exp backup to try to connect every 10m after that.
func NatsEdgeConnect(server, authToken string) (*nats.Conn, error) {
	nc, err := nats.Connect(server,
		nats.Timeout(30*time.Second),
		nats.DrainTimeout(30*time.Second),
		nats.PingInterval(2*time.Minute),
		nats.MaxPingsOutstanding(5),
		nats.RetryOnFailedConnect(true),
		nats.ReconnectBufSize(5*1024*1024),
		nats.MaxReconnects(-1),
		nats.SetCustomDialer(&net.Dialer{
			KeepAlive: -1,
		}),
		nats.CustomReconnectDelay(func(attempts int) time.Duration {
			delay := ExpBackoff(attempts, 10*time.Minute)
			log.Printf("NATS reconnect attempts: %v, delay: %v", attempts, delay)
			return delay
		}),
		nats.Token(authToken),
	)

	if err != nil {
		return nil, err
	}

	nc.SetErrorHandler(func(_ *nats.Conn, _ *nats.Subscription,
		err error) {
		log.Printf("NATS Error: %s\n", err)
	})

	nc.SetReconnectHandler(func(_ *nats.Conn) {
		log.Println("NATS Reconnected!")
	})

	nc.SetDisconnectHandler(func(_ *nats.Conn) {
		log.Println("NATS Disconnected!")
	})

	nc.SetClosedHandler(func(_ *nats.Conn) {
		log.Println("Connection to NATS is closed!")
	})

	return nc, nil
}
