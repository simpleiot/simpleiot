package main

import (
	"flag"
	"log"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/assets/frontend"
	"github.com/simpleiot/simpleiot/sim"
)

func main() {
	flagSim := flag.Bool("sim", false, "Start device simulator")
	flagSimPortal := flag.String("simPortal", "http://localhost:8080", "Portal URL")
	flagSimDeviceID := flag.String("simDeviceId", "1234", "Simulation Device ID")

	flag.Parse()

	if *flagSim {
		sim.DeviceSim(*flagSimPortal, *flagSimDeviceID)
	}

	// default action is to start server
	err := api.Server(frontend.Asset, frontend.FileSystem())
	if err != nil {
		log.Println("Error starting server: ", err)
	}
}
