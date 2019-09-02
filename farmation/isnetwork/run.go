package isnetwork

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/network"
)

// Run is the entry point for the isnetwork subsystem
func Run(in, out chan interface{}) {
	var modemManager *network.ModemManager
	if runtime.GOARCH == "arm" {
		port, err := isio.OpenSerialModem()
		if err != nil {
			fmt.Println("Error opening modem port: ", err)
		} else {
			modem := network.NewModem(port, "hologram", false)
			modemManager = network.NewModemManager(modem)
		}
	}

	modemPoll := time.NewTicker(time.Second * 10)
	modemState := network.ModemState{}

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			default:
				log.Printf("isnet mux: unhandled message of type %T: %+v\r\n", m, m)
			}
		case <-modemPoll.C:
			if modemManager == nil {
				continue
			}

			s, err := modemManager.GetState()
			if err != nil {
				log.Println("Error getting modem state: ", err)
				continue
			}

			if s != modemState {
				out <- s
			}
		}
	}
}
