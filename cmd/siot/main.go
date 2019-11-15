package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/assets/frontend"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db"
	"github.com/simpleiot/simpleiot/particle"
	"github.com/simpleiot/simpleiot/sim"
)

func main() {
	flagSim := flag.Bool("sim", false, "Start device simulator")
	flagSimPortal := flag.String("simPortal", "http://localhost:8080", "Portal URL")
	flagSimDeviceID := flag.String("simDeviceId", "1234", "Simulation Device ID")
	flagDebugHTTP := flag.Bool("debugHttp", false, "Dump http requests")
	flag.Parse()

	if *flagSim {
		sim.DeviceSim(*flagSimPortal, *flagSimDeviceID)
	}

	// default action is to start server

	// set up local database
	dataDir := os.Getenv("SIOT_DATA")
	if dataDir == "" {
		dataDir = "./"
	}

	dbInst, err := db.NewDb(dataDir)
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

	err = api.Server(port, dbInst, influx, frontend.Asset,
		frontend.FileSystem(), *flagDebugHTTP)

	if err != nil {
		log.Println("Error starting server: ", err)
	}
}
