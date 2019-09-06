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

	sendInitialPortalData := func() {
		samples := []data.Sample{
			{
				Type:  "inputWaterOn",
				Value: 0,
			},
			{
				Type:  "inputIrrigator",
				Value: 0,
			},
			{
				Type:  "inputInjector",
				Value: 0,
			},
			{
				Type:  "gpioRelayInjectorEn",
				Value: 0,
			},
			{
				Type:  "gpioShutdownEn",
				Value: 0,
			},
		}

		err := sendSamples(sn, samples)
		if err != nil {
			log.Println("Error sending data to portal: ", err)
		}
	}

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

	sendInitialPortalData()

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.State:
				samples := []data.Sample{}
				if state.InputWaterOn != m.InputWaterOn {
					v := 0.0
					if m.InputWaterOn == isdata.InputStateOn {
						v = 1.0
					}
					samples = append(samples,
						data.Sample{
							Type:  "inputWaterOn",
							Value: v,
						})
				}

				if state.InputIrrigator != m.InputIrrigator {
					v := 0.0
					if m.InputIrrigator == isdata.InputStateOn {
						v = 1.0
					}
					samples = append(samples,
						data.Sample{
							Type:  "inputIrrigator",
							Value: v,
						})
				}

				if state.InputInjector != m.InputInjector {
					v := 0.0
					if m.InputInjector == isdata.InputStateOn {
						v = 1.0
					}
					samples = append(samples,
						data.Sample{
							Type:  "inputInjector",
							Value: v,
						})
				}

				if state.GpioRelayInjectorEn != m.GpioRelayInjectorEn {
					v := 0.0
					if m.GpioRelayInjectorEn {
						v = 1.0
					}
					samples = append(samples,
						data.Sample{
							Type:  "gpioRelayInjectorEn",
							Value: v,
						})
				}

				if state.GpioRelayShutdownEn != m.GpioRelayShutdownEn {
					v := 0.0
					if m.GpioRelayShutdownEn {
						v = 1.0
					}
					samples = append(samples,
						data.Sample{
							Type:  "gpioShutdownEn",
							Value: v,
						})
				}

				if len(samples) > 0 {
					err := sendSamples(sn, samples)
					if err != nil {
						log.Println("Error sending data to portal: ", err)
					}
				}

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
			samples := []data.Sample{
				{
					Type:  "flowRate",
					Value: state.FlowRate,
				},
				{
					Type:  "currentTankVolume",
					Value: state.CurrentTankVolume,
				},
				{
					Type:  "avgFlowRate",
					Value: state.AvgFlowRate,
				},
				{
					Type:  "PressureMin",
					Value: state.PressureMin,
				},
				{
					Type:  "PressureMax",
					Value: state.PressureMax,
				},
			}

			err := sendSamples(sn, samples)
			if err != nil {
				log.Println("Error sending data to portal: ", err)
			}
		}
	}
}
