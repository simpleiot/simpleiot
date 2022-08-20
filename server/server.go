package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/oklog/run"
	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/assets/frontend"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/node"
	"github.com/simpleiot/simpleiot/particle"
	"github.com/simpleiot/simpleiot/store"
)

// Options used for starting Simple IoT
type Options struct {
	StoreFile         string
	DataDir           string
	HTTPPort          string
	DebugHTTP         bool
	DebugLifecycle    bool
	DisableAuth       bool
	NatsServer        string
	NatsDisableServer bool
	NatsPort          int
	NatsHTTPPort      int
	NatsWSPort        int
	NatsTLSCert       string
	NatsTLSKey        string
	NatsTLSTimeout    float64
	AuthToken         string
	ParticleAPIKey    string
	AppVersion        string
	OSVersionField    string
}

// Server represents a SIOT server process
type Server struct {
	nc                 *nats.Conn
	options            Options
	natsServer         *server.Server
	chNatsClientClosed chan struct{}
	chStop             chan struct{}

	// sync stuff
	startedLock sync.Mutex
	started     bool
	wait        []chan struct{}
}

// NewServer creates a new server
func NewServer(o Options) (*Server, *nats.Conn, error) {
	chNatsClientClosed := make(chan struct{})

	// start the server side nats client
	nc, err := nats.Connect(o.NatsServer,
		nats.Timeout(10*time.Second),
		nats.PingInterval(60*5*time.Second),
		nats.MaxPingsOutstanding(5),
		nats.ReconnectBufSize(5*1024*1024),
		nats.SetCustomDialer(&net.Dialer{
			KeepAlive: -1,
		}),
		nats.Token(o.AuthToken),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(60),
		nats.ReconnectWait(time.Millisecond*250),
		nats.ErrorHandler(func(_ *nats.Conn,
			sub *nats.Subscription, err error) {
			log.Printf("NATS server client error, sub: %v, err: %s\n", sub.Subject, err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Println("Nats server client reconnect")
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			log.Println("Nats server client closed")
			close(chNatsClientClosed)
		}),
	)

	return &Server{
		nc:                 nc,
		options:            o,
		chNatsClientClosed: chNatsClientClosed,
		chStop:             make(chan struct{}),
	}, nc, err
}

// Start the server -- only returns if there is an error
func (s *Server) Start() error {
	var g run.Group

	logLS := func(m ...any) {}

	if s.options.DebugLifecycle {
		logLS = func(m ...any) {
			log.Println(m...)
		}
	}

	o := s.options

	var auth api.Authorizer
	var err error

	if o.DisableAuth {
		auth = api.AlwaysValid{}
	} else {
		auth, err = api.NewKey(20)
		if err != nil {
			log.Println("Error generating key: ", err)
		}
	}

	// ====================================
	// Nats server
	// ====================================
	natsOptions := natsServerOptions{
		Port:       o.NatsPort,
		HTTPPort:   o.NatsHTTPPort,
		WSPort:     o.NatsWSPort,
		Auth:       o.AuthToken,
		TLSCert:    o.NatsTLSCert,
		TLSKey:     o.NatsTLSKey,
		TLSTimeout: o.NatsTLSTimeout,
	}

	if !o.NatsDisableServer {
		s.natsServer, err = newNatsServer(natsOptions)
		if err != nil {
			return fmt.Errorf("Error setting up nats server: %v", err)
		}

		g.Add(func() error {
			s.natsServer.Start()
			s.natsServer.WaitForShutdown()
			logLS("LS: Exited: nats server")
			return fmt.Errorf("NATS server stopped")
		}, func(err error) {
			s.natsServer.Shutdown()
			logLS("LS: Shutdown: nats server")
		})
	}

	// ====================================
	// Monitor Nats server client
	// ====================================
	g.Add(func() error {
		// block until client exits
		<-s.chNatsClientClosed
		logLS("LS: Exited: nats client")
		return errors.New("Nats server client closed")
	}, func(error) {
		s.nc.Close()
		logLS("LS: Shutdown: nats client")
	})

	// ====================================
	// SIOT Store
	// ====================================
	storeParams := store.Params{
		File:      o.StoreFile,
		AuthToken: o.AuthToken,
		Server:    o.NatsServer,
		Key:       auth,
		Nc:        s.nc,
	}

	siotStore, err := store.NewStore(storeParams)

	if err != nil {
		log.Fatal("Error creating store: ", err)
	}

	g.Add(func() error {
		err := siotStore.Start()
		logLS("LS: Exited: store")
		return err
	}, func(err error) {
		siotStore.Stop(err)
		logLS("LS: Shutdown: store")
	})

	metricsCtx, metricsCancel := context.WithTimeout(context.Background(),
		time.Second*10)
	cancelTimer := make(chan struct{})

	g.Add(func() error {
		err := siotStore.WaitStart(metricsCtx)
		if err != nil {
			logLS("LS: Exited: node manager")
			return err
		}

		// Hack -- this needs moved to a client
		t := time.NewTimer(10 * time.Second)

		select {
		case <-t.C:
		case <-cancelTimer:
			logLS("LS: Exited: store metrics")
			return nil
		}

		rootNode, err := client.GetNode(s.nc, "root", "")

		if err != nil {
			logLS("LS: Exited: store metrics")
			return fmt.Errorf("Error getting root id for metrics: %v", err)
		} else if len(rootNode) == 0 {
			logLS("LS: Exited: store metrics")
			return fmt.Errorf("Error getting root node, no data")
		}

		err = siotStore.StartMetrics(rootNode[0].ID)
		logLS("LS: Exited: store metrics")
		return err
	}, func(err error) {
		metricsCancel()
		close(cancelTimer)
		siotStore.StopMetrics(err)
		logLS("LS: Shutdown: store metrics")
	})

	// ====================================
	// Node client manager
	// ====================================
	nodeManager := node.NewManger(s.nc, o.AppVersion, o.OSVersionField)

	g.Add(func() error {
		err := nodeManager.Start()
		logLS("LS: Exited: node manager")
		return err
	}, func(err error) {
		nodeManager.Stop(err)
		logLS("LS: Shutdown: node manager")
	})

	// ====================================
	// Build in clients manager
	// ====================================

	clientsManager := client.NewBuiltInClients(s.nc)
	g.Add(func() error {
		err := clientsManager.Start()
		logLS("LS: Exited: clients manager")
		return err
	}, func(err error) {
		clientsManager.Stop(err)
		logLS("LS: Shutdown: clients manager")
	})

	// ====================================
	// Particle client
	// FIXME move this to a node, or get rid of it
	// ====================================

	if o.ParticleAPIKey != "" {
		go func() {
			err := particle.PointReader("sample", o.ParticleAPIKey,
				func(id string, points data.Points) {
					err := client.SendNodePoints(s.nc, id, points, false)
					if err != nil {
						log.Println("Error getting particle sample: ", err)
					}
				})

			if err != nil {
				log.Println("Get returned error: ", err)
			}
		}()
	}

	// ====================================
	// HTTP API
	// ====================================
	httpAPI := api.NewServer(api.ServerArgs{
		Port:       o.HTTPPort,
		NatsWSPort: o.NatsWSPort,
		GetAsset:   frontend.Asset,
		Filesystem: frontend.FileSystem(),
		Debug:      o.DebugHTTP,
		JwtAuth:    auth,
		AuthToken:  o.AuthToken,
		Nc:         s.nc,
	})

	g.Add(func() error {
		err := httpAPI.Start()
		logLS("LS: Exited: http api")
		return err
	}, func(err error) {
		httpAPI.Stop(err)
		logLS("LS: Shutdown: http api")
	})

	// Give us a way to stop the server
	nodeManagerCtx, nodeManagerCancel := context.WithTimeout(context.Background(),
		time.Second*10)

	chShutdown := make(chan struct{})
	g.Add(func() error {
		err := nodeManager.WaitStart(nodeManagerCtx)
		if err != nil {
			logLS("LS: Exited: node manager")
			return err
		}

		// signal that the server is started
		s.startedLock.Lock()
		s.started = true
		for _, c := range s.wait {
			close(c)
		}
		s.startedLock.Unlock()

		select {
		case <-s.chStop:
			logLS("LS: Exited: stop handler")
			return errors.New("Server stopped")
		case <-chShutdown:
			logLS("LS: Exited: stop handler")
			return nil
		}
	}, func(_ error) {
		nodeManagerCancel()
		close(chShutdown)
		logLS("LS: Shutdown: stop handler")
	})

	// now, run all this stuff
	return g.Run()
}

// Stop server
func (s *Server) Stop(err error) {
	s.nc.Close()
	close(s.chStop)
}

// WaitStart waits for server to start. Clients should wait for this
// to complete before trying to fetch nodes, etc.
func (s *Server) WaitStart(ctx context.Context) error {
	s.startedLock.Lock()
	if s.started {
		s.startedLock.Unlock()
		return nil
	}

	wait := make(chan struct{})
	s.wait = append(s.wait, wait)
	s.startedLock.Unlock()

	select {
	case <-ctx.Done():
		return errors.New("server wait timeout or canceled")
	case <-wait:
		return nil
	}
}
