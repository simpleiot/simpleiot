package simpleiot

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/assets/files"
	"github.com/simpleiot/simpleiot/assets/frontend"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/natsserver"
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

// Siot is used to manage the Siot server
type Siot struct {
	dbInst  *store.Db
	options Options
}

// NewSiot create new siot instance
func NewSiot(o Options) *Siot {
	return &Siot{options: o}
}

// Close the siot server
func (s *Siot) Close() error {
	if s.dbInst != nil {
		s.dbInst.Close()
	}
	// TODO can add a lot more stuff in here for clean shutdown
	return nil
}

// Start Simple IoT data store. The nats connection returned
// can be used with helper functions in the simpleiot nats package.
// Note, this function cannot be used directly because we don't
// checkin the frontend assets for the SIOT web ui. See this
// example for how you can embed SIOT in your project by adding
// it as a submodule:
// https://github.com/simpleiot/custom-application-examples/tree/main/example-1
func (s *Siot) Start() (*nats.Conn, error) {
	// =============================================
	// Start server, default action
	// =============================================

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

	natsOptions := natsserver.Options{
		Port:       o.NatsPort,
		HTTPPort:   o.NatsHTTPPort,
		WSPort:     o.NatsWSPort,
		Auth:       o.AuthToken,
		TLSCert:    o.NatsTLSCert,
		TLSKey:     o.NatsTLSKey,
		TLSTimeout: o.NatsTLSTimeout,
	}

	if !o.NatsDisableServer {
		go natsserver.StartNatsServer(natsOptions)
	}

	storeParams := store.Params{
		Type:      o.StoreType,
		DataDir:   o.DataDir,
		AuthToken: o.AuthToken,
		Server:    o.NatsServer,
		Key:       auth,
	}

	siotStore, err := store.NewStore(storeParams)

	if err != nil {
		return nil, fmt.Errorf("Error starting store: %v", err)
	}

	var nc *nats.Conn

	// this is a bit of a hack, but we're not sure when the NATS
	// server will be started, so try several times
	for i := 0; i < 10; i++ {
		// FIXME should we get nc with edgeConnect here?
		nc, err = siotStore.Connect()
		if err != nil {
			log.Println("NATS local connect retry: ", i)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		break
	}

	if err != nil {
		return nil, fmt.Errorf("Error connecting to NATs server: %v", err)
	}

	if nc == nil {
		return nil, fmt.Errorf("Timeout connecting to NATs server")
	}

	nodeManager := node.NewManger(nc, o.AppVersion, o.OSVersionField)
	err = nodeManager.Init()
	if err != nil {
		return nil, fmt.Errorf("Error initializing node manager: %v", err)
	}
	go nodeManager.Run()

	rootNode, err := client.GetNode(nc, "root", "")

	if err != nil {
		log.Println("Error getting root id for metrics: ", err)
	} else if len(rootNode) == 0 {
		log.Println("Error getting root node, no data")
	} else {

		err = siotStore.StartMetrics(rootNode[0].ID)
		if err != nil {
			log.Println("Error starting nats metrics: ", err)
		}
	}

	// FIXME move this to a node, or get rid of it
	if o.ParticleAPIKey != "" {
		go func() {
			err := particle.PointReader("sample", o.ParticleAPIKey,
				func(id string, points data.Points) {
					err := client.SendNodePoints(nc, id, points, false)
					if err != nil {
						log.Println("Error getting particle sample: ", err)
					}
				})

			if err != nil {
				fmt.Println("Get returned error: ", err)
			}
		}()
	}

	go func() {
		err = api.Server(api.ServerArgs{
			Port:       o.HTTPPort,
			NatsWSPort: o.NatsWSPort,
			GetAsset:   frontend.Asset,
			Filesystem: frontend.FileSystem(),
			Debug:      o.DebugHTTP,
			JwtAuth:    auth,
			AuthToken:  o.AuthToken,
			Nc:         nc,
		})

		if err != nil {
			log.Fatal("Error starting SIOT HTTP interface: ", err)
		}
	}()

	return nc, err
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
	fmt.Printf("SimpleIOT %v\n", version)

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

	siot := NewSiot(o)

	nc, err = siot.Start()

	// this is not used yet
	_ = nc

	if err != nil {
		log.Fatal("Error starting SIOT store: ", err)
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Println()
		log.Println(sig)
		done <- true
	}()

	log.Println("running ...")
	<-done
	// cleanup
	log.Println("cleaning up ...")
	siot.Close()
	log.Println("exiting")

	return nil
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
