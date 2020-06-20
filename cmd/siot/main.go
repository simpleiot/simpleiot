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
	"github.com/simpleiot/simpleiot/assets/frontend"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
	"github.com/simpleiot/simpleiot/particle"
	"github.com/simpleiot/simpleiot/sim"
)

func send(portal, sample string) error {
	frags := strings.Split(sample, ":")
	if len(frags) != 4 {
		return errors.New("format for sample is: 'devId:sensId:value:type'")
	}

	deviceID := frags[0]
	sampleID := frags[1]
	value, err := strconv.ParseFloat(frags[2], 64)
	if err != nil {
		return err
	}

	sampleType := frags[3]

	sendSamples := api.NewSendSamples(portal, deviceID, time.Second*10, false)

	err = sendSamples([]data.Sample{
		{
			ID:    sampleID,
			Type:  sampleType,
			Value: value,
		},
	})

	return err
}

func main() {
	flagDebugHTTP := flag.Bool("debugHttp", false, "Dump http requests")
	flagSim := flag.Bool("sim", false, "Start device simulator")
	flagDisableAuth := flag.Bool("disableAuth", false, "Disable auth (used for development)")
	flagPortal := flag.String("portal", "http://localhost:8080", "Portal URL")
	flagSendSample := flag.String("sendSample", "", "Send sample to 'portal': 'devId:sensId:value:type'")
	flagSyslog := flag.Bool("syslog", false, "log to syslog instead of stdout")
	flagDumpDb := flag.Bool("dumpDb", false, "dump database to file")
	flag.Parse()

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

	if *flagSim {
		go sim.DeviceSim(*flagPortal, "1234")
		go sim.DeviceSim(*flagPortal, "5678")
	}

	// default action is to start server

	// set up local database
	dataDir := os.Getenv("SIOT_DATA")
	if dataDir == "" {
		dataDir = "./"
	}

	if *flagDumpDb {
		dbInst, err := api.NewDb(dataDir, false)
		if err != nil {
			log.Println("Error opening db: ", err)
			os.Exit(-1)
		}

		f, err := os.Create("data.json")
		if err != nil {
			log.Println("Error opening data.json: ", err)
			os.Exit(-1)
		}
		err = api.DumpDb(dbInst, f)

		if err != nil {
			log.Println("Error dumping database: ", err)
			os.Exit(-1)
		}

		f.Close()
		log.Println("Database written to data.json")

		os.Exit(0)
	}

	dbInst, err := api.NewDb(dataDir, true)
	if err != nil {
		log.Println("Error opening db: ", err)
		os.Exit(-1)
	}

	// set up influxdb support if configured
	influxURL := os.Getenv("SIOT_INFLUX_URL")
	influxUser := os.Getenv("SIOT_INFLUX_USER")
	influxPass := os.Getenv("SIOT_INFLUX_PASS")

	var influx *db.Influx

	if influxURL != "" {
		influx, err = db.NewInflux(influxURL, "siot", influxUser, influxPass)
		if err != nil {
			log.Fatal("Error connecting to influxdb: ", err)
		}
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
					if influx != nil {
						err = influx.WriteSamples(samples)
						if err != nil {
							log.Println("Error writing particle samples to influx: ", err)
						}
					}
				})

			if err != nil {
				fmt.Println("Get returned error: ", err)
			}
		}()
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

	err = api.Server(api.ServerArgs{
		Port:       port,
		DbInst:     dbInst,
		Influx:     influx,
		GetAsset:   frontend.Asset,
		Filesystem: frontend.FileSystem(),
		Debug:      *flagDebugHTTP,
		Auth:       auth})

	if err != nil {
		log.Println("Error starting server: ", err)
	}
}
