package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isapi"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
	"github.com/simpleiot/simpleiot/farmation/isflow"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/farmation/islcd"
	"github.com/simpleiot/simpleiot/farmation/islog"
	"github.com/simpleiot/simpleiot/farmation/ispressure"
	"github.com/simpleiot/simpleiot/farmation/issim"
	"github.com/simpleiot/simpleiot/farmation/isui"
	"github.com/simpleiot/simpleiot/farmation/keypad"
	"github.com/simpleiot/simpleiot/file"
)

// Run is the entry point for the IS application
func Run(sim bool, debugState bool, debugConfig bool, dataDir string) {
	db, err := isdb.NewDb(dataDir)

	if err != nil {
		log.Fatal("Error opening db: ", err)
	}

	log.Println("Data directory: ", dataDir)

	isio.GpioInit()

	// Config and state are *ONLY* modified in app.go
	// Config is anything modified by the user
	// State is anything modified by the program
	config := isdata.Config{}
	state := isdata.State{}
	stateDirty := false

	err = db.ReadConfig(&config)

	if err != nil {
		log.Println("Error reading config, resetting: ", err)
		err := db.ResetDb()
		if err != nil {
			log.Println("Error resetting db: ", err)
		}
	}

	err = db.ReadState(&state)

	if err != nil {
		log.Println("Error reading state: ", err)
	}

	stateDirty = isdata.InitState(&state)
	config.Init()

	// incoming channel to mux
	appChan := make(chan interface{}, 1000)

	// outgoing channels to various other parts of the system
	keypadChan := make(chan interface{}, 100)
	uiChan := make(chan interface{}, 100)
	ioChan := make(chan interface{}, 100)
	webChan := make(chan interface{}, 100)
	simChan := make(chan interface{}, 100)
	lcdChan := make(chan interface{}, 100)
	flowChan := make(chan interface{}, 100)
	logChan := make(chan interface{}, 1000)
	presChan := make(chan interface{}, 1000)

	channels := []struct {
		name    string
		channel chan interface{}
	}{
		{"app", appChan},
		{"keypad", keypadChan},
		{"ui", uiChan},
		{"io", ioChan},
		{"web", webChan},
		{"sim", simChan},
		{"lcd", lcdChan},
		{"flow", flowChan},
		{"log", logChan},
		{"pres", presChan},
	}

	// fire up subsystems
	go keypad.Run(keypadChan, appChan)
	go isui.Run(uiChan, appChan, config)
	go isio.Run(ioChan, appChan, config, state) // this is where io Run is called, w/ ioChan as in chan and appChan as out chan
	go isapi.Server(webChan, appChan)
	go issim.Run(simChan, appChan)
	go islcd.Run(lcdChan, appChan)
	go isflow.Run(flowChan, appChan, sim)
	go islog.Run(logChan, appChan)
	go ispressure.Run(presChan, appChan, config)

	lastFillingWarning := time.Time{}

	saveConfig := func() {
		if debugConfig {
			fmt.Printf("Config: %+v\n", config)
		}
		uiChan <- config
		flowChan <- config
		presChan <- config
		logChan <- config
		ioChan <- config
		presChan <- config
		err := db.WriteConfig(&config)
		if err != nil {
			log.Println("Error saving config: ", err)
		}
	}
	// TODO use this structure to save state
	saveState := func() {
		if debugState {
			fmt.Printf("State: %+v\n", state)
		}
		stateDirty = true
		uiChan <- state
		ioChan <- state
	}

	saveStateTimer := time.NewTicker(time.Minute)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	for {
		// max sure queues between subsystems are not full
		for _, c := range channels {
			if len(c.channel) >= cap(c.channel)-1 {
				log.Println("Warning channel full: ", c.name, len(c.channel))
				e := <-c.channel
				log.Printf("dropping entry of type: %T\n", e)
			} else if len(c.channel) > 30 &&
				time.Now().Sub(lastFillingWarning) > time.Minute {
				log.Println("Warning channel is filling: ", c.name, len(c.channel))
				lastFillingWarning = time.Now()
			}
		}
		select {
		case s := <-sigChan:
			fmt.Println("Received signal: ", s)
			config.ManualRelayInj = isdata.RelayControlStateType(isdata.RelayControlAutoStateType)
			config.ManualRelayAux = isdata.RelayControlStateType(isdata.RelayControlAutoStateType)
			config.ManualRelayShutdown = isdata.RelayControlStateType(isdata.RelayControlAutoStateType)
			saveConfig()
			db.WriteState(&state)
			db.WriteConfig(&config)
			fmt.Println("state and config saved, SEE YA!")
			os.Exit(0)

		case <-saveStateTimer.C:
			if stateDirty {
				db.WriteState(&state)
				stateDirty = false
			}
		case m := <-appChan:
			switch m := m.(type) {
			case isdata.LcdPixel:
				webChan <- m

			case isdata.LcdBlt:
				webChan <- m
				lcdChan <- m

			case isdata.LcdBltSolid:
				webChan <- m

			case data.Sample:
				switch m.Type {
				case isdata.SampleTypePressure:
					if config.LogPressureData {
						logChan <- m
					}
				case isdata.SampleTypePressureMin:
					state.PressureMin = m.Value
					saveState()
				case isdata.SampleTypePressureMax:
					state.PressureMax = m.Value
					saveState()
				case isdata.SampleTypePressureAvg:
					state.PressureAvg = m.Value
					saveState()
				case isdata.SampleTypePressureVRef:
					state.PressureVRef = m.Value
					saveState()
				case isdata.SampleTypePressureVSense:
					state.PressureVSense = m.Value
					saveState()
				case isdata.SampleTypeFlowRate:
					state.ProcessSample(m)
					uiChan <- state
				case isdata.SampleTypeKey:
					// convert from sample to key
					key := isdata.KeyFromString(m.ID)
					/*switch key {
					case isdata.KeyUp: // map down key to arm
						config.Arm = !config.Arm
						saveConfig()
					default:*/
					uiChan <- key
					//}
				default:
					log.Println("Sample type not handled: ", m.Type)
				}
			case isdata.Key:
				switch m {
				case isdata.KeyArm:
					// toggle the arm switch
					config.Arm = !config.Arm
					saveConfig()
				default:
					// send to ui to handle
					uiChan <- m
				}

			case isdata.UpdateFieldName:
				config.FieldConfigs[m.Index].Description = m.Name
				saveConfig()

			case isdata.UpdateProductName:
				config.ProductConfigs[m.Index].Description = m.Name
				saveConfig()

			case isdata.UpdateDevName:
				config.DeviceName = string(m)
				saveConfig()

			case isdata.UpdatePulsesPerGallon:
				config.PulsesPerGallon = int(m)
				saveConfig()

			case isdata.UpdatePressureSetting:
				config.PressureSetting = int(m)
				saveConfig()

			case isdata.Flow:
				state.FlowRate = m.RateAvg
				state.Total1 += m.Amount
				state.Total2 += m.Amount
				state.FlowPulseCount += m.Pulses
				if config.LogFlowData {
					logChan <- m
				}
				saveState()

			case isdata.UpdateResetFlowPulseCount:
				state.FlowPulseCount = 0
				saveState()

			case isdata.UpdateResetTotal1:
				state.Total1 = 0
				saveState()

			case isdata.UpdateResetTotal2:
				state.Total2 = 0
				saveState()

			case isdata.UpdateLogPulseEnable:
				config.LogPulseData = bool(m)
				saveConfig()
				if !m {
					err := file.SyncDisks()
					if err != nil {
						log.Println("sync error: ", err)
					}
				}

			case isdata.UpdateLogFlowEnable:
				config.LogFlowData = bool(m)
				saveConfig()
				if !m {
					err := file.SyncDisks()
					if err != nil {
						log.Println("sync error: ", err)
					}
				}

			case isdata.UpdateLogPressureEnable:
				config.LogPressureData = bool(m)
				saveConfig()
				if !m {
					err := file.SyncDisks()
					if err != nil {
						log.Println("sync error: ", err)
					}
				}

			case isdata.UpdateTankAlertEnable:
				config.TankAlertOn = bool(m)
				saveConfig()
				if !m {
					err := file.SyncDisks()
					if err != nil {
						log.Println("sync error: ", err)
					}
				}

			case isdata.UpdateGpioDigitalInjector:
				state.GpioDigitalInjector = bool(m)
				saveState()

			case isdata.UpdateGpioDigitalIrrigator:
				state.GpioDigitalIrrigator = bool(m)
				saveState()

			case isdata.UpdateGpioDigitalWaterOn:
				state.GpioDigitalWaterOn = bool(m)
				saveState()

			case isdata.UpdateGpioDigitalIn:
				state.GpioDigitalIn = bool(m)
				saveState()

			case isdata.UpdateManualRelayInj:
				//fmt.Println(config.ManualRelayInj)
				config.ManualRelayInj = isdata.RelayControlStateType(m)
				//fmt.Println(isdata.RelayControlStateType(m))
				saveConfig()

			case isdata.UpdateManualRelayAux:
				//fmt.Println(config.ManualRelayAux)
				config.ManualRelayAux = isdata.RelayControlStateType(m)
				//fmt.Println(isdata.RelayControlStateType(m))
				saveConfig()

			case isdata.UpdateManualRelayShutdown:
				//fmt.Println(config.ManualRelayShutdown)
				config.ManualRelayShutdown = isdata.RelayControlStateType(m)
				//fmt.Println(isdata.RelayControlStateType(m))
				saveConfig()

			case isdata.UpdateManualRelayAll:
				config.ManualRelayInj = isdata.RelayControlStateType(m)
				config.ManualRelayAux = isdata.RelayControlStateType(m)
				config.ManualRelayShutdown = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdateOperatingMode:
				config.OperatingMode = isdata.ISOperatingMode(m)
				saveConfig()

			case isdata.UpdateCurrentFieldIndex:
				config.CurrentFieldIndex = int(m)
				saveConfig()

			case isdata.UpdateCurrentProductIndex:
				config.CurrentProductIndex = int(m)
				saveConfig()

			case isdata.Pulse:
				logChan <- m

			case isdata.Reboot:
				if runtime.GOARCH != "arm" {
					log.Println("on development platform, not rebooting")
				} else {
					err := exec.Command("reboot").Run()
					if err != nil {
						fmt.Println("Error running reboot command")
					}
				}

			default:
				// \r is required below to handle unknown keycode messages -- not sure why
				log.Printf("App Mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		}
	}
}
