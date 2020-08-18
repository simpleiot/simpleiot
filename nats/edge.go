package nats

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

// NatsEdgeConnect is a function that attempts connections for edge devices with appropriate
// timeouts, backups, etc. Currently set to disconnect if we don't have a connection after 10m,
// and then exp backup to try to connect every 10m after that.
func NatsEdgeConnect(server, authToken string) (*nats.Conn, error) {
	nc, err := nats.Connect(server,
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
		nats.Secure(&tls.Config{MaxVersion: tls.VersionTLS12}),
		nats.CustomReconnectDelay(func(attempts int) time.Duration {
			delay := ExpBackoff(attempts, 5*time.Minute)
			log.Printf("NATS reconnect attempts: %v, delay: %v", attempts, delay)
			return delay
		}),
		nats.Token(authToken),
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
		log.Println("NATS Reconnected!")
	})

	nc.SetDisconnectHandler(func(_ *nats.Conn) {
		log.Println("NATS Disconnected!")
	})

	nc.SetClosedHandler(func(_ *nats.Conn) {
		log.Println("Connection to NATS is closed! -- this should never happen, waiting 15m then exitting")
		time.Sleep(15 * time.Minute)
		os.Exit(-1)
	})

	return nc, nil
}
