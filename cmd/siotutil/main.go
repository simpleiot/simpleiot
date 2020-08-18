package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/internal/pb"
	"github.com/simpleiot/simpleiot/nats"
	"google.golang.org/protobuf/proto"
)

func main() {
	flagNatsServer := flag.String("natsServer", "nats://localhost:4222", "NATS Server")
	flagSendFile := flag.String("sendFile", "", "URL of file to send")
	flagSendCmd := flag.String("sendCmd", "", "Command to send (cmd:detail)")
	flagSendVersion := flag.String("sendVersion", "", "Command to send version to portal (HW:OS:App)")
	flagID := flag.String("id", "1234", "ID of edge device")
	flagNatsAuth := flag.String("natsAuth", "", "NATS auth token")

	flag.Parse()

	if (*flagSendFile == "" && *flagSendCmd == "" && *flagSendVersion == "") || *flagID == "" {
		log.Println("Error, must provide sendFile/sendCmd and device")
		flag.Usage()
		os.Exit(-1)
	}

	nc, err := nats.EdgeConnect(*flagNatsServer, *flagNatsAuth)

	if err != nil {
		log.Println("Error connecting to NATS server: ", err)
		os.Exit(-1)
	}

	log.Println("Connected to server")

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

	if *flagSendFile != "" {
		err = api.NatsSendFileFromHTTP(nc, *flagID, *flagSendFile, func(percDone int) {
			log.Println("% done: ", percDone)
		})

		if err != nil {
			log.Println("Error sending file: ", err)
			os.Exit(-1)
		}

		log.Println("File sent!")
	}

	if *flagSendCmd != "" {
		chunks := strings.Split(*flagSendCmd, ":")
		cmd := data.DeviceCmd{
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

	if *flagSendVersion != "" {
		chunks := strings.Split(*flagSendVersion, ":")
		if len(chunks) < 3 {
			log.Println("Error, we need 3 chunks for version")
			flag.Usage()
			os.Exit(-1)
		}

		v := &pb.DeviceVersion{
			Hw:  chunks[0],
			Os:  chunks[1],
			App: chunks[2],
		}

		out, err := proto.Marshal(v)

		if err != nil {
			log.Println("Error marshalling version: ", err)
			os.Exit(-1)
		}

		subject := fmt.Sprintf("device.%v.version", *flagID)
		err = nc.Publish(subject, out)

		if err != nil {
			log.Println("Error sending version: ", err)
			os.Exit(-1)
		}

		log.Println("Version sent!")
	}
}
