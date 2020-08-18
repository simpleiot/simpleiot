package main

import (
	"flag"
	"log"
	"os"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/nats"
)

func main() {
	flagNatsServer := flag.String("natsServer", "nats://localhost:4222", "NATS Server")
	flagID := flag.String("id", "1234", "ID of edge device")
	flagNatsAuth := flag.String("natsAuth", "", "NATS auth token")

	flag.Parse()

	log.Printf("SIOT Edge, ID: %v, server: %v\n", *flagID, *flagNatsServer)

	nc, err := nats.NatsEdgeConnect(*flagNatsServer, *flagNatsAuth)

	if err != nil {
		log.Println("Error connecting to NATS server: ", err)
		os.Exit(-1)
	}

	log.Println("Connected to server")

	nats.NatsListenForFile(nc, "./", *flagID, func(name string) {
		log.Println("File downloaded: ", name)
	})

	nats.NatsListenForCmd(nc, *flagID, func(cmd data.DeviceCmd) {
		log.Println("Received command: ", cmd)
	})

	select {}
}
