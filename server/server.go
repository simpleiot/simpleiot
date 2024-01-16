package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/oklog/run"
	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/frontend"
	"github.com/simpleiot/simpleiot/node"
	"github.com/simpleiot/simpleiot/store"
)

// ErrServerStopped is returned when the server is stopped
var ErrServerStopped = errors.New("Server stopped")

// Options used for starting Simple IoT
type Options struct {
	StoreFile         string
	ResetStore        bool
	DataDir           string
	HTTPPort          string
	DebugHTTP         bool
	DebugLifecycle    bool
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
	LogNats           bool
	Dev               bool
	CustomUIDir       string
	CustomUIFS        fs.FS
	// optional ID (must be unique) for this instance, otherwise, a UUID will be used
	ID string
}

// Server represents a SIOT server process
type Server struct {
	nc                 *nats.Conn
	options            Options
	natsServer         *server.Server
	clients            *client.Group
	chNatsClientClosed chan struct{}
	chStop             chan struct{}
	chWaitStart        chan struct{}
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
		nats.ErrorHandler(func(_ *nats.Conn,
			sub *nats.Subscription, err error) {
			var subject string
			if sub != nil {
				subject = sub.Subject
			}
			log.Printf("Server NATS client error, sub: %v, err: %s\n", subject, err)
		}),
		nats.CustomReconnectDelay(func(attempts int) time.Duration {
			log.Println("Server NATS client reconnect attempt #", attempts)
			return time.Millisecond * 250
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Println("Server NATS client: reconnected")
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			log.Println("Server NATS client: closed")
			close(chNatsClientClosed)
		}),
		nats.ConnectHandler(func(_ *nats.Conn) {
			log.Println("Server NATS client: connected")
		}),
	)

	return &Server{
		nc:                 nc,
		options:            o,
		chNatsClientClosed: chNatsClientClosed,
		chStop:             make(chan struct{}),
		chWaitStart:        make(chan struct{}),
		clients:            client.NewGroup("Server clients"),
	}, nc, err
}

// AddClient can be used to add clients to the server.
// Clients must be added before start is called. The
// Server makes sure all clients are shut down before
// shutting down the server. This makes for a cleaner
// shutdown.
func (s *Server) AddClient(client client.RunStop) {
	s.clients.Add(client)
}

// Run the server -- only returns if there is an error
func (s *Server) Run() error {
	var g run.Group

	logLS := func(m ...any) {}

	if s.options.DebugLifecycle {
		logLS = func(m ...any) {
			log.Println(m...)
		}
	}

	o := s.options

	var err error

	// anything that needs to use the store or nats server should add to this wait group.
	// The store will wait on this before shutting down
	var storeWg sync.WaitGroup

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
			go func() {
				storeWg.Wait()
				s.natsServer.Shutdown()
				logLS("LS: Shutdown: nats server")
			}()
		})
	}

	// ====================================
	// SIOT Store
	// ====================================

	storeParams := store.Params{
		File:      o.StoreFile,
		AuthToken: o.AuthToken,
		Server:    o.NatsServer,
		Nc:        s.nc,
		ID:        s.options.ID,
	}

	siotStore, err := store.NewStore(storeParams)

	if o.ResetStore {
		if err := siotStore.Reset(); err != nil {
			log.Fatal("Error resetting store:", err)
		}
	}

	if err != nil {
		log.Fatal("Error creating store: ", err)
	}

	siotWaitCtx, siotWaitCancel := context.WithTimeout(context.Background(), time.Second*10)

	g.Add(func() error {
		err := siotStore.Run()
		logLS("LS: Exited: store")
		return err
	}, func(err error) {
		// we just run in goroutine else this Stop blocking will block everything else
		go func() {
			storeWg.Wait()
			siotWaitCancel()
			siotStore.Stop(err)
			logLS("LS: Shutdown: store")
		}()
	})

	cancelTimer := make(chan struct{})
	storeWg.Add(1)
	g.Add(func() error {
		defer storeWg.Done()
		err := siotStore.WaitStart(siotWaitCtx)
		if err != nil {
			logLS("LS: Exited: metrics timeout waiting for store")
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

		rootNode, err := client.GetNodes(s.nc, "root", "all", "", false)

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
		close(cancelTimer)
		siotStore.StopMetrics(err)
		logLS("LS: Shutdown: store metrics")
	})

	// ====================================
	// Node manager
	// ====================================

	nodeManager := node.NewManger(s.nc, o.AppVersion, o.OSVersionField)

	storeWg.Add(1)
	g.Add(func() error {
		defer storeWg.Done()
		err := siotStore.WaitStart(siotWaitCtx)
		if err != nil {
			logLS("LS: Exited: node manager timeout waiting for store")
			return err
		}

		err = nodeManager.Start()
		logLS("LS: Exited: node manager")
		return err
	}, func(err error) {
		nodeManager.Stop(err)
		logLS("LS: Shutdown: node manager")
	})

	// ====================================
	// Build in clients manager
	// ====================================

	storeWg.Add(1)
	g.Add(func() error {
		defer storeWg.Done()
		err := siotStore.WaitStart(siotWaitCtx)
		if err != nil {
			logLS("LS: Exited: client manager timeout waiting for store")
			return err
		}

		err = s.clients.Run()
		logLS("LS: Exited: clients manager: ", err)
		return err
	}, func(err error) {
		s.clients.Stop(err)
		logLS("LS: Shutdown: clients manager")
	})

	// ====================================
	// Embedded files
	// ====================================

	var feFS fs.FS

	if o.CustomUIDir != "" {
		log.Println("Using custom frontend directory: ", o.CustomUIDir)
		feFS = os.DirFS(o.CustomUIDir)
	} else if o.CustomUIFS != nil {
		feFS, err = fs.Sub(o.CustomUIFS, "public")
		if err != nil {
			log.Fatal("Error getting frontend subtree: ", err)
		}
	} else {
		if o.Dev {
			log.Println("SIOT HTTP Server -- using local instead of embedded files")
			feFS = os.DirFS("./frontend/public")
		} else {
			// remove output dir name from frontend assets filesystem
			feFS, err = fs.Sub(frontend.Content, "public")
			if err != nil {
				log.Fatal("Error getting frontend subtree: ", err)
			}
		}
	}

	// wrap with fs that will automatically look for and decompress gz
	// versions of files.
	feFSDecomp := newFsDecomp(feFS, "index.html")

	// ====================================
	// HTTP API
	// ====================================
	httpAPI := api.NewServer(api.ServerArgs{
		Port:       o.HTTPPort,
		NatsWSPort: o.NatsWSPort,
		Filesystem: http.FS(feFSDecomp),
		Debug:      o.DebugHTTP,
		JwtAuth:    siotStore.GetAuthorizer(),
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
	// and signal to waiters we have started
	chShutdown := make(chan struct{})
	g.Add(func() error {
		err := siotStore.WaitStart(siotWaitCtx)
		if err != nil {
			logLS("LS: Exited: server stopper, timeout waiting for store")
			return err
		}

		select {
		case <-s.chStop:
			logLS("LS: Exited: stop handler")
			return ErrServerStopped
		case <-chShutdown:
			logLS("LS: Exited: stop handler")
			return nil
		}
	}, func(_ error) {
		close(chShutdown)
		logLS("LS: Shutdown: stop handler")
	})

	chRunError := make(chan error)

	go func() {
		chRunError <- g.Run()
	}()

	var retErr error

done:
	for {
		select {
		// unblock any waits
		case <-s.chWaitStart:
			// No-op, reading channel is enough to unblock wait
		case retErr = <-chRunError:
			break done
		}
	}

	s.nc.Close()

	return retErr
}

// Stop server
func (s *Server) Stop(_ error) {
	close(s.chStop)
}

// WaitStart waits for server to start. Clients should wait for this
// to complete before trying to fetch nodes, etc.
func (s *Server) WaitStart(ctx context.Context) error {
	waitDone := make(chan struct{})

	go func() {
		// the following will block until the main store select
		// loop starts
		s.chWaitStart <- struct{}{}
		close(waitDone)
	}()

	select {
	case <-ctx.Done():
		return errors.New("Store wait timeout or canceled")
	case <-waitDone:
		// all is well
		return nil
	}

}
