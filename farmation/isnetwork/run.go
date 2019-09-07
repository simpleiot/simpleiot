package isnetwork

import (
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
	sendSamples := api.NewSendSamples(portal, sn, time.Second*10, debugPortal)
	errorCnt := 0

	manager := network.NewManager(10)
	if runtime.GOOS == "windows" {
		manager.AddInterface(network.NewDummyInterface())
	} else {
		if runtime.GOARCH == "arm" {
			manager.AddInterface(network.NewEthernet("eth0"))
			manager.AddInterface(network.NewModem("bg96",
				"/dev/ttyUSB2", isio.ResetModem, false))
		} else {
			// various interfaces on development machines
			manager.AddInterface(network.NewEthernet("eno1"))
			manager.AddInterface(network.NewEthernet("wlp58s0"))
		}
	}

	networkState, interfaceStatus := manager.Run()
	_ = networkState

	initialDigitalDataSent := false
	sendInitialDigitalData := func() {
		if !interfaceStatus.Connected || initialDigitalDataSent {
			return
		}

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

		err := sendSamples(samples)
		if err != nil {
			log.Println("Error sending data to portal: ", err)
			manager.Error()
			errorCnt++
			return
		}
		manager.Success()

		initialDigitalDataSent = true
	}

	manageTicker := time.NewTicker(time.Second * 10)
	sendPortal := time.NewTicker(time.Second * 5)

	if sn == "" {
		log.Println("IS Serial is not set, not sending data to portal")
		sendPortal.Stop()
	}

	if portal == "" {
		log.Println("Portal URL is not set, not sending data to portal")
		sendPortal.Stop()
	}

	sendInitialDigitalData()

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
					err := sendSamples(samples)
					if err != nil {
						log.Println("Error sending data to portal: ", err)
						manager.Error()
						errorCnt++
					} else {
						manager.Success()
					}
				}

				state = m
			default:
				log.Printf("isnet mux: unhandled message of type %T: %+v\r\n", m, m)
			}
		case <-manageTicker.C:
			networkState, interfaceStatus = manager.Run()
			out <- isdata.NetworkState{
				Description:     manager.Desc(),
				InterfaceStatus: interfaceStatus,
				ErrorCnt:        errorCnt,
			}

		case <-sendPortal.C:
			sendInitialDigitalData()
			if !interfaceStatus.Connected {
				continue
			}
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
					Type:  "pressureMin",
					Value: state.PressureMin,
				},
				{
					Type:  "pressureMax",
					Value: state.PressureMax,
				},
			}

			err := sendSamples(samples)
			if err != nil {
				log.Println("Error sending data to portal: ", err)
				manager.Error()
				errorCnt++
			} else {
				manager.Success()
			}
		}
	}
}
