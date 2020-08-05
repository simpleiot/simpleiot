package main

import (
	"flag"
	"log"
	"os"

	"github.com/simpleiot/simpleiot/api"
)

func main() {
	flagNatsServer := flag.String("natsServer", "nats://localhost:4222", "NATS Server")
	flagID := flag.String("id", "1234", "ID of edge device")

	flag.Parse()

	log.Printf("SIOT Edge, ID: %v, server: %v\n", *flagID, *flagNatsServer)

	nc, err := api.NatsEdgeConnect(*flagNatsServer, "")

	if err != nil {
		log.Println("Error connecting to NATS server: ", err)
		os.Exit(-1)
	}

	log.Println("Connected to server")

	api.NatsListenForFile(nc, *flagID, func(name string) {
		log.Println("File downloaded: ", name)
	})

	select {}

	// FIXME, add exit handler
	defer nc.Close()
}
