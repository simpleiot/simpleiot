package main

import (
	"flag"
	"log"
	"os"

	"github.com/simpleiot/simpleiot/nats"
)

func main() {
	flagNatsServer := flag.String("natsServer", "nats://localhost:4222", "NATS Server")
	flagID := flag.String("id", "1234", "ID of edge device")
	flagNatsAuth := flag.String("natsAuth", "", "NATS auth token")

	flag.Parse()

	log.Printf("SIOT Edge, ID: %v, server: %v\n", *flagID, *flagNatsServer)

	opts := nats.EdgeOptions{
		Server:    *flagNatsServer,
		AuthToken: *flagNatsAuth,
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

	nc, err := nats.EdgeConnect(opts)

	if err != nil {
		log.Println("Error connecting to NATS server: ", err)
		os.Exit(-1)
	}

	nats.ListenForFile(nc, "./", *flagID, func(name string) {
		log.Println("File downloaded: ", name)
	})

	select {}
}
