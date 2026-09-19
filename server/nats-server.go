package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

type natsServerOptions struct {
	Port int
	// HTTPPort is the monitoring port; zero turns it off.
	HTTPPort int
	// WSPort is the WebSocket port; zero picks a free one. Monitoring and
	// the WebSocket bind to loopback: monitoring has no authentication,
	// and browsers reach the WebSocket through the HTTP port's proxy.
	WSPort int
	// WSOrigins limits which page origins may open a WebSocket; empty
	// allows any.
	WSOrigins []string
	MQTTPort  int
	// Auth authenticates every connection on every listener; see
	// authorizer. AuthEnabled is only reported at start-up.
	Auth        server.Authentication
	AuthEnabled bool
	TLSCert     string
	TLSKey      string
	TLSTimeout  float64
	StoreDir    string
	// ID is the optional instance ID; when set it makes the NATS server name
	// stable across restarts, which MQTT sessions depend on.
	ID string
	// SyncInterval overrides the JetStream file sync interval; zero
	// keeps the NATS default (2m). SyncAlways fsyncs every write.
	SyncInterval time.Duration
	SyncAlways   bool
}

// newNatsServer creates a new nats server instance
func newNatsServer(o natsServerOptions) (*server.Server, error) {
	opts := server.Options{
		Port:                       o.Port,
		HTTPPort:                   o.HTTPPort,
		HTTPHost:                   "127.0.0.1",
		CustomClientAuthentication: o.Auth,
		// device credentials sign the connection nonce
		AlwaysEnableNonce: true,
		NoSigs:            true,
		JetStream:         true,
		StoreDir:          o.StoreDir,
	}

	if o.SyncAlways {
		opts.SyncAlways = true
	} else if o.SyncInterval > 0 {
		opts.SyncInterval = o.SyncInterval
	}

	if o.TLSCert != "" && o.TLSKey != "" {
		log.Println("Setting up NATS TLS ...")
		opts.TLS = true
		opts.TLSCert = o.TLSCert
		opts.TLSKey = o.TLSKey
		opts.TLSTimeout = o.TLSTimeout
		tc := server.TLSConfigOpts{CertFile: o.TLSCert, KeyFile: o.TLSKey}

		var err error
		opts.TLSConfig, err = server.GenTLSConfig(&tc)

		if err != nil {
			return nil, fmt.Errorf("error setting up TLS: %v", err)
		}
	}

	if o.MQTTPort != 0 {
		// NATS keys MQTT sessions and retained messages to the server name,
		// so it has to be set, and it has to be the same after a restart or
		// clients lose their sessions.
		opts.ServerName = natsServerName(o)
		opts.MQTT.Port = o.MQTTPort
		opts.MQTT.AuthTimeout = o.TLSTimeout

		if opts.TLSConfig != nil {
			opts.MQTT.TLSConfig = opts.TLSConfig
			opts.MQTT.TLSTimeout = o.TLSTimeout
		}
	}

	opts.Websocket.Port = o.WSPort
	if o.WSPort == 0 {
		opts.Websocket.Port = server.RANDOM_PORT
	}
	opts.Websocket.Host = "127.0.0.1"
	opts.Websocket.AuthTimeout = o.TLSTimeout
	// the listener serves TLS whenever the server has a certificate; the
	// HTTP port's proxy pins that certificate
	opts.Websocket.NoTLS = opts.TLSConfig == nil
	if opts.TLSConfig != nil {
		opts.Websocket.TLSConfig = opts.TLSConfig
	}
	opts.Websocket.HandshakeTimeout = time.Second * 20
	// the HTTP server's proxy forwards the page's Origin header
	opts.Websocket.AllowedOrigins = o.WSOrigins

	natsServer, err := server.NewServer(&opts)

	if err != nil {
		return nil, fmt.Errorf("error create new Nats server: %v", err)
	}

	authEnabled := "no"

	if o.AuthEnabled {
		authEnabled = "yes"
	}

	log.Printf("NATS server, port: %v, monitoring port: %v, auth enabled: %v\n",
		o.Port, o.HTTPPort, authEnabled)

	if o.MQTTPort != 0 {
		log.Printf("NATS server MQTT enabled on port: %v, server name: %v\n",
			o.MQTTPort, opts.ServerName)
	}

	return natsServer, nil
}

// natsServerName returns a name for this NATS server that stays the same
// across restarts. The instance ID is used when one is configured; otherwise
// the store directory identifies the instance, since two instances on one
// machine never share it.
func natsServerName(o natsServerOptions) string {
	if o.ID != "" {
		return "siot-" + o.ID
	}

	dir, err := filepath.Abs(o.StoreDir)
	if err != nil {
		dir = o.StoreDir
	}

	sum := sha256.Sum256([]byte(dir))

	return "siot-" + hex.EncodeToString(sum[:6])
}
