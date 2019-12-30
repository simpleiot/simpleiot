package app

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isapi"
	"github.com/simpleiot/simpleiot/farmation/iscontrol"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
	"github.com/simpleiot/simpleiot/farmation/isflow"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/farmation/islcd"
	"github.com/simpleiot/simpleiot/farmation/islog"
	"github.com/simpleiot/simpleiot/farmation/isnetwork"
	"github.com/simpleiot/simpleiot/farmation/ispressure"
	"github.com/simpleiot/simpleiot/farmation/isserial"
	"github.com/simpleiot/simpleiot/farmation/issim"
	"github.com/simpleiot/simpleiot/farmation/isui"
	"github.com/simpleiot/simpleiot/farmation/keypad"
	"github.com/simpleiot/simpleiot/file"
	"github.com/simpleiot/simpleiot/system"
)

// Params are used to configure the app
type Params struct {
	Sim          bool
	DataDir      string
	DebugState   bool
	DebugConfig  bool
	DebugModem   bool
	DebugPortal  bool
	PortalURL    string
	SerialNumber string
}

// Run is the entry point for the IS application
func Run(params Params) {
	log.Println("Starting Injectory Sentry app")

	if params.SerialNumber == "" {
		if runtime.GOARCH == "arm" {
			data, err := ioutil.ReadFile("/boot/serial-number")
			if err == nil {
				params.SerialNumber = string(data)
			} else {
				params.SerialNumber = "unknown"
			}
		} else {
			params.SerialNumber = "pcsim"
		}
	}

	log.Printf("App params: %+v\n", params)
	db, err := isdb.NewDb(params.DataDir)

	if err != nil {
		// FIXME this error  should display a message on screen to run recovery
		// process
		log.Fatal("Error opening db: ", err)
	}

	err = isdb.RunMigrations(db)

	if err != nil {
		log.Println("Error running migrations: ", err)
	}

	err = db.WriteSample(data.Sample{
		Time: time.Now(),
		Type: data.SampleTypeStartApp,
	})

	log.Println("Data directory: ", params.DataDir)

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
		log.Println("Error reading state, resetting: ", err)
		err := db.ResetDb()
		if err != nil {
			log.Println("Error resetting db: ", err)
		}
	}

	stateDirty = isdata.InitState(&state)
	state.SerialNumber = params.SerialNumber

	config.Init()

	// Check that the system timezone didn't get messed up
	zonePath, zone, _ := system.GetTimezone()

	if zone != config.Timezone || zonePath != "US" {
		system.SetTimezone("US", config.Timezone)
	}

	// incoming channel to mux
	appChan := make(chan interface{}, 1000)

	// outgoing channels to various other parts of the system
	keypadChan := make(chan interface{}, 100)
	uiChan := make(chan interface{}, 100)
	ioChan := make(chan interface{}, 100)
	cntrlChan := make(chan interface{}, 100)
	webChan := make(chan interface{}, 100)
	simChan := make(chan interface{}, 100)
	lcdChan := make(chan interface{}, 100)
	flowChan := make(chan interface{}, 100)
	logChan := make(chan interface{}, 1000)
	presChan := make(chan interface{}, 1000)
	serialChan := make(chan interface{}, 1000)
	networkChan := make(chan interface{}, 100)

	channels := []struct {
		name    string
		channel chan interface{}
	}{
		{"app", appChan},
		{"keypad", keypadChan},
		{"ui", uiChan},
		{"io", ioChan},
		{"cntrl", cntrlChan},
		{"web", webChan},
		{"sim", simChan},
		{"lcd", lcdChan},
		{"flow", flowChan},
		{"log", logChan},
		{"pres", presChan},
		{"serial", serialChan},
		{"network", networkChan},
	}

	// fire up subsystems
	go keypad.Run(keypadChan, appChan)
	go isui.Run(uiChan, appChan, config, db)
	go isio.Run(ioChan, appChan, config, state) // this is where io Run is called, w/ ioChan as in chan and appChan as out chan
	go iscontrol.Run(cntrlChan, appChan, config, state)
	go isapi.Server(webChan, appChan)
	go issim.Run(simChan, appChan)
	go islcd.Run(lcdChan, appChan)
	go isflow.Run(flowChan, appChan, params.Sim, config)
	go islog.Run(logChan, appChan, db)
	go ispressure.Run(presChan, appChan, config)
	go isserial.Run(serialChan, appChan, config)
	go isnetwork.Run(networkChan, appChan, config, state,
		params.PortalURL, params.DebugPortal)

	lastFillingWarning := time.Time{}

	saveConfig := func() {

		// Make sure all config items are within
		// reasonable bounds
		config.ApplyBounds()

		if params.DebugConfig {
			fmt.Printf("Config: %+v\n", config)
		}
		uiChan <- config
		flowChan <- config
		presChan <- config
		logChan <- config
		ioChan <- config
		presChan <- config
		cntrlChan <- config
		networkChan <- config
		err := db.WriteConfig(&config)
		if err != nil {
			log.Println("Error saving config: ", err)
		}
	}

	var lastStateSend time.Time
	var lastStateSendSlow time.Time

	saveState := func() {
		if params.DebugState {
			fmt.Println("State:")
			spew.Dump(state)
		}

		state.UpdateInputs(&config)

		stateDirty = true

		// pace the sending of states to various subsystems
		// so we don't overload things
		now := time.Now()
		if now.Sub(lastStateSend) > 200*time.Millisecond {
			uiChan <- state
			ioChan <- state
			cntrlChan <- state
			webChan <- state

			lastStateSend = now
		}

		if now.Sub(lastStateSendSlow) > 5*time.Second {
			networkChan <- state
			lastStateSendSlow = now
		}
	}

	saveStateTimer := time.NewTicker(time.Minute)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	updateNotFile := "/update-not"
	if runtime.GOARCH != "arm" {
		updateNotFile = "./update-not"
	}

	if !file.Exists(updateNotFile) {
		log.Println("System updated to: v", state.OSVersion)
		exec.Command("touch", updateNotFile).Run()
		state.DialogUpdate.Active = true
		state.DialogUpdate.Message = "System updated to v" + state.OSVersion.String()
		saveState()
	}

	//var lastPanelDialog time.Time
	//var panelChangeCount int

	/*
		newPanelType := func(def isdata.PanelDefinition) {
			if def.Type != state.PanelDefinition.Type {
				state.PanelDefinition = def
				saveState()
				panelChangeCount++

				if panelChangeCount < 5 || time.Since(lastPanelDialog) > 30*time.Minute {
					switch state.PanelDefinition.Type {
					case isdata.PanelTypeLindsay, isdata.PanelTypeStandardPump, isdata.PanelTypeStandardPivot:
						state.DialogInvalidPanel.Active = true
						state.DialogInvalidPanel.Message = "Panel detected\nType: " + state.PanelDefinition.Type.String()
					default:
						state.DialogInvalidPanel.Active = true
						state.DialogInvalidPanel.Message = "Unsupported panel detected\nType: " + state.PanelDefinition.Type.String()
						saveState()
					}
					lastPanelDialog = time.Now()
				}
			}
		}
	*/

	// Create an averager to calculate average flow since armed
	// Average is reset every time the system is armed
	flowAverager := data.NewSampleAverager(isdata.SampleTypeFlowWindowAvg)

	var lastVisionUnknownStateDisplay time.Time

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
			config.ManualRelayInj = isdata.RelayControlStateType(isdata.RelayControlStateAuto)
			config.ManualRelayAux = isdata.RelayControlStateType(isdata.RelayControlStateAuto)
			config.ManualRelayShutdown = isdata.RelayControlStateType(isdata.RelayControlStateAuto)
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

			case isdata.Flow:
				state.FlowPulseCount += int(m.Pulses)
				// log flow data (engineering purposes)
				if config.LogFlowData {
					logChan <- m
				}
				// update short moving average window used
				config.FlowAvgWindowShortUsed = m.ShortWin
				saveState()
				saveConfig()

			case data.Sample:
				switch m.Type {
				case isdata.SampleTypeFlowWindowAvg:

					// compute and update average flow rate in arming period
					if config.Arm {
						flowAverager.AddSample(m)
						state.AvgFlowRate = flowAverager.GetAverage().Value
					}

					// update flow rate
					state.FlowRate = m.Value
					saveState()

					// send data to logging goroutine to store in database
					logChan <- m

				case isdata.SampleTypeAmount:

					// update totals
					state.Total1 += m.Value
					state.Total2 += m.Value
					state.FieldStates[config.CurrentFieldIndex][config.CurrentProductIndex].Total += m.Value
					state.LifetimeTotal += m.Value

					// tank monitoring functions
					state.CurrentTankVolume -= m.Value
					if state.CurrentTankVolume < 0 {
						state.CurrentTankVolume = 0
					}

					saveState()

					// send data to logging goroutine to store in database
					logChan <- m

				case isdata.SampleTypePressure:

					// update pressure
					state.PressureAvg = m.Value
					state.PressureMin = m.Min
					state.PressureMax = m.Max

					saveState()

					// send data to logging goroutine to store in database
					logChan <- m

				case isdata.SampleTypePressureVRef:
					state.PressureVRef = m.Value
					saveState()

				case isdata.SampleTypePressureVSense:
					state.PressureVSense = m.Value
					saveState()

				case isdata.SampleTypeFaultFlowOff,
					isdata.SampleTypeFaultPresLow,
					isdata.SampleTypeFaultShutdown:
					state.FaultsActive = append(state.FaultsActive, m)
					saveState()

					// directly store fault sample
					db.WriteSample(m)

				case isdata.SampleTypeKey:
					// this is used for the simulator
					// convert from sample to key
					key := isdata.KeyFromString(m.ID)
					keyRel := isdata.KeyReleaseFromString(m.ID)
					uiChan <- key
					uiChan <- keyRel

				case isdata.SampleTypeSimFlowRate:
					flowChan <- m

				case isdata.SampleTypeSimPressure:
					presChan <- m

				case isdata.SampleTypeSimGpioDigInj:
					state.GpioDigitalInjector = m.Bool()
					saveState()

					// save to database for system logs
					db.WriteSample(data.Sample{
						Type:  isdata.SampleTypeInputInjector,
						Time:  time.Now(),
						Value: boolToSampleVal(m.Bool()),
					})

				case isdata.SampleTypeSimGpioDigIrg:
					state.GpioDigitalIrrigator = m.Bool()
					saveState()

					// save to database for system logs
					db.WriteSample(data.Sample{
						Type:  isdata.SampleTypeInputIrrigator,
						Time:  time.Now(),
						Value: boolToSampleVal(m.Bool()),
					})

				case isdata.SampleTypeSimGpioDigWaterOn:
					state.GpioDigitalWaterOn = m.Bool()
					saveState()

					// save to database for system logs
					db.WriteSample(data.Sample{
						Type:  isdata.SampleTypeInputWaterOn,
						Time:  time.Now(),
						Value: boolToSampleVal(m.Bool()),
					})

				case isdata.SampleTypeSimGpioDigIn:
					state.GpioDigitalIn = m.Bool()
					saveState()

				case isdata.SampleTypeSimArm:
					oldArm := config.Arm
					toggleArmOrOpenDialog(&config, &state)
					if config.Arm {
						flowAverager.ResetAverage()
						state.AvgFlowRateStart = time.Now()
					}
					saveConfig()
					saveState()

					if config.Arm != oldArm {
						// save to database for system logs
						db.WriteSample(data.Sample{
							Type:  isdata.SampleTypeArm,
							Time:  time.Now(),
							Value: boolToSampleVal(config.Arm),
						})
					}

				case isdata.SampleTypeSimPanelType:
					/*
						newPanelType(isdata.PanelDefinition{
							Type: isdata.PanelType(m.Value),
						})
					*/

				default:
					log.Println("Sample type not handled: ", m.Type)
				}

			case isdata.Key:
				switch m {
				case isdata.KeyArm:
					oldArm := config.Arm
					toggleArmOrOpenDialog(&config, &state)
					if config.Arm {
						flowAverager.ResetAverage()
						state.AvgFlowRateStart = time.Now()
					}
					saveConfig()
					saveState()

					if config.Arm != oldArm {
						// save to database for system logs
						db.WriteSample(data.Sample{
							Type:  isdata.SampleTypeArm,
							Time:  time.Now(),
							Value: boolToSampleVal(config.Arm),
						})
					}

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

			case isdata.UpdateEditedTime:
				config.EditedTime = time.Time(m)
				saveConfig()

			case isdata.UpdateTimezone:
				config.Timezone = string(m)
				saveConfig()

			case isdata.UpdatePulsesPerGallon:
				config.PulsesPerGallon = int(m)
				saveConfig()

			case isdata.UpdateFlowAvgWindow:
				config.FlowAvgWindow = int(m)
				saveConfig()

			case isdata.UpdateFlowAvgWindowLong:
				config.FlowAvgWindowLong = int(m)
				saveConfig()

			case isdata.UpdateFlowAvgPercDiff:
				config.FlowAvgPercDiff = int(m)
				saveConfig()

			case isdata.UpdatePressureSetting:
				config.PressureSetting = int(m)
				saveConfig()

			case isdata.UpdatePulseOutputK:
				config.PulseOutputK = int(m)
				saveConfig()

			case isdata.UpdatePulseOutputTestOn:
				config.PulseOutputTestOn = bool(m)
				saveConfig()

			case isdata.UpdatePulseOutputTestFlowRate:
				config.PulseOutputTestFlowRate = int(m)
				saveConfig()

			case isdata.UpdateSampleDuration:
				config.SampleDuration = int(m)
				saveConfig()

			case isdata.UpdateMaxNoPulseDuration:
				config.MaxNoPulseDuration = int(m)
				saveConfig()

			case isdata.UpdateAlarmRecognizeSec:
				config.AlarmRecognizeSec = float64(m)
				saveConfig()

			case isdata.UpdateLowWindowPerc:
				config.LowWindowPerc = float64(m)
				saveConfig()

			case isdata.UpdateHighWindowPerc:
				config.HighWindowPerc = float64(m)
				saveConfig()

			case isdata.UpdateManualLowAlarmGPH:
				config.ManualLowAlarmGPH = float64(m)
				saveConfig()

			case isdata.UpdateLowPresPerc:
				config.LowPresPerc = float64(m)
				saveConfig()

			case isdata.UpdateManualHighAlarmGPH:
				config.ManualHighAlarmGPH = float64(m)
				saveConfig()

			case isdata.UpdateDisarm:
				config.Arm = false
				saveConfig()

			case isdata.UpdateResetFlowPulseCount:
				state.FlowPulseCount = 0
				saveState()

			case isdata.UpdateResetTotal1:
				state.Total1 = 0
				saveState()

			case isdata.UpdateResetTotal2:
				state.Total2 = 0
				saveState()

			case isdata.UpdateResetCurrentProduct:
				state.FieldStates[config.CurrentFieldIndex][config.CurrentProductIndex].Total = 0
				saveState()

			case isdata.UpdateResetLifetime:
				state.LifetimeTotal = 0
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

			case isdata.RestartApp:
				state.DialogRestartApp.Active = true
				state.DialogRestartApp.Message = "The timezone was changed.\nThe IS will be restarted."

			case isdata.ExportData:
				if !state.DialogExport.Active {
					// we only want one export process running at a time
					logChan <- isdata.ExportData{}
				}
				state.DialogExport.Active = true
				state.DialogExport.Message = "Exporting data to USB Disk\nPlease Wait"

			case isdata.ExportAlreadyInProcess:
				state.DialogExport.Active = true
				state.DialogExport.Message = "Exporting already in process\nPlease Wait"

			case isdata.ExportDataFinished:
				state.DialogExport.Active = true
				state.DialogExport.Message = "Exporting data to USB Done\nPlease remove USB disk"

			case isdata.NoDiskPresent:
				state.DialogExport.Active = true
				state.DialogExport.Message = "Error: No USB disk present\nPlease insert USB drive\nand try again"

			case isdata.ErrWriteDisk:
				state.DialogExport.Active = true
				state.DialogExport.Message = "Error writing to USB disk"

			case isdata.UpdateTankAlertVolume:
				config.TankAlertVolume = int(m)
				saveConfig()

			case isdata.UpdateTankCapacity:
				config.TankCapacity = int(m)
				if config.TankCapacity > 9999 {
					config.TankCapacity = 9999
				}
				saveConfig()

			case isdata.UpdateTankFull:
				state.CurrentTankVolume = float64(config.TankCapacity)
				saveState()

			case isdata.UpdateCurrentTankVolume:
				state.CurrentTankVolume = float64(m)
				capacity := float64(config.TankCapacity)
				if state.CurrentTankVolume > capacity {
					state.CurrentTankVolume = capacity
				}
				saveState()

			case isdata.UpdateTankAlertEnable:
				config.TankAlertOn = bool(m)
				/*if !config.TankAlertOn {
					state.DialogApp.Active = true
					state.DialogApp.Message = "You just disabled tank\nlow-level alert"
				}*/
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

				// save to database for system logs
				db.WriteSample(data.Sample{
					Type:  isdata.SampleTypeInputInjector,
					Time:  time.Now(),
					Value: boolToSampleVal(bool(m)),
				})

			case isdata.UpdateGpioDigitalIrrigator:
				state.GpioDigitalIrrigator = bool(m)
				saveState()

				// save to database for system logs
				db.WriteSample(data.Sample{
					Type:  isdata.SampleTypeInputIrrigator,
					Time:  time.Now(),
					Value: boolToSampleVal(bool(m)),
				})

			case isdata.UpdateGpioDigitalWaterOn:
				state.GpioDigitalWaterOn = bool(m)
				saveState()

				// save to database for system logs
				db.WriteSample(data.Sample{
					Type:  isdata.SampleTypeInputWaterOn,
					Time:  time.Now(),
					Value: boolToSampleVal(bool(m)),
				})

			case isdata.UpdateGpioDigitalIn:
				state.GpioDigitalIn = bool(m)
				saveState()

			case isdata.UpdateManualRelayInj:
				config.ManualRelayInj = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdateManualRelayAux:
				config.ManualRelayAux = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdateManualRelayShutdown:
				config.ManualRelayShutdown = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdateManualRelayAll:
				config.ManualRelayInj = isdata.RelayControlStateType(m)
				config.ManualRelayAux = isdata.RelayControlStateType(m)
				config.ManualRelayShutdown = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdatePressureShutdownEnabled:
				config.PressureShutdownEnabled = !config.PressureShutdownEnabled
				if !config.PressureShutdownEnabled {
					state.DialogApp.Active = true
					state.DialogApp.Message = "You just disabled low-\npressure shutdown"
				}
				saveConfig()

			case isdata.UpdatePressureStartupLow:
				config.PressureStartupLow = int(m)
				saveConfig()

			case isdata.UpdateGpioRelayInjector:
				state.GpioRelayInjectorEn = bool(m)
				saveState()

			case isdata.UpdateGpioRelayShutdown:
				state.GpioRelayShutdownEn = bool(m)
				saveState()

			case isdata.UpdateGpioRelayAux:
				state.GpioRelayAuxEn = bool(m)
				saveState()

			case isdata.UpdateOperatingMode:
				config.OperatingMode = isdata.ISOperatingMode(m)
				if config.OperatingMode == isdata.ISOperatingModeMonitor {
					config.Arm = false // system can't be armed in monitor only mode
				}
				saveConfig()

			case isdata.UpdateUserPumpMode:
				config.UserPumpMode = isdata.UserPumpMode(m)
				saveConfig()

			case isdata.UpdateCurrentFieldIndex:
				config.CurrentFieldIndex = int(m)
				saveConfig()

			case isdata.UpdateCurrentProductIndex:
				config.CurrentProductIndex = int(m)
				saveConfig()

			case isdata.UpdateFlowStatus:
				state.FlowStatus = isdata.FlowStatus(m)
				saveState()

			case isdata.Pulse:
				logChan <- m

			case isdata.Reboot:
				state.DialogReboot.Active = true
				state.DialogReboot.Message = "Reboot started, please wait"
				if runtime.GOARCH != "arm" {
					log.Println("on development platform, not rebooting")
				} else {
					err := exec.Command("reboot").Run()
					if err != nil {
						fmt.Println("Error running reboot command")
					}
				}

			case isdata.UpdateLedRed:
				ioChan <- m
				state.GpioStatusLedRed = bool(m)
				saveState()

			case isdata.UpdateLedGreen:
				ioChan <- m
				state.GpioStatusLedGreen = bool(m)
				saveState()

			case isdata.LindsayStatusRegs:
				// if the Vision panel state is unknown, alert user
				if m.State.String() == "Unknown" &&
					time.Since(lastVisionUnknownStateDisplay) > 10*time.Minute {
					state.DialogUnknownVisionState.Message = "Vision panel state is unknown.\nOutputs shutting off."
					state.DialogUnknownVisionState.Active = true
					lastVisionUnknownStateDisplay = time.Now()
				}

				state.LindsayRegs = m
				state.LindsayLastUpdate = time.Now()
				saveState()

			case isdata.UpdateFaultActiveClearAll:
				state.FaultsActive = nil
				saveState()

			case isdata.UpdateDialogRestartAppClose:
				state.DialogRestartApp.Active = false
				saveState()
				exec.Command("/etc/init.d/isapp", "restart").Run()

			case isdata.UpdateDialogStateMachineMessage:
				state.DialogStateMachine.Message = string(m)
				state.DialogStateMachine.Active = true
				state.DialogStateMachine.Acknowledged = false
				saveState()

			case isdata.UpdateDialogStateMachineAck:
				state.DialogStateMachine.Acknowledged = true
				state.DialogStateMachine.Active = false
				saveState()

			case isdata.UpdateDialogStateMachineClose:
				state.DialogStateMachine.Active = false
				saveState()

			case isdata.UpdateDialogUpdateClose:
				state.DialogUpdate.Active = false
				saveState()

			case isdata.UpdateDialogArmClose:
				state.DialogArm.Active = false
				saveState()

			case isdata.UpdateDialogArmInputsClose:
				state.DialogArmInputs.Active = false
				saveState()

			case isdata.UpdateDialogArmReqClose:
				state.DialogArmReq.Active = false
				saveState()

			case isdata.UpdateDialogAppClose:
				state.DialogApp.Active = false
				saveState()

			case isdata.UpdateDialogExportClose:
				state.DialogExport.Active = false
				saveState()

			case isdata.UpdateDialogInvalidPanelClose:
				state.DialogInvalidPanel.Active = false
				saveState()

			case isdata.UpdateDialogUnknownVisionStateClose:
				state.DialogUnknownVisionState.Active = false
				saveState()

			case isdata.PanelDefinition:
				//newPanelType(m)

			case isdata.PanelType:
				config.PanelType = m
				saveConfig()

			case isdata.NetworkState:
				state.NetworkState = m
				saveState()

			default:
				// \r is required below to handle unknown keycode messages -- not sure why
				log.Printf("App Mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		}
	}
}

func toggleArmOrOpenDialog(config *isdata.Config, state *isdata.State) {
	if config.OperatingMode == isdata.ISOperatingModeMonitor {
		state.DialogArm.Active = true
		state.DialogArm.Message = "Error: Cannot arm in Monitor \nonly mode, please switch \nto Monitor and Shutdown \nmode"
		return
	}
	if config.UserPumpMode == isdata.UserPumpModeNotSet {
		state.DialogArmInputs.Active = true
		state.DialogArmInputs.Message = "Error: Injector Command \nInput not selected, please \nselect before arming"
		return
	}

	if !config.Arm { // if the arm switch will be turned on
		if isdata.AllArmReqMet(config, state) {
			config.Arm = !config.Arm
			config.FlowRateTarget = state.FlowRate // set target flow rate to current
			config.PressureShutdownLow = state.PressureMin - state.PressureMin*config.LowPresPerc/100
		} else {
			state.DialogArmReq.Active = true
		}
	} else {
		config.Arm = !config.Arm
	}
}

func boolToSampleVal(b bool) float64 {
	if b {
		return float64(1)
	}
	return float64(0)
}
