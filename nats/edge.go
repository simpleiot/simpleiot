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
	NoEcho       bool
	Disconnected func()
	Reconnected  func()
	Closed       func()
}

// EdgeConnect is a function that attempts connections for edge devices with appropriate
// timeouts, backups, etc. Currently set to disconnect if we don't have a connection after 6m,
// and then exp backup to try to connect every 6m after that.
func EdgeConnect(eo EdgeOptions) (*nats.Conn, error) {
	authEnabled := "no"
	if eo.AuthToken != "" {
		authEnabled = "yes"
	}

	siotOptions := func(o *nats.Options) error {
		nats.Timeout(30 * time.Second)(o)
		nats.DrainTimeout(30 * time.Second)(o)
		nats.PingInterval(2 * time.Minute)(o)
		nats.MaxPingsOutstanding(3)(o)
		nats.RetryOnFailedConnect(true)(o)
		nats.ReconnectBufSize(128 * 1024)(o)
		nats.ReconnectWait(10 * time.Second)(o)
		nats.MaxReconnects(-1)(o)
		nats.SetCustomDialer(&net.Dialer{
			KeepAlive: -1,
		})(o)
		nats.CustomReconnectDelay(func(attempts int) time.Duration {
			delay := ExpBackoff(attempts, 6*time.Minute)
			log.Printf("NATS reconnect attempts: %v, delay: %v", attempts, delay)
			return delay
		})(o)
		nats.Token(eo.AuthToken)(o)

		if eo.NoEcho {
			o.NoEcho = true
		}

		return nil
	}

	log.Printf("NATS edge connect to: %v, auth enabled: %v", eo.Server, authEnabled)
	nc, err := nats.Connect(eo.Server, siotOptions)

	if err != nil {
		return nil, err
	}

	fmt.Println("NATS: TLS required: ", nc.TLSRequired())

	nc.SetErrorHandler(func(_ *nats.Conn, _ *nats.Subscription,
		err error) {
		log.Printf("NATS Error: %s\n", err)
	})

	nc.SetReconnectHandler(func(_ *nats.Conn) {
		eo.Reconnected()
	})

	nc.SetDisconnectHandler(func(_ *nats.Conn) {
		eo.Disconnected()
	})

	nc.SetClosedHandler(func(_ *nats.Conn) {
		eo.Closed()
	})

	return nc, nil
}
