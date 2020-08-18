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

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/assets/files"
	"github.com/simpleiot/simpleiot/assets/frontend"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
	"github.com/simpleiot/simpleiot/device"
	"github.com/simpleiot/simpleiot/msg"
	"github.com/simpleiot/simpleiot/nats"
	"github.com/simpleiot/simpleiot/natsserver"
	"github.com/simpleiot/simpleiot/particle"
	"github.com/simpleiot/simpleiot/sim"

	natsgo "github.com/nats-io/nats.go"
)

func parseSample(s string) (string, data.Sample, error) {
	frags := strings.Split(s, ":")
	if len(frags) != 4 {
		return "", data.Sample{},
			errors.New("format for sample is: 'devId:sensId:value:type'")
	}

	deviceID := frags[0]
	sampleID := frags[1]
	value, err := strconv.ParseFloat(frags[2], 64)
	if err != nil {
		return "", data.Sample{}, err
	}

	sampleType := frags[3]

	return deviceID, data.Sample{
		ID:    sampleID,
		Type:  sampleType,
		Value: value,
		Time:  time.Now(),
	}, nil

}

func send(portal, s string) error {
	deviceID, sample, err := parseSample(s)

	if err != nil {
		return err
	}

	sendSamples := api.NewSendSamples(portal, deviceID, time.Second*10, false)

	err = sendSamples([]data.Sample{sample})

	return err
}

func sendNats(natsServer, authToken, s string, count int) error {
	deviceID, sample, err := parseSample(s)

	if err != nil {
		return err
	}

	nc, err := nats.EdgeConnect(natsServer, authToken)

	if err != nil {
		return err
	}
	defer nc.Close()

	nc.SetErrorHandler(func(_ *natsgo.Conn, _ *natsgo.Subscription,
		err error) {
		log.Printf("NATS Error: %s\n", err)
	})

	nc.SetReconnectHandler(func(_ *natsgo.Conn) {
		log.Println("NATS Reconnected!")
	})

	nc.SetDisconnectHandler(func(_ *natsgo.Conn) {
		log.Println("NATS Disconnected!")
	})

	nc.SetClosedHandler(func(_ *natsgo.Conn) {
		panic("Connection to NATS is closed!")
	})

	subject := fmt.Sprintf("device.%v.samples", deviceID)

	samples := data.Samples{}

	//for i := 0; i < count; i++ {
	samples = append(samples, sample)
	//}

	data, err := samples.PbEncode()

	if err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		if err := nc.Publish(subject, data); err != nil {
			return err
		}
		time.Sleep(time.Second)
	}

	// wait for everything to get sent to server
	nc.Flush()

	nc.Close()

	return err
}

func main() {
	defaultNatsServer := "nats://localhost:4222"
	flagDebugHTTP := flag.Bool("debugHttp", false, "Dump http requests")
	flagSim := flag.Bool("sim", false, "Start device simulator")
	flagDisableAuth := flag.Bool("disableAuth", false, "Disable auth (used for development)")
	flagPortal := flag.String("portal", "http://localhost:8080", "Portal URL")
	flagSendSample := flag.String("sendSample", "", "Send sample to 'portal': 'devId:sensId:value:type'")
	flagNatsServer := flag.String("natsServer", defaultNatsServer, "NATS Server")
	flagNatsDisableServer := flag.Bool("natsDisableServer", false, "Disable NATS server (if you want to run NATS separately)")
	flagSendSampleNats := flag.String("sendSampleNats", "", "Send sample to 'portal' via NATS: 'devId:sensId:value:type'")
	flagSendCount := flag.Int("sendCount", 1, "number of samples to send")
	flagSyslog := flag.Bool("syslog", false, "log to syslog instead of stdout")
	flagDumpDb := flag.Bool("dumpDb", false, "dump database to file")
	flagAuthToken := flag.String("token", "", "Auth token")
	flag.Parse()

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

	if *flagSendSample != "" {
		err := send(*flagPortal, *flagSendSample)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
		os.Exit(0)
	}

	if *flagSendSampleNats != "" {
		err := sendNats(*flagNatsServer, authToken,
			*flagSendSampleNats, *flagSendCount)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
		os.Exit(0)
	}

	if *flagSim {
		go sim.DeviceSim(*flagPortal, "1234")
		go sim.DeviceSim(*flagPortal, "5678")
	}

	// default action is to start server

	if *flagDumpDb {
		dbInst, err := db.NewDb(dataDir, nil, false)
		if err != nil {
			log.Println("Error opening db: ", err)
			os.Exit(-1)
		}

		f, err := os.Create("data.json")
		if err != nil {
			log.Println("Error opening data.json: ", err)
			os.Exit(-1)
		}
		err = db.DumpDb(dbInst, f)

		if err != nil {
			log.Println("Error dumping database: ", err)
			os.Exit(-1)
		}

		f.Close()
		log.Println("Database written to data.json")

		os.Exit(0)
	}

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

	dbInst, err := db.NewDb(dataDir, influx, true)
	if err != nil {
		log.Println("Error opening db: ", err)
		os.Exit(-1)
	}

	// set up particle connection if configured
	particleAPIKey := os.Getenv("SIOT_PARTICLE_API_KEY")

	if particleAPIKey != "" {
		go func() {
			err := particle.SampleReader("sample", particleAPIKey,
				func(id string, samples []data.Sample) {
					for _, s := range samples {
						err = dbInst.DeviceSample(id, s)
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

	deviceManager := device.NewManger(dbInst, messenger)
	go deviceManager.Run()

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
		Auth:       auth,
		NH:         natsHandler,
	})

	if err != nil {
		log.Println("Error starting server: ", err)
	}
}
