package natsserver

import (
	"log"
	"os"

	"github.com/nats-io/nats-server/v2/server"
)

// StartNatsServer starts a nats server instance. This function will block
// so should be started with a go routine
func StartNatsServer(port, httpPort int, auth, tlsCert, tlsKey string, tlsTimeout float64) {
	opts := server.Options{
		Port:          port,
		HTTPPort:      httpPort,
		Authorization: auth,
	}

	if tlsCert != "" && tlsKey != "" {
		log.Println("Setting up NATS TLS ...")
		opts.TLS = true
		opts.TLSCert = tlsCert
		opts.TLSKey = tlsKey
		opts.TLSTimeout = tlsTimeout
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
