package client

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

// EdgeOptions describes options for connecting edge devices
type EdgeOptions struct {
	URI       string
	AuthToken string
	// NkeyPub and NkeySign are a device credential: the public key the
	// upstream knows and a function that signs the server's nonce with
	// it (see DeviceSigner). When set, AuthToken is not sent.
	NkeyPub  string
	NkeySign nats.SignatureHandler
	// CACert is a PEM certificate chain the upstream has to present,
	// for an instance that pins its upstream rather than trusting the
	// system store. Empty keeps the system store.
	CACert       string
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
		authEnabled = "token"
	}

	if eo.NkeyPub != "" {
		if eo.NkeySign == nil {
			return nil, errors.New("device credential needs a signer")
		}
		if !nkeys.IsValidPublicUserKey(eo.NkeyPub) {
			return nil, errors.New("device credential is not a user key")
		}
		authEnabled = "device credential " + eo.NkeyPub
	}

	var tlsConfig *tls.Config
	if eo.CACert != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(eo.CACert)) {
			return nil, errors.New("caCert holds no certificate")
		}
		tlsConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	}

	natsErrHandler := func(_ *nats.Conn, sub *nats.Subscription, natsErr error) {
		log.Printf("error: %v\n", natsErr)
		switch natsErr {
		case nats.ErrSlowConsumer:
			pendingMsgs, _, err := sub.Pending()
			if err != nil {
				log.Printf("couldn't get pending messages: %v", err)
				return
			}
			log.Printf("Falling behind with %d pending messages on subject %q.\n",
				pendingMsgs, sub.Subject)
			// Log error, notify operations...
		default:
			log.Println("Nats client error:", natsErr)
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

		if eo.NkeyPub != "" {
			_ = nats.Nkey(eo.NkeyPub, eo.NkeySign)(o)
			// the upstream grants this key its own inbox and nothing
			// else, so replies have to arrive there
			_ = nats.CustomInboxPrefix(InboxPrefix(eo.NkeyPub))(o)
		} else {
			_ = nats.Token(eo.AuthToken)(o)
		}

		if tlsConfig != nil {
			_ = nats.Secure(tlsConfig)(o)
		}

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
			if sub != nil {
				log.Printf("NATS Error, sub: %v, err: %v\n", sub.Subject, err)
			} else {
				log.Printf("NATS Error, err: %v\n", err)
			}
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

	log.Println("NATS: TLS required:", nc.TLSRequired())

	return nc, nil
}
