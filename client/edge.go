package client

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/nats-io/nats.go"
)

// EdgeOptions describes options for connecting edge devices
type EdgeOptions struct {
	URI          string
	AuthToken    string
	NoEcho       bool
	Connected    func()
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

	natsErrHandler := func(nc *nats.Conn, sub *nats.Subscription, natsErr error) {
		fmt.Printf("error: %v\n", natsErr)
		switch natsErr {
		case nats.ErrSlowConsumer:
			pendingMsgs, _, err := sub.Pending()
			if err != nil {
				fmt.Printf("couldn't get pending messages: %v", err)
				return
			}
			fmt.Printf("Falling behind with %d pending messages on subject %q.\n",
				pendingMsgs, sub.Subject)
			// Log error, notify operations...
		default:
			log.Println("Nats client error: ", natsErr)
		}
		// check for other errors
	}

	siotOptions := func(o *nats.Options) error {
		_ = nats.Timeout(30 * time.Second)(o)
		_ = nats.DrainTimeout(30 * time.Second)(o)
		_ = nats.PingInterval(2 * time.Minute)(o)
		_ = nats.MaxPingsOutstanding(3)(o)
		_ = nats.RetryOnFailedConnect(true)(o)
		_ = nats.ReconnectBufSize(128 * 1024)(o)
		_ = nats.ReconnectWait(10 * time.Second)(o)
		_ = nats.MaxReconnects(-1)(o)
		_ = nats.SetCustomDialer(&net.Dialer{
			KeepAlive: -1,
		})(o)

		_ = nats.CustomReconnectDelay(func(attempts int) time.Duration {
			delay := ExpBackoff(attempts, 6*time.Minute)
			log.Printf("NATS reconnect attempts: %v, delay: %v", attempts, delay)
			return delay
		})(o)

		_ = nats.Token(eo.AuthToken)(o)

		if eo.NoEcho {
			o.NoEcho = true
		}

		_ = nats.ErrorHandler(natsErrHandler)(o)

		_ = nats.ConnectHandler(func(_ *nats.Conn) {
			if eo.Connected != nil {
				eo.Connected()
			}
		})(o)

		_ = nats.ErrorHandler(func(_ *nats.Conn, sub *nats.Subscription,
			err error) {
			log.Printf("NATS Error, sub: %v, err: %s\n", sub.Subject, err)
		})(o)

		_ = nats.ReconnectHandler(func(_ *nats.Conn) {
			if eo.Reconnected != nil {
				eo.Reconnected()
			}
		})(o)

		_ = nats.DisconnectHandler(func(_ *nats.Conn) {
			if eo.Disconnected != nil {
				eo.Disconnected()
			}
		})(o)

		_ = nats.ClosedHandler(func(_ *nats.Conn) {
			if eo.Closed != nil {
				eo.Closed()
			}
		})(o)

		return nil
	}

	uri, err := sanitizeURI(eo.URI)
	if err != nil {
		log.Printf("Error sanitizing URI %v: %v", eo.URI, err)
		return nil, err
	}

	log.Printf("NATS edge connect to: %v, auth enabled: %v", uri, authEnabled)
	nc, err := nats.Connect(uri, siotOptions)

	if err != nil {
		return nil, err
	}

	fmt.Println("NATS: TLS required: ", nc.TLSRequired())

	return nc, nil
}
