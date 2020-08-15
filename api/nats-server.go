package api

import (
	"log"
	"os"

	"github.com/nats-io/nats-server/v2/server"
)

// StartNatsServer starts a nats server instance. This function will block
// so should be started with a go routine
func StartNatsServer(port, httpPort int, auth string) {
	opts := server.Options{
		Port:          port,
		HTTPPort:      httpPort,
		Authorization: auth,
		TLS:           true,
		TLSCert:       "server-cert.pem",
		TLSKey:        "server-key.pem",
	}

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

	natsServer, err := server.NewServer(&opts)

	if err != nil {
		log.Fatal("Error create new Nats server: ", err)
	}

	natsServer.Start()

	natsServer.WaitForShutdown()

	// should never get here, so exit app
	log.Fatal("Nats server start returned, this should not happen, exitting app!")
	os.Exit(-1)
}
