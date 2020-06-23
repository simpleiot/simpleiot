package isnetwork

import (
	"log"
	"runtime"
	"strconv"
	"time"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/farmation/version"
	"github.com/simpleiot/simpleiot/network"
	"github.com/simpleiot/simpleiot/system"
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

func getConfigSamples(config *isdata.Config) []data.Sample {
	windowHigh, windowLow := config.CalculateFlowWindow()

	return []data.Sample{
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
	}
}

func getDigIoSamples(state *isdata.State) []data.Sample {
	return []data.Sample{
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
}

func getAnalogSamples(state *isdata.State) []data.Sample {
	return []data.Sample{
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
			Value: state.FlowAverager.GetAverage().Value,
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
}

// Run is the entry point for the isnetwork subsystem
func Run(in, out chan interface{}, configIn isdata.Config,
	stateIn isdata.State, portal string, debugPortal bool) {
	config := configIn
	state := stateIn
	errorCnt := 0

	// the GPIO init sometimes resets the modem, so give the modem
	// time to come on line before network init
	time.Sleep(10 * time.Second)

	manager := network.NewManager(10)

	sendSamplesAPI := api.NewSendSamples(portal, state.SerialNumber, time.Second*10, debugPortal)
	getCmdAPI := api.NewGetCmd(portal, state.SerialNumber, time.Second*10, debugPortal)
	setVersionAPI := api.NewSetVersion(portal, state.SerialNumber, time.Second*10, debugPortal)

	// the following function is used to stub out portal communication
	// during shutdown
	stopTalking := func() {
		sendSamplesAPI = func(samples []data.Sample) error {
			return nil
		}

		getCmdAPI = func() (data.DeviceCmd, error) {
			return data.DeviceCmd{}, nil
		}
	}

	whenChangedSamplesFilter := api.NewSampleFilter(0, 15*time.Minute)
	analogSamplesFilter := api.NewSampleFilter(30*time.Second, 15*time.Minute)

	versionSent := false

	sendSamples := func(samples []data.Sample) error {
		if len(samples) <= 0 {
			return nil
		}

		if state.SerialNumber == "" || portal == "" {
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

	var modem *network.Modem

	if runtime.GOOS == "windows" {
		manager.AddInterface(network.NewDummyInterface())
	} else {
		if runtime.GOARCH == "arm" {
			//manager.AddInterface(network.NewEthernet("eth0"))
			modem = network.NewModem(
				network.ModemConfig{
					ChatScript:    "bg96",
					AtCmdPortName: "/dev/ttyUSB2",
					Reset:         isio.ResetModem,
					Debug:         false,
					APN:           "vzwinternet",
				})
			manager.AddInterface(modem)
		} else {
			// various interfaces on development machines
			manager.AddInterface(network.NewEthernet("eno1"))
			manager.AddInterface(network.NewEthernet("wlp58s0"))
			manager.AddInterface(network.NewEthernet("enp39s0"))
		}
	}

	if modem != nil {
		modem.Enable(!config.ModemDisabled)
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

		samples := whenChangedSamplesFilter.Add(getConfigSamples(&config))
		samples = append(samples, analogSamplesFilter.Add(getDigIoSamples(&state))...)

		err := sendSamples(samples)
		if err == nil {
			initialDigitalDataSent = true
		}
	}

	manageTicker := time.NewTicker(time.Second * 10)
	pollPortal := time.NewTicker(time.Minute)

	if runtime.GOARCH != "arm" {
		// poll faster on development systems
		pollPortal = time.NewTicker(time.Second * 5)
	}

	var lastTimeSync time.Time

	const (
		displayInterval = time.Hour
		displayWait     = time.Minute
	)

	if state.SerialNumber == "" {
		log.Println("IS Serial is not set, not polling portal")
		pollPortal.Stop()
	}

	if portal == "" {
		log.Println("Portal URL is not set, not polling portal")
		pollPortal.Stop()
	}

	sendInitialDigitalData()

	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case isdata.Config:
				config = m
				samples := whenChangedSamplesFilter.Add(getConfigSamples(&config))
				sendSamples(samples)

			case isdata.State:
				state = m

				samples := whenChangedSamplesFilter.Add(getDigIoSamples(&state))
				sendSamples(samples)

				samples = analogSamplesFilter.Add(getAnalogSamples(&state))
				sendSamples(samples)

			case data.Sample:
				switch m.Type {
				case isdata.SampleTypeFaultFlowOff,
					isdata.SampleTypeFaultPresLow,
					isdata.SampleTypeFaultPresHigh,
					isdata.SampleTypeFaultShutdown,
					isdata.SampleTypeFaultNtFlowOff,
					isdata.SampleTypeFaultNtPresLow,
					isdata.SampleTypeFaultNtPresHigh:
					samples := []data.Sample{m}
					sendSamples(samples)
				}

			case isdata.UpdateModemDisabled:
				if modem != nil {
					modem.Enable(!bool(m))
				}

			case data.DeviceCmd:
				switch m.Cmd {
				case isdata.CmdFillTank, isdata.CmdSetTankLevel:
					samples := []data.Sample{
						{Type: "currentTankVolume",
							Value: state.CurrentTankVolume,
						},
					}
					sendSamples(samples)
				default:
					log.Println("isnetwork: unknown cmd: ", m.Cmd)
				}

			case isdata.ShutdownStart:
				if !interfaceStatus.Connected {
					log.Println("Sending poweroff state to portal")
					// send latest data to portal and then
					// power off cmd
					samples := getConfigSamples(&config)
					samples = append(samples, getDigIoSamples(&state)...)
					samples = append(samples, getAnalogSamples(&state)...)
					samples = append(samples, data.Sample{
						Type:  data.SampleTypeSysState,
						Value: float64(data.SysStatePowerOff),
					})
					sendSamples(samples)
				}
				// it is important no additional samples be sent
				// after poweroff system state or else the portal
				// will tag the GW as being online
				stopTalking()
				out <- isdata.Shutdown{}

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
				system.UpdateTimeFromNetwork()
				lastTimeSync = time.Now()
			}

			if modem != nil && !config.ModemDisabled {
				loc, err := modem.GetLocation()
				if err != nil {
					if err != network.ErrorModemNotDetected {
						log.Println("Error reading GPS: ", err)
					}
				} else {
					out <- loc
				}
			}

		case <-pollPortal.C:
			// look for commands from portal
			if !interfaceStatus.Connected {
				continue
			}

			sendInitialDigitalData()

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
		}
	}
}
