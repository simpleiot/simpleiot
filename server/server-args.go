package server

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/oklog/run"
	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/assets/files"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/sim"
	"github.com/simpleiot/simpleiot/store"
	"github.com/simpleiot/simpleiot/system"
)

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
	flagStore := flags.String("store", "siot.sqlite", "store file, default siot.sqlite")
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

	storeFilePath := path.Join(dataDir, *flagStore)

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
		log.Fatal("not supported")

		/* FIXME
		dbInst, err := store.NewSqliteDb(*flagStore, dataDir)
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
		*/

		os.Exit(0)
	}

	if *flagImportDb {
		log.Fatal("not supported")
		/* FIXME
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
		*/

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
		StoreFile:         storeFilePath,
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
