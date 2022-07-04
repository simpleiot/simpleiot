package server

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/oklog/run"
	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/assets/files"
	"github.com/simpleiot/simpleiot/assets/frontend"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/node"
	"github.com/simpleiot/simpleiot/particle"
	"github.com/simpleiot/simpleiot/sim"
	"github.com/simpleiot/simpleiot/store"
	"github.com/simpleiot/simpleiot/system"
)

// Options used for starting Simple IoT
type Options struct {
	StoreType         store.Type
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
		nats.MaxReconnects(5),
		nats.ReconnectWait(time.Second*3),
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
		Type:      o.StoreType,
		DataDir:   o.DataDir,
		AuthToken: o.AuthToken,
		Server:    o.NatsServer,
		Key:       auth,
		Nc:        s.nc,
	}

	siotStore, err := store.NewStore(storeParams)

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

	g.Add(func() error {
		err := siotStore.WaitStart(metricsCtx)
		if err != nil {
			logLS("LS: Exited: node manager")
			return err
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
		siotStore.StopMetrics(err)
		logLS("LS: Shutdown: store metrics")
	})

	// ====================================
	// Node client manager
	// ====================================
	nodeManager := node.NewManger(s.nc, o.AppVersion, o.OSVersionField)
	nodeManagerCtx, nodeManagerCancel := context.WithTimeout(context.Background(),
		time.Second*10)

	g.Add(func() error {
		err := siotStore.WaitStart(nodeManagerCtx)
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

		err = nodeManager.Start()
		logLS("LS: Exited: node manager")
		return err
	}, func(err error) {
		nodeManagerCancel()
		nodeManager.Stop(err)
		logLS("LS: Shutdown: node manager")
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
	chShutdown := make(chan struct{})
	g.Add(func() error {
		select {
		case <-s.chStop:
			logLS("LS: Exited: stop handler")
			return errors.New("Server stopped")
		case <-chShutdown:
			logLS("LS: Exited: stop handler")
			return nil
		}
	}, func(_ error) {
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

	return nil
}

var version = "Development"

// StartArgs starts SIOT with more command line style args
func StartArgs(args []string) error {
	defaultNatsServer := "nats://localhost:4222"

	// =============================================
	// Command line options
	// =============================================
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	// configuration options
	flagDebugHTTP := flags.Bool("debugHttp", false, "Dump http requests")
	flagDebugLifecycle := flags.Bool("debugLifecycle", false, "Debug program lifecycle")
	flagSim := flags.Bool("sim", false, "Start node simulator")
	flagDisableAuth := flags.Bool("disableAuth", false, "Disable user auth (used for development)")
	flagPortal := flags.String("portal", "http://localhost:8080", "Portal URL")
	flagSendPoint := flags.String("sendPoint", "", "Send point to 'portal': 'devId:sensId:value:type'")
	flagNatsServer := flags.String("natsServer", defaultNatsServer, "NATS Server")
	flagNatsDisableServer := flags.Bool("natsDisableServer", false, "Disable NATS server (if you want to run NATS separately)")
	flagStore := flags.String("store", "bolt", "db store type: bolt, badger, memory")
	flagAuthToken := flags.String("token", "", "Auth token")
	flagNatsAck := flags.Bool("natsAck", false, "request response")
	flagID := flags.String("id", "1234", "ID of node")
	flagSyslog := flags.Bool("syslog", false, "log to syslog instead of stdout")

	// commands to run, if no commands are given the main server starts up
	flagSendPointNats := flags.String("sendPointNats", "", "Send point to 'portal' via NATS: 'devId:sensId:value:type'")
	flagSendPointText := flags.String("sendPointText", "", "Send text point to 'portal' via NATS: 'devId:sensId:text:type'")
	flagSendFile := flags.String("sendFile", "", "URL of file to send")
	flagVersion := flags.Bool("version", false, "Show version number")
	flagDumpDb := flags.Bool("dumpDb", false, "dump database to data.json file")
	flagImportDb := flags.Bool("importDb", false, "import database from data.json")
	flagLogNats := flags.Bool("logNats", false, "attach to NATS server and dump messages")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	// =============================================
	// General Setup
	// =============================================
	if *flagVersion {
		if version == "" {
			version = "Development"
		}
		fmt.Printf("SimpleIOT %v\n", version)
		os.Exit(0)
	}

	log.Printf("SimpleIOT %v\n", version)

	// set up local database
	dataDir := os.Getenv("SIOT_DATA")
	if dataDir == "" {
		dataDir = "./"
	}

	// populate files in file system
	err := files.UpdateFiles(dataDir)

	if err != nil {
		log.Println("Error updating files: ", err)
		os.Exit(-1)
	}

	// =============================================
	// NATS stuff
	// =============================================

	// populate general args
	natsPort := 4222

	natsPortE := os.Getenv("SIOT_NATS_PORT")
	if natsPortE != "" {
		n, err := strconv.Atoi(natsPortE)
		if err != nil {
			log.Println("Error parsing SIOT_NATS_PORT: ", err)
			os.Exit(-1)
		}
		natsPort = n
	}

	natsHTTPPort := 8222

	natsHTTPPortE := os.Getenv("SIOT_NATS_HTTP_PORT")
	if natsHTTPPortE != "" {
		n, err := strconv.Atoi(natsHTTPPortE)
		if err != nil {
			log.Println("Error parsing SIOT_NATS_HTTP_PORT: ", err)
			os.Exit(-1)
		}
		natsHTTPPort = n
	}

	natsWSPort := 9222
	natsWSPortE := os.Getenv("SIOT_NATS_WS_PORT")
	if natsWSPortE != "" {
		n, err := strconv.Atoi(natsWSPortE)
		if err != nil {
			log.Println("Error parsing SIOT_NATS_WS_PORT: ", err)
			os.Exit(-1)
		}
		natsWSPort = n
	}

	natsServer := *flagNatsServer
	// only consider env if command line option is something different
	// that default
	if natsServer == defaultNatsServer {
		natsServerE := os.Getenv("SIOT_NATS_SERVER")
		if natsServerE != "" {
			natsServer = natsServerE
		}
	}

	natsTLSCert := os.Getenv("SIOT_NATS_TLS_CERT")
	natsTLSKey := os.Getenv("SIOT_NATS_TLS_KEY")
	natsTLSTimeoutS := os.Getenv("SIOT_NATS_TLS_TIMEOUT")

	natsTLSTimeout := 0.5

	if natsTLSTimeoutS != "" {
		natsTLSTimeout, err = strconv.ParseFloat(natsTLSTimeoutS, 64)
		if err != nil {
			log.Println("Error parsing nats TLS timeout: ", err)
			os.Exit(-1)
		}
	}

	authToken := os.Getenv("SIOT_AUTH_TOKEN")
	if *flagAuthToken != "" {
		authToken = *flagAuthToken
	}

	if *flagSyslog {
		err := system.EnableSyslog()
		if err != nil {
			log.Println("Error enabling syslog: ", err)
		}
	}

	var nc *nats.Conn

	if *flagSendPointNats != "" ||
		*flagSendFile != "" ||
		*flagSendPointText != "" ||
		*flagLogNats {

		opts := client.EdgeOptions{
			URI:       natsServer,
			AuthToken: authToken,
			NoEcho:    true,
			Disconnected: func() {
				log.Println("NATS Disconnected")
			},
			Reconnected: func() {
				log.Println("NATS Reconnected")
			},
			Closed: func() {
				log.Println("NATS Closed")
				os.Exit(0)
			},
		}

		nc, err = client.EdgeConnect(opts)

		if err != nil {
			log.Println("Error connecting to NATS server: ", err)
			os.Exit(-1)
		}
	}

	if *flagSendFile != "" {
		err = store.NatsSendFileFromHTTP(nc, *flagID, *flagSendFile, func(percDone int) {
			log.Println("% done: ", percDone)
		})

		if err != nil {
			log.Println("Error sending file: ", err)
		}

		log.Println("File sent!")
	}

	if *flagSendPointNats != "" {
		nodeID, point, err := parsePoint(*flagSendPointNats)
		if err != nil {
			log.Println("Error parsing point: ", err)
			os.Exit(-1)
		}

		err = client.SendNodePointCreate(nc, nodeID, point, *flagNatsAck)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
	}

	if *flagSendPointText != "" {
		nodeID, point, err := parsePointText(*flagSendPointText)
		if err != nil {
			log.Println("Error parsing point: ", err)
			os.Exit(-1)
		}

		err = client.SendNodePointCreate(nc, nodeID, point, *flagNatsAck)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
	}

	if *flagLogNats {
		log.Println("Logging all NATS messages")
		_, err := nc.Subscribe("node.*.points", func(msg *nats.Msg) {
			err := client.Dump(nc, msg)
			if err != nil {
				log.Println("Error dumping nats msg: ", err)
			}
		})

		_, err = nc.Subscribe("node.*.not", func(msg *nats.Msg) {
			err := client.Dump(nc, msg)
			if err != nil {
				log.Println("Error dumping nats msg: ", err)
			}
		})

		_, err = nc.Subscribe("node.*.msg", func(msg *nats.Msg) {
			err := client.Dump(nc, msg)
			if err != nil {
				log.Println("Error dumping nats msg: ", err)
			}
		})

		_, err = nc.Subscribe("node.*.*.points", func(msg *nats.Msg) {
			err := client.Dump(nc, msg)
			if err != nil {
				log.Println("Error dumping nats msg: ", err)
			}
		})

		if err != nil {
			log.Println("Nats subscribe error: ", err)
			os.Exit(-1)
		}

		_, err = nc.Subscribe("node.*", func(msg *nats.Msg) {
			err := client.Dump(nc, msg)
			if err != nil {
				log.Println("Error dumping nats msg: ", err)
			}
		})

		if err != nil {
			log.Println("Nats subscribe error: ", err)
			os.Exit(-1)
		}

		_, err = nc.Subscribe("edge.*.*", func(msg *nats.Msg) {
			err := client.Dump(nc, msg)
			if err != nil {
				log.Println("Error dumping nats msg: ", err)
			}
		})

		if err != nil {
			log.Println("Nats subscribe error: ", err)
			os.Exit(-1)
		}

		select {}
	}

	if nc != nil {
		// wait for everything to get sent to server
		nc.Flush()
		nc.Close()
		os.Exit(0)
	}

	// =============================================
	// HTTP request
	// =============================================

	if *flagSendPoint != "" {
		err := sendPoint(*flagPortal, *flagAuthToken, *flagSendPoint)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
		os.Exit(0)
	}

	if *flagSim {
		go sim.NodeSim(*flagPortal, "1234")
		go sim.NodeSim(*flagPortal, "5678")
	}

	// =============================================
	// Dump Database
	// =============================================

	if *flagDumpDb {
		dbInst, err := store.NewDb(store.Type(*flagStore), dataDir)
		if err != nil {
			log.Println("Error opening db: ", err)
			os.Exit(-1)
		}
		defer dbInst.Close()

		f, err := os.Create("data.json")
		if err != nil {
			log.Println("Error opening data.json: ", err)
			os.Exit(-1)
		}
		err = store.DumpDb(dbInst, f)

		if err != nil {
			log.Println("Error dumping database: ", err)
			os.Exit(-1)
		}

		f.Close()
		log.Println("Database written to data.json")

		os.Exit(0)
	}

	if *flagImportDb {
		dbInst, err := store.NewDb(store.Type(*flagStore), dataDir)
		if err != nil {
			log.Println("Error opening db: ", err)
			os.Exit(-1)
		}
		defer dbInst.Close()

		f, err := os.Open("data.json")
		if err != nil {
			log.Println("Error opening data.json: ", err)
			os.Exit(-1)
		}
		err = store.ImportDb(dbInst, f)

		if err != nil {
			log.Println("Error importing database: ", err)
			os.Exit(-1)
		}

		f.Close()
		log.Println("Database imported from data.json")

		os.Exit(0)
	}

	// finally, start web server
	port := os.Getenv("SIOT_HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	osVersionField := os.Getenv("OS_VERSION_FIELD")
	if osVersionField == "" {
		osVersionField = "VERSION"
	}

	// set up particle connection if configured
	// todo -- move this to a node
	particleAPIKey := os.Getenv("SIOT_PARTICLE_API_KEY")

	// TODO, convert this to builder pattern
	o := Options{
		StoreType:         store.Type(*flagStore),
		DataDir:           dataDir,
		HTTPPort:          port,
		DebugHTTP:         *flagDebugHTTP,
		DebugLifecycle:    *flagDebugLifecycle,
		DisableAuth:       *flagDisableAuth,
		NatsServer:        natsServer,
		NatsDisableServer: *flagNatsDisableServer,
		NatsPort:          natsPort,
		NatsHTTPPort:      natsHTTPPort,
		NatsWSPort:        natsWSPort,
		NatsTLSCert:       natsTLSCert,
		NatsTLSKey:        natsTLSKey,
		NatsTLSTimeout:    natsTLSTimeout,
		AuthToken:         authToken,
		ParticleAPIKey:    particleAPIKey,
		AppVersion:        version,
		OSVersionField:    osVersionField,
	}

	var g run.Group

	siot, _, err := NewServer(o)

	if err != nil {
		siot.Stop(nil)
		return fmt.Errorf("Error starting server: %v", err)
	}

	g.Add(siot.Start, siot.Stop)

	g.Add(run.SignalHandler(context.Background(),
		syscall.SIGINT, syscall.SIGTERM))

	// add check to make sure server started
	chStartCheck := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*9)
	g.Add(func() error {
		err := siot.WaitStart(ctx)
		if err != nil {
			return errors.New("Timeout waiting for SIOT to start")
		}
		<-chStartCheck
		return nil
	}, func(err error) {
		cancel()
		close(chStartCheck)
	})

	return g.Run()
}

func parsePointText(s string) (string, data.Point, error) {
	frags := strings.Split(s, ":")
	if len(frags) != 4 {
		return "", data.Point{},
			errors.New("format for point is: 'devId:sensId:value:type'")
	}

	nodeID := frags[0]
	pointKey := frags[1]
	text := frags[2]
	pointType := frags[3]

	return nodeID, data.Point{
		Key:  pointKey,
		Type: pointType,
		Text: text,
		Time: time.Now(),
	}, nil

}

func parsePoint(s string) (string, data.Point, error) {
	frags := strings.Split(s, ":")
	if len(frags) != 4 {
		return "", data.Point{},
			errors.New("format for point is: 'devId:sensId:value:type'")
	}

	nodeID := frags[0]
	pointKey := frags[1]
	value, err := strconv.ParseFloat(frags[2], 64)
	if err != nil {
		return "", data.Point{}, err
	}

	pointType := frags[3]

	return nodeID, data.Point{
		Key:   pointKey,
		Type:  pointType,
		Value: value,
		Time:  time.Now(),
	}, nil

}

func sendPoint(portal, authToken, s string) error {
	nodeID, point, err := parsePoint(s)

	if err != nil {
		return err
	}

	sendPoints := api.NewSendPoints(portal, nodeID, authToken, time.Second*10, false)

	err = sendPoints(data.Points{point})

	return err
}
