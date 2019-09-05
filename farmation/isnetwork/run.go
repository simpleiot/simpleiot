package isnetwork

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/network"
)

// Run is the entry point for the isnetwork subsystem
func Run(in, out chan interface{}, stateIn isdata.State, sn, portal string,
	debugPortal bool) {
	state := stateIn
	sendSamples := api.NewSendSamples(portal, debugPortal)
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
	sendPortal := time.NewTicker(time.Second * 5)
	modemState := network.ModemState{}

	if sn == "" {
		log.Println("IS Serial is not set, not sending data to portal")
		sendPortal.Stop()
	}

	if portal == "" {
		log.Println("Portal URL is not set, not sending data to portal")
		sendPortal.Stop()
	}

	if modemManager != nil {
		s, err := modemManager.GetState()
		if err != nil {
			log.Println("Error getting modem state: ", err)
		} else {
			if s != modemState {
				out <- s
			}
		}
	}

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				state = m
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
		case <-sendPortal.C:
			now := time.Now()
			samples := []data.Sample{
				{
					Type:  "flowRate",
					Value: state.FlowRate,
					Time:  now,
				},
			}

			fmt.Println("CLIFF: sending samples")

			err := sendSamples(sn, samples)
			if err != nil {
				log.Println("Error sending data to portal: ", err)
			}
		}
	}
}
