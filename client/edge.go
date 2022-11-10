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

		nats.ErrorHandler(natsErrHandler)(o)

		return nil
	}

	uri, err := sanitizeURI(eo.URI)
	if err != nil {
		log.Printf("Error sanitizing URI %v: %v", eo.URI, err)
	}

	log.Printf("NATS edge connect to: %v, auth enabled: %v", uri, authEnabled)
	nc, err := nats.Connect(uri, siotOptions)

	if err != nil {
		return nil, err
	}

	fmt.Println("NATS: TLS required: ", nc.TLSRequired())

	go func() {
		for {
			status := nc.Status()
			switch status {
			case nats.CONNECTED:
				eo.Connected()
				// we only get one connected, the rest are
				// reconnected
				return
			case nats.CLOSED:
				// return as the client was closed
				return
			}
			time.Sleep(5 * time.Second)
		}
	}()

	nc.SetErrorHandler(func(_ *nats.Conn, sub *nats.Subscription,
		err error) {
		log.Printf("NATS Error, sub: %v, err: %s\n", sub.Subject, err)
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
