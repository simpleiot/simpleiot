package main

import (
	"flag"
	"log"

	"github.com/simpleiot/simpleiot/farmation/app"
)

func main() {
	flagSim := flag.Bool("sim", false, "Start device simulator")
	flag.Parse()
	log.Println("Starting IS app ...")
	app.Run(*flagSim)
}
