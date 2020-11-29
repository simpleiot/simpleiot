package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/genjidb/genji/sql/driver"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/assets/files"
	"github.com/simpleiot/simpleiot/assets/frontend"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
	"github.com/simpleiot/simpleiot/db/genji"
	"github.com/simpleiot/simpleiot/msg"
	"github.com/simpleiot/simpleiot/nats"
	"github.com/simpleiot/simpleiot/natsserver"
	"github.com/simpleiot/simpleiot/node"
	"github.com/simpleiot/simpleiot/particle"
	"github.com/simpleiot/simpleiot/sim"

	natsgo "github.com/nats-io/nats.go"
)

var siotVersion = "Development"

func parsePointText(s string) (string, data.Point, error) {
	frags := strings.Split(s, ":")
	if len(frags) != 4 {
		return "", data.Point{},
			errors.New("format for sample is: 'devId:sensId:value:type'")
	}

	nodeID := frags[0]
	sampleID := frags[1]
	text := frags[2]
	sampleType := frags[3]

	return nodeID, data.Point{
		ID:   sampleID,
		Type: sampleType,
		Text: text,
		Time: time.Now(),
	}, nil

}

func parsePoint(s string) (string, data.Point, error) {
	frags := strings.Split(s, ":")
	if len(frags) != 4 {
		return "", data.Point{},
			errors.New("format for sample is: 'devId:sensId:value:type'")
	}

	nodeID := frags[0]
	sampleID := frags[1]
	value, err := strconv.ParseFloat(frags[2], 64)
	if err != nil {
		return "", data.Point{}, err
	}

	sampleType := frags[3]

	return nodeID, data.Point{
		ID:    sampleID,
		Type:  sampleType,
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

	err = sendPoints([]data.Point{point})

	return err
}

func sendPointNats(nc *natsgo.Conn, authToken, s string, ack bool) error {
	nodeID, point, err := parsePoint(s)

	if err != nil {
		return err
	}

	subject := fmt.Sprintf("node.%v.points", nodeID)

	points := data.Points{}

	points = append(points, point)

	data, err := points.PbEncode()

	if err != nil {
		return err
	}

	if ack {
		msg, err := nc.Request(subject, data, time.Second)

		if err != nil {
			return err
		}

		if string(msg.Data) != "" {
			log.Println("Request returned error: ", string(msg.Data))
		}

	} else {
		if err := nc.Publish(subject, data); err != nil {
			return err
		}
	}

	return err
}

func sendPointText(nc *natsgo.Conn, authToken, s string, ack bool) error {
	nodeID, point, err := parsePointText(s)

	if err != nil {
		return err
	}

	subject := fmt.Sprintf("node.%v.points", nodeID)

	points := data.Points{}

	points = append(points, point)

	data, err := points.PbEncode()

	if err != nil {
		return err
	}

	if ack {
		msg, err := nc.Request(subject, data, time.Second)

		if err != nil {
			return err
		}

		if string(msg.Data) != "" {
			log.Println("Request returned error: ", string(msg.Data))
		}

	} else {
		if err := nc.Publish(subject, data); err != nil {
			return err
		}
	}

	return err
}

func parseSample(s string) (string, data.Sample, error) {
	frags := strings.Split(s, ":")
	if len(frags) != 4 {
		return "", data.Sample{},
			errors.New("format for sample is: 'devId:sensId:value:type'")
	}

	nodeID := frags[0]
	sampleID := frags[1]
	value, err := strconv.ParseFloat(frags[2], 64)
	if err != nil {
		return "", data.Sample{}, err
	}

	sampleType := frags[3]

	return nodeID, data.Sample{
		ID:    sampleID,
		Type:  sampleType,
		Value: value,
		Time:  time.Now(),
	}, nil

}

func sendSample(portal, authToken, s string) error {
	nodeID, sample, err := parseSample(s)

	if err != nil {
		return err
	}

	sendSamples := api.NewSendSamples(portal, nodeID, time.Second*10, false)

	err = sendSamples([]data.Sample{sample})

	return err
}

func sendSampleNats(nc *natsgo.Conn, authToken, s string, ack bool) error {
	nodeID, sample, err := parseSample(s)

	if err != nil {
		return err
	}

	subject := fmt.Sprintf("node.%v.samples", nodeID)

	samples := data.Samples{}

	samples = append(samples, sample)

	data, err := samples.PbEncode()

	if err != nil {
		return err
	}

	if ack {
		msg, err := nc.Request(subject, data, time.Second)

		if err != nil {
			return err
		}

		if string(msg.Data) != "" {
			log.Println("Request returned error: ", string(msg.Data))
		}

	} else {
		if err := nc.Publish(subject, data); err != nil {
			return err
		}
	}

	return err
}

func main() {
	defaultNatsServer := "nats://localhost:4222"

	// =============================================
	// Command line options
	// =============================================

	flagDebugHTTP := flag.Bool("debugHttp", false, "Dump http requests")
	flagSim := flag.Bool("sim", false, "Start node simulator")
	flagDisableAuth := flag.Bool("disableAuth", false, "Disable user auth (used for development)")
	flagPortal := flag.String("portal", "http://localhost:8080", "Portal URL")
	flagSendSample := flag.String("sendSample", "", "Send sample to 'portal': 'devId:sensId:value:type'")
	flagSendPoint := flag.String("sendPoint", "", "Send point to 'portal': 'devId:sensId:value:type'")
	flagNatsServer := flag.String("natsServer", defaultNatsServer, "NATS Server")
	flagNatsDisableServer := flag.Bool("natsDisableServer", false, "Disable NATS server (if you want to run NATS separately)")
	flagNatsAck := flag.Bool("natsAck", false, "request response")
	flagSendSampleNats := flag.String("sendSampleNats", "", "Send sample to 'portal' via NATS: 'devId:sensId:value:type'")
	flagSendPointNats := flag.String("sendPointNats", "", "Send point to 'portal' via NATS: 'devId:sensId:value:type'")
	flagSendPointText := flag.String("sendPointText", "", "Send text point to 'portal' via NATS: 'devId:sensId:text:type'")
	flagSendFile := flag.String("sendFile", "", "URL of file to send")
	flagSendCmd := flag.String("sendCmd", "", "Command to send (cmd:detail)")
	flagID := flag.String("id", "1234", "ID of node")
	flagVersion := flag.Bool("version", false, "Show version number")

	flagSyslog := flag.Bool("syslog", false, "log to syslog instead of stdout")
	flagDumpDb := flag.Bool("dumpDb", false, "dump database to data.json file")
	flagImportDb := flag.Bool("importDb", false, "import database from data.json")
	flagStore := flag.String("store", "bolt", "db store type: bolt, badger, memory")
	flagAuthToken := flag.String("token", "", "Auth token")
	flag.Parse()

	// =============================================
	// General Setup
	// =============================================
	if *flagVersion {
		if siotVersion == "" {
			siotVersion = "Development"
		}
		fmt.Printf("SimpleIOT %v\n", siotVersion)
		os.Exit(0)
	}
	fmt.Printf("SimpleIOT %v\n", siotVersion)

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
		lgr, err := syslog.New(syslog.LOG_NOTICE, "SIOT")
		if err != nil {
			log.Println("Error setting up syslog: ", err)
		} else {
			log.SetOutput(lgr)
		}
	}

	var nc *natsgo.Conn

	if *flagSendSampleNats != "" ||
		*flagSendPointNats != "" ||
		*flagSendFile != "" ||
		*flagSendCmd != "" ||
		*flagSendPointText != "" {

		opts := nats.EdgeOptions{
			Server:    natsServer,
			AuthToken: *flagAuthToken,
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

		nc, err = nats.EdgeConnect(opts)

		if err != nil {
			log.Println("Error connecting to NATS server: ", err)
			os.Exit(-1)
		}
	}

	if *flagSendFile != "" {
		err = api.NatsSendFileFromHTTP(nc, *flagID, *flagSendFile, func(percDone int) {
			log.Println("% done: ", percDone)
		})

		if err != nil {
			log.Println("Error sending file: ", err)
		}

		log.Println("File sent!")
	}

	if *flagSendCmd != "" {
		chunks := strings.Split(*flagSendCmd, ":")
		cmd := data.NodeCmd{
			ID:  *flagID,
			Cmd: chunks[0],
		}

		if len(chunks) > 1 {
			cmd.Detail = chunks[1]
		}

		err := nats.SendCmd(nc, cmd, 10*time.Second)

		if err != nil {
			log.Println("Error sending cmd: ", err)
			os.Exit(-1)
		}

		log.Println("Command sent!")
	}

	if *flagSendSampleNats != "" {
		err := sendSampleNats(nc, authToken, *flagSendSampleNats, *flagNatsAck)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
	}

	if *flagSendPointNats != "" {
		err := sendPointNats(nc, authToken, *flagSendPointNats, *flagNatsAck)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
	}

	if *flagSendPointText != "" {
		err := sendPointText(nc, authToken, *flagSendPointText, *flagNatsAck)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
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

	if *flagSendSample != "" {
		err := sendSample(*flagPortal, *flagAuthToken, *flagSendSample)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
		os.Exit(0)
	}

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
		dbInst, err := genji.NewDb(genji.StoreType(*flagStore), dataDir, nil, false)
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
		err = genji.DumpDb(dbInst, f)

		if err != nil {
			log.Println("Error dumping database: ", err)
			os.Exit(-1)
		}

		f.Close()
		log.Println("Database written to data.json")

		os.Exit(0)
	}

	if *flagImportDb {
		dbInst, err := genji.NewDb(genji.StoreType(*flagStore), dataDir, nil, false)
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
		err = genji.ImportDb(dbInst, f)

		if err != nil {
			log.Println("Error importing database: ", err)
			os.Exit(-1)
		}

		f.Close()
		log.Println("Database imported from data.json")

		os.Exit(0)
	}

	// =============================================
	// Start server, default action
	// =============================================

	// set up influxdb support if configured
	influxURL := os.Getenv("SIOT_INFLUX_URL")
	influxUser := os.Getenv("SIOT_INFLUX_USER")
	influxPass := os.Getenv("SIOT_INFLUX_PASS")
	influxDb := os.Getenv("SIOT_INFLUX_DB")

	var influx *db.Influx

	if influxURL != "" {
		var err error
		influx, err = db.NewInflux(influxURL, influxDb, influxUser, influxPass)
		if err != nil {
			log.Fatal("Error connecting to influxdb: ", err)
		}
	}

	dbInst, err := genji.NewDb(genji.StoreType(*flagStore), dataDir, influx, true)
	if err != nil {
		log.Println("Error opening db: ", err)
		os.Exit(-1)
	}
	defer dbInst.Close()

	// set up particle connection if configured
	particleAPIKey := os.Getenv("SIOT_PARTICLE_API_KEY")

	if particleAPIKey != "" {
		go func() {
			err := particle.PointReader("sample", particleAPIKey,
				func(id string, points data.Points) {
					for _, p := range points {
						err = dbInst.NodePoint(id, p)
						if err != nil {
							log.Println("Error getting particle sample: ", err)
						}
					}
				})

			if err != nil {
				fmt.Println("Get returned error: ", err)
			}
		}()
	}

	// get twilio info if enabled
	twilioSid := os.Getenv("TWILIO_SID")
	twilioAuth := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioFrom := os.Getenv("TWILIO_FROM")

	var messenger *msg.Messenger
	if twilioSid != "" && twilioAuth != "" {
		messenger = msg.NewMessenger(twilioSid, twilioAuth, twilioFrom)
	}

	// finally, start web server
	port := os.Getenv("SIOT_PORT")
	if port == "" {
		port = "8080"
	}

	var auth api.Authorizer

	if *flagDisableAuth {
		auth = api.AlwaysValid{}
	} else {
		auth, err = api.NewKey(20)
		if err != nil {
			log.Println("Error generating key: ", err)
		}
	}

	nodeManager := node.NewManger(dbInst, messenger)
	go nodeManager.Run()

	if !*flagNatsDisableServer {
		go natsserver.StartNatsServer(natsPort, natsHTTPPort, authToken,
			natsTLSCert, natsTLSKey, natsTLSTimeout)
	}

	natsHandler := api.NewNatsHandler(dbInst, authToken, natsServer)

	// this is a bit of a hack, but we're not sure when the NATS
	// server will be started, so try several times
	for i := 0; i < 10; i++ {
		err = natsHandler.Connect()
		if err != nil {
			log.Println("NATS local connect retry: ", i)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		break
	}

	if err != nil {
		log.Fatal("Error connecting to NATs server: ", err)
	}

	err = api.Server(api.ServerArgs{
		Port:       port,
		DbInst:     dbInst,
		GetAsset:   frontend.Asset,
		Filesystem: frontend.FileSystem(),
		Debug:      *flagDebugHTTP,
		JwtAuth:    auth,
		AuthToken:  authToken,
		NH:         natsHandler,
		Messenger:  messenger,
	})

	if err != nil {
		log.Println("Error starting server: ", err)
	}
}
