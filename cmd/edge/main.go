package main

import (
	"flag"
	"log"
	"net"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/api"
)

func main() {
	flagNatsServer := flag.String("natsServer", "nats://localhost:4222", "NATS Server")
	flagID := flag.String("id", "1234", "ID of edge device")

	flag.Parse()

	log.Printf("SIOT Edge, ID: %v, server: %v\n", *flagID, *flagNatsServer)

	nc, err := nats.Connect(*flagNatsServer,
		nats.Timeout(10*time.Second),
		nats.PingInterval(60*2*time.Second),
		nats.MaxPingsOutstanding(5),
		nats.ReconnectBufSize(5*1024*1024),
		nats.MaxReconnects(-1),
		nats.SetCustomDialer(&net.Dialer{
			KeepAlive: -1,
		}),
		//nats.Token(authToken),
	)
	if err != nil {
		log.Println("Error connecting to NATS server: ", err)
		os.Exit(-1)
	}

	log.Println("Connected to server")

	nc.SetErrorHandler(func(_ *nats.Conn, _ *nats.Subscription,
		err error) {
		log.Printf("NATS Error: %s\n", err)
	})

	nc.SetReconnectHandler(func(_ *nats.Conn) {
		log.Println("NATS Reconnected!")
	})

	nc.SetDisconnectHandler(func(_ *nats.Conn) {
		log.Println("NATS Disconnected!")
	})

	nc.SetClosedHandler(func(_ *nats.Conn) {
		panic("Connection to NATS is closed!")
	})

	api.NatsListenForFile(nc, *flagID)

	select {}

	defer nc.Close()
}
