package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/simpleiot/simpleiot/msg"
)

func main() {
	flagTo := flag.String("to", "", "destination phone number")
	flagFrom := flag.String("from", "", "source phone number")
	flagMsg := flag.String("msg", "", "message to send")

	flag.Parse()

	sid := os.Getenv("TWILIO_SID")
	auth := os.Getenv("TWILIO_AUTH_TOKEN")

	if *flagTo == "" || *flagFrom == "" || sid == "" ||
		auth == "" || *flagMsg == "" {
		log.Println("Don't have needed information")
		flag.Usage()
		os.Exit(-1)
	}

	messenger := msg.NewMessenger(sid, auth, *flagFrom)

	err := messenger.SendSMS(*flagTo, *flagMsg)

	if err != nil {
		log.Println("Error sending message: ", err)
		os.Exit(-1)
	}

	fmt.Println("Message sent")
}
