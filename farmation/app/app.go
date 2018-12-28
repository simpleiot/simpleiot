package app

import (
	"log"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isapi"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/farmation/isui"
	"github.com/simpleiot/simpleiot/farmation/keypad"
)

// Run is the entry point for the IS application
func Run() {
	db, err := isdb.NewDb("./")

	if err != nil {
		log.Fatal("Error opening db: ", err)
	}

	config := isdata.ISConfig{}

	err = db.ReadConfig(&config)

	if err != nil {
		log.Println("Error reading config: ", err)
	}

	// incoming channel to mux
	appChan := make(chan interface{}, 100)

	// outgoing channels to various other parts of the system
	keypadChan := make(chan interface{}, 100)
	uiChan := make(chan interface{}, 100)
	ioChan := make(chan interface{}, 100)
	webChan := make(chan interface{}, 100)

	channels := []struct {
		name    string
		channel chan interface{}
	}{
		{"app", appChan},
		{"keypad", keypadChan},
		{"ui", uiChan},
		{"io", ioChan},
		{"web", webChan},
	}

	// fire up subsystems
	go keypad.Run(keypadChan, appChan)
	go isui.Run(uiChan, appChan)
	go isio.Run(ioChan, appChan)
	go isapi.Server(webChan, appChan)

	lastFillingWarning := time.Time{}

	for {
		// max sure queues between subsystems are not full
		for _, c := range channels {
			if len(c.channel) >= 99 {
				log.Println("Warning channel full: ", c.name, len(c.channel))
				log.Println("dropping entry: ", c.name, len(c.channel))
				<-c.channel
			} else if len(c.channel) > 30 &&
				time.Now().Sub(lastFillingWarning) > time.Minute {
				log.Println("Warning channel is filling: ", c.name, len(c.channel))
				lastFillingWarning = time.Now()
			}
		}
		select {
		case m := <-appChan:
			switch m := m.(type) {
			case isdata.LcdPixel:
				webChan <- m

			default:
				log.Printf("Mux: unhandled message of type %T: %+v\n", m, m)

			}
		}
	}

}
