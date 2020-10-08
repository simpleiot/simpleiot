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
	flagMsg := flag.String("msg", "", "message to send")

	flag.Parse()

	sid := os.Getenv("TWILIO_SID")
	auth := os.Getenv("TWILIO_AUTH_TOKEN")
	from := os.Getenv("TWILIO_FROM")

	if *flagTo == "" || sid == "" ||
		auth == "" || *flagMsg == "" || from == "" {
		log.Println("Don't have needed information")
		flag.Usage()
		os.Exit(-1)
	}

	messenger := msg.NewMessenger(sid, auth, from)

	err := messenger.SendSMS(*flagTo, *flagMsg)

	if err != nil {
		log.Println("Error sending message: ", err)
		os.Exit(-1)
	}

	fmt.Println("Message sent")
}
