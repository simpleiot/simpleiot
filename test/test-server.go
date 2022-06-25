package test

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/natsserver"
	"github.com/simpleiot/simpleiot/node"
	"github.com/simpleiot/simpleiot/store"
)

// Server is used to set up test servers for unit tests that run
// all in memory.
type Server struct {
	natsServer  *server.Server
	store       *store.Store
	nodeManager *node.Manager
}

// Stop a SIOT server
func (s *Server) Stop() {
	s.natsServer.Shutdown()
}

// StartServer is used to spin up a test nats store for testing
// we run everything on out of the way ports so we should not
// conflict with other running instances
func StartServer() (*Server, *nats.Conn, error) {
	var err error

	s := &Server{}

	natsOptions := natsserver.Options{
		Port:     5222,
		HTTPPort: 8900,
		WSPort:   8901,
	}

	s.natsServer, err = natsserver.StartNatsServer(natsOptions)

	storeParams := store.Params{
		Type:   store.StoreTypeMemory,
		Server: "nats://localhost:5222",
	}

	s.store, err = store.NewStore(storeParams)

	if err != nil {
		return nil, nil, fmt.Errorf("Error starting store: %v", err)
	}

	var nc *nats.Conn

	// this is a bit of a hack, but we're not sure when the NATS
	// server will be started, so try several times
	for i := 0; i < 10; i++ {
		// FIXME should we get nc with edgeConnect here?
		nc, err = s.store.Connect()
		if err != nil {
			log.Println("NATS local connect retry: ", i)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		break
	}

	if err != nil {
		return nil, nil, fmt.Errorf("Error connecting to NATs server: %v", err)
	}

	if nc == nil {
		return nil, nil, fmt.Errorf("Timeout connecting to NATs server")
	}

	// NodeManager is required to create a root node + admin user
	s.nodeManager = node.NewManger(nc, "0.0.0", "0.0.0")
	err = s.nodeManager.Init()
	if err != nil {
		return nil, nil, fmt.Errorf("Error initializing node manager: %v", err)
	}

	return s, nc, nil
}
