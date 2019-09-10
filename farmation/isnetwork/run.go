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

func bool2Float(in bool) float64 {
	if in {
		return 1
	}

	return 0
}

func input2Float(in isdata.InputState) float64 {
	if in == isdata.InputStateOn {
		return 1
	}

	return 0
}

// Run is the entry point for the isnetwork subsystem
func Run(in, out chan interface{}, configIn isdata.Config,
	stateIn isdata.State, portal string, debugPortal bool) {
	config := configIn
	state := stateIn
	errorCnt := 0

	manager := network.NewManager(10)

	sendSamplesAPI := api.NewSendSamples(portal, state.SerialNumber,
		time.Second*10, debugPortal)

	sendSamples := func(samples []data.Sample) error {
		if len(samples) <= 0 {
			return nil
		}

		err := sendSamplesAPI(samples)
		if err != nil {
			log.Println("Error sending data to portal: ", err)
			manager.Error()
			errorCnt++
			return err
		}
		manager.Success()

		return err
	}

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

		windowHigh, windowLow := config.CalculateFlowWindow()

		samples := []data.Sample{
			{
				Type:  "armed",
				Value: bool2Float(config.Arm),
			},
			{
				Type:  "flowRateTarget",
				Value: config.FlowRateTarget,
			},
			{
				Type:  "flowWindowLow",
				Value: windowLow,
			},
			{
				Type:  "flowWindowHigh",
				Value: windowHigh,
			},
			{
				Type:  "tankCapacity",
				Value: float64(config.TankCapacity),
			},
			{
				Type:  "inputWaterOn",
				Value: input2Float(state.InputWaterOn),
			},
			{
				Type:  "inputIrrigator",
				Value: input2Float(state.InputIrrigator),
			},
			{
				Type:  "inputInjector",
				Value: input2Float(state.InputInjector),
			},
			{
				Type:  "gpioRelayInjectorEn",
				Value: bool2Float(state.GpioRelayInjectorEn),
			},
			{
				Type:  "gpioShutdownEn",
				Value: bool2Float(state.GpioRelayShutdownEn),
			},
		}

		err := sendSamples(samples)
		if err == nil {
			initialDigitalDataSent = true
		}
	}

	manageTicker := time.NewTicker(time.Second * 10)
	sendPortal := time.NewTicker(time.Second * 5)

	if state.SerialNumber == "" {
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
			case isdata.Config:
				samples := []data.Sample{}
				if config.Arm != m.Arm {
					samples = append(samples,
						data.Sample{
							Type:  "armed",
							Value: bool2Float(m.Arm),
						})
				}

				if config.FlowRateTarget != m.FlowRateTarget {
					samples = append(samples,
						data.Sample{
							Type:  "flowRateTarget",
							Value: m.FlowRateTarget,
						})
				}

				windowHigh, windowLow := config.CalculateFlowWindow()
				windowHighN, windowLowN := config.CalculateFlowWindow()

				if windowLow != windowLowN {
					samples = append(samples,
						data.Sample{
							Type:  "flowWindowLow",
							Value: windowLowN,
						})
				}

				if windowHigh != windowHighN {
					samples = append(samples,
						data.Sample{
							Type:  "flowWindowHigh",
							Value: windowHighN,
						})
				}

				sendSamples(samples)

				config = m

			case isdata.State:
				samples := []data.Sample{}
				if state.InputWaterOn != m.InputWaterOn {
					samples = append(samples,
						data.Sample{
							Type:  "inputWaterOn",
							Value: input2Float(m.InputWaterOn),
						})
				}

				if state.InputIrrigator != m.InputIrrigator {
					samples = append(samples,
						data.Sample{
							Type:  "inputIrrigator",
							Value: input2Float(m.InputIrrigator),
						})
				}

				if state.InputInjector != m.InputInjector {
					samples = append(samples,
						data.Sample{
							Type:  "inputInjector",
							Value: input2Float(m.InputInjector),
						})
				}

				if state.GpioRelayInjectorEn != m.GpioRelayInjectorEn {
					samples = append(samples,
						data.Sample{
							Type:  "gpioRelayInjectorEn",
							Value: bool2Float(m.GpioRelayInjectorEn),
						})
				}

				if state.GpioRelayShutdownEn != m.GpioRelayShutdownEn {
					samples = append(samples,
						data.Sample{
							Type:  "gpioShutdownEn",
							Value: bool2Float(m.GpioRelayShutdownEn),
						})
				}

				sendSamples(samples)

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

			sendSamples(samples)
		}
	}
}
