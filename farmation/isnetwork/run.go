package isnetwork

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/beevik/ntp"
	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/farmation/version"
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

	sendSamplesAPI := api.NewSendSamples(portal, state.SerialNumber, time.Second*10, debugPortal)
	getCmdAPI := api.NewGetCmd(portal, state.SerialNumber, time.Second*10, debugPortal)
	setVersionAPI := api.NewSetVersion(portal, state.SerialNumber, time.Second*10, debugPortal)

	versionSent := false

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
			//manager.AddInterface(network.NewEthernet("eth0"))
			manager.AddInterface(network.NewModem(
				network.ModemConfig{
					ChatScript:    "bg96",
					AtCmdPortName: "/dev/ttyUSB2",
					Reset:         isio.ResetModem,
					Debug:         false,
					APN:           "vzwinternet",
				}))
		} else {
			// various interfaces on development machines
			manager.AddInterface(network.NewEthernet("eno1"))
			manager.AddInterface(network.NewEthernet("wlp58s0"))
		}
	}

	networkState, interfaceConfig, interfaceStatus := manager.Run()
	_ = networkState

	if interfaceConfig.Apn != "" {
		out <- interfaceConfig
		log.Printf("Network Interface Config: %+v\n", interfaceConfig)
	}

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
	sendPortal := time.NewTicker(time.Minute * 10)
	pollPortal := time.NewTicker(time.Minute)

	if runtime.GOARCH != "arm" {
		// poll faster on development systems
		pollPortal = time.NewTicker(time.Second * 5)
	}

	var lastTimeSync, lastLostConnectionAlert time.Time

	if state.SerialNumber == "" {
		log.Println("IS Serial is not set, not sending data to portal")
		sendPortal.Stop()
		pollPortal.Stop()
	}

	if portal == "" {
		log.Println("Portal URL is not set, not sending data to portal")
		sendPortal.Stop()
		pollPortal.Stop()
	}

	sendInitialDigitalData()

	lastReportedFlow := 0.0
	lastReportedTankVolume := 0.0

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

				if math.Abs(lastReportedFlow-m.FlowRate) > 5 {
					samples = append(samples,
						data.Sample{
							Type:  "flowRate",
							Value: m.FlowRate,
						})

					lastReportedFlow = m.FlowRate

				}

				if math.Abs(lastReportedTankVolume-m.CurrentTankVolume) > 10 {
					samples = append(samples,
						data.Sample{
							Type:  "currentTankVolume",
							Value: m.CurrentTankVolume,
						})

					lastReportedTankVolume = m.CurrentTankVolume
				}

				sendSamples(samples)

				state = m

			case data.Sample:
				switch m.Type {
				case isdata.SampleTypeInputInjector,
					isdata.SampleTypeInputIrrigator,
					isdata.SampleTypeInputWaterOn,
					isdata.SampleTypeArm,
					isdata.SampleTypeFaultFlowOff,
					isdata.SampleTypeFaultPresLow,
					isdata.SampleTypeFaultPresHigh,
					isdata.SampleTypeFaultShutdown:
					fmt.Println("COLLIN, network thread -- got sample of type: ", m.Type)
					samples := []data.Sample{m}
					sendSamples(samples)
				}

			case isdata.NoNetworkDialogDisplayed:
				lastLostConnectionAlert = time.Now()

			default:
				log.Printf("isnet mux: unhandled message of type %T: %+v\r\n", m, m)
			}

		case <-manageTicker.C:
			networkState, interfaceConfig, interfaceStatus = manager.Run()

			if interfaceConfig.Apn != "" {
				out <- interfaceConfig
			}

			out <- isdata.NetworkState{
				Description:     manager.Desc(),
				InterfaceStatus: interfaceStatus,
				ErrorCnt:        errorCnt,
			}

			// Time syncing through network
			if interfaceStatus.Connected &&
				(lastTimeSync.IsZero() || time.Since(lastTimeSync) >= time.Hour) {
				updateTime()
				lastTimeSync = time.Now()
			}

			// If the system is in Monitor and Notify mode, alert
			// of lost network connection
			if config.OperatingMode == isdata.ISOperatingModeMonitorAndNotify &&
				!interfaceStatus.Connected &&
				(lastLostConnectionAlert.IsZero() ||
					time.Since(lastLostConnectionAlert) >= time.Hour) {
				out <- isdata.NoNetworkConnection{}
				// This is also happening when the NoNetworkDialogDisplayed message
				// comes in, but doing it here just to make sure it doesn't get
				// stuck in a loop of displaying the dialog
				lastLostConnectionAlert = time.Now()
			}

		case <-pollPortal.C:
			// look for commands from portal
			if !interfaceStatus.Connected {
				continue
			}

			cmd, err := getCmdAPI()

			if err != nil {
				log.Println("Error getting command from portal: ", err)
				continue
			}

			if cmd.Cmd != "" {
				out <- cmd
			}

			if !versionSent {
				err := setVersionAPI(data.DeviceVersion{
					OS:  state.OSVersion.String(),
					App: version.AppVersion,
					HW:  strconv.Itoa(state.HWVersion),
				})

				if err != nil {
					log.Println("Error sending version info to portal: ", err)
					continue
				}

				versionSent = true
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

			if err := sendSamples(samples); err != nil {
				log.Println("Error sending samples to server: ", err)
			} else {
				lastReportedFlow = state.FlowRate
				lastReportedTankVolume = state.CurrentTankVolume
			}
		}
	}
}

func updateTime() (err error) {

	current, err := ntp.Time("0.pool.ntp.org")
	if err != nil {
		log.Println("Error fetching time from ntp.org: ", err)
		return err
	}
	log.Println("Time: ", current)

	tv := syscall.NsecToTimeval(current.UnixNano())
	err = syscall.Settimeofday(&tv)
	if err != nil {
		log.Println("Error synchronizing system clock: ", err)
		return err
	}

	return nil
}
