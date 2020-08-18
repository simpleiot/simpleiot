package nats

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/nats-io/nats.go"
)

// EdgeOptions describes options for connecting edge devices
type EdgeOptions struct {
	Server       string
	AuthToken    string
	Disconnected func()
	Reconnected  func()
	Closed       func()
}

// EdgeConnect is a function that attempts connections for edge devices with appropriate
// timeouts, backups, etc. Currently set to disconnect if we don't have a connection after 6m,
// and then exp backup to try to connect every 6m after that.
func EdgeConnect(o EdgeOptions) (*nats.Conn, error) {
	authEnabled := "no"
	if o.AuthToken != "" {
		authEnabled = "yes"
	}
	log.Printf("NATS edge connect to: %v, auth enabled: %v", o.Server, authEnabled)
	nc, err := nats.Connect(o.Server,
		nats.Timeout(30*time.Second),
		nats.DrainTimeout(30*time.Second),
		nats.PingInterval(2*time.Minute),
		nats.MaxPingsOutstanding(3),
		nats.RetryOnFailedConnect(true),
		nats.ReconnectBufSize(128*1024),
		nats.ReconnectWait(10*time.Second),
		nats.MaxReconnects(-1),
		nats.SetCustomDialer(&net.Dialer{
			KeepAlive: -1,
		}),
		nats.CustomReconnectDelay(func(attempts int) time.Duration {
			delay := ExpBackoff(attempts, 6*time.Minute)
			log.Printf("NATS reconnect attempts: %v, delay: %v", attempts, delay)
			return delay
		}),
		nats.Token(o.AuthToken),
	)

	if err != nil {
		return nil, err
	}

	fmt.Println("NATS: TLS required: ", nc.TLSRequired())

	nc.SetErrorHandler(func(_ *nats.Conn, _ *nats.Subscription,
		err error) {
		log.Printf("NATS Error: %s\n", err)
	})

	nc.SetReconnectHandler(func(_ *nats.Conn) {
		o.Reconnected()
	})

	nc.SetDisconnectHandler(func(_ *nats.Conn) {
		o.Disconnected()
	})

	nc.SetClosedHandler(func(_ *nats.Conn) {
		o.Closed()
	})

	return nc, nil
}
