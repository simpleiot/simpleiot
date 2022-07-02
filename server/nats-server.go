package server

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

type natsServerOptions struct {
	Port       int
	HTTPPort   int
	WSPort     int
	Auth       string
	TLSCert    string
	TLSKey     string
	TLSTimeout float64
}

// newNatsServer creates a new nats server instance
func newNatsServer(o natsServerOptions) (*server.Server, error) {
	opts := server.Options{
		Port:          o.Port,
		HTTPPort:      o.HTTPPort,
		Authorization: o.Auth,
		NoSigs:        true,
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
			return nil, fmt.Errorf("Error setting up TLS: %v", err)
		}
	}

	if o.WSPort != 0 {
		opts.Websocket.Port = o.WSPort
		opts.Websocket.Token = o.Auth
		opts.Websocket.AuthTimeout = o.TLSTimeout
		opts.Websocket.NoTLS = true // will likely be fronted by Caddy anyway
		opts.Websocket.HandshakeTimeout = time.Second * 20
	}

	natsServer, err := server.NewServer(&opts)

	if err != nil {
		return nil, fmt.Errorf("Error create new Nats server: %v", err)
	}

	authEnabled := "no"

	if o.Auth != "" {
		authEnabled = "yes"
	}

	log.Printf("NATS server, port: %v, http port: %v, auth enabled: %v\n",
		o.Port, o.HTTPPort, authEnabled)

	if o.WSPort != 0 {
		log.Printf("NATS server WS enabled on port: %v\n", o.WSPort)
	}

	return natsServer, nil
}
