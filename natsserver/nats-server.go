package natsserver

import (
	"log"
	"os"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

// Options for starting the nat server
type Options struct {
	Port       int
	HTTPPort   int
	Auth       string
	TLSCert    string
	TLSKey     string
	TLSTimeout float64
}

// StartNatsServer starts a nats server instance. This function will block
// so should be started with a go routine
func StartNatsServer(o Options) {
	opts := server.Options{
		Port:          o.Port,
		HTTPPort:      o.HTTPPort,
		Authorization: o.Auth,
	}

	if o.TLSCert != "" && o.TLSKey != "" {
		log.Println("Setting up NATS TLS ...")
		opts.TLS = true
		opts.TLSCert = o.TLSCert
		opts.TLSKey = o.TLSKey
		opts.TLSTimeout = o.TLSTimeout
		tc := server.TLSConfigOpts{}
		tc.CertFile = opts.TLSCert
		tc.KeyFile = opts.TLSKey
		tc.CaFile = opts.TLSCaCert
		tc.Verify = opts.TLSVerify

		var err error
		opts.TLSConfig, err = server.GenTLSConfig(&tc)

		if err != nil {
			log.Fatal("Error setting up TLS: ", err)
		}
	}

	opts.Websocket.Port = 9090
	opts.Websocket.Token = o.Auth
	opts.Websocket.AuthTimeout = o.TLSTimeout
	opts.Websocket.NoTLS = true // will likely be fronted by Caddy anyway
	opts.Websocket.HandshakeTimeout = time.Second * 20

	natsServer, err := server.NewServer(&opts)

	if err != nil {
		log.Fatal("Error create new Nats server: ", err)
	}

	authEnabled := "no"

	if o.Auth != "" {
		authEnabled = "yes"
	}

	log.Printf("Starting NATS server, port: %v, http port: %v, auth enabled: %v\n",
		o.Port, o.HTTPPort, authEnabled)

	natsServer.Start()

	natsServer.WaitForShutdown()

	// should never get here, so exit app
	log.Fatal("Nats server start returned, this should not happen, exitting app!")
	os.Exit(-1)
}
