package app

import (
	"fmt"
	"image"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isapi"
	"github.com/simpleiot/simpleiot/farmation/iscontrol"
	"github.com/simpleiot/simpleiot/farmation/isdata"
	"github.com/simpleiot/simpleiot/farmation/isdb"
	"github.com/simpleiot/simpleiot/farmation/isflow"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/farmation/islcd"
	"github.com/simpleiot/simpleiot/farmation/islog"
	"github.com/simpleiot/simpleiot/farmation/isnetwork"
	"github.com/simpleiot/simpleiot/farmation/ispower"
	"github.com/simpleiot/simpleiot/farmation/ispressure"
	"github.com/simpleiot/simpleiot/farmation/issim"
	"github.com/simpleiot/simpleiot/farmation/isui"
	"github.com/simpleiot/simpleiot/farmation/isupdate"
	"github.com/simpleiot/simpleiot/farmation/keypad"
	"github.com/simpleiot/simpleiot/farmation/version"
	"github.com/simpleiot/simpleiot/file"
	"github.com/simpleiot/simpleiot/network"
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
	ViewMsg      bool
	ReadVcap     bool
	WebUI        bool
}

// Run is the entry point for the IS application
func Run(params Params) {
	log.Println("Starting Injectory Sentry app")

	if params.ReadVcap {
		v, err := isio.ReadVcap()
		if err != nil {
			log.Println("Error reading vcap: ", err)
			os.Exit(-1)
		}
		log.Println("Vcap: ", v)
		os.Exit(0)
	}

	if params.SerialNumber == "" {
		if runtime.GOARCH == "arm" {
			sn, err := isdb.ReadSerialNumber()
			if err == nil {
				params.SerialNumber = sn
			} else {
				params.SerialNumber = "unknown"
			}
		} else {
			params.SerialNumber = "pcsim"
		}
	}

	log.Printf("App params: %+v\n", params)

	err := isio.GpioInit()
	if err != nil {
		log.Println("Error initializing GPIO: ", err)
		os.Exit(-1)
	}

	config := isdata.Config{}
	state := isdata.State{}

	dbConfig, dbState, dbData, err := isdb.DbInit(params.DataDir, &config, &state)

	if err != nil {
		log.Println("Fatal error opening db: ", err)

		// Fixme should display error message on display here for user
		lcd, err := islcd.NewLcd()

		if err != nil {
			log.Println("Error opening LCD", err)
			os.Exit(-1)
		}

		err = lcd.Init()
		if err != nil {
			log.Println("Error initializing LCD: ", err)
			os.Exit(-1)
		}

		img := image.NewRGBA(image.Rect(0, 0, 128, 64))
		isui.Clear(img)

		isui.DrawTxtCentered(img, "Fatal data error", 64, 20, tightpixel15.Font)
		isui.DrawTxtCentered(img, "Please contact support", 64, 40, tightpixel15.Font)
		lcd.Write(isui.ImageToBlt(0, 0, img, false).Data)

		os.Exit(-1)
	}

	err = dbData.WriteSample(data.Sample{
		Time: time.Now(),
		Type: data.SampleTypeStartApp,
	})

	// Config and state are *ONLY* modified in app.go
	// Config is anything modified by the user
	// State is anything modified by the program

	stateDirty := isdata.InitState(&state)
	state.SerialNumber = params.SerialNumber
	state.ViewMsg = params.ViewMsg
	state.HWVersion = isio.GetHwID()

	log.Println("HWVersion: ", state.HWVersion)
	log.Println("OSVersion: ", state.OSVersion)
	log.Println("AppVersion: ", version.AppVersion)

	config.Init(&state)

	// incoming channel to mux
	appChan := make(chan interface{}, 1000)

	// outgoing channels to various other parts of the system
	keypadChan := make(chan interface{}, 100)
	uiChan := make(chan interface{}, 1000)
	ioChan := make(chan interface{}, 100)
	cntrlChan := make(chan interface{}, 100)
	webChan := make(chan interface{}, 100)
	simChan := make(chan interface{}, 100)
	lcdChan := make(chan interface{}, 100)
	flowChan := make(chan interface{}, 100)
	logChan := make(chan interface{}, 2000) // make this channel big to handle export processes
	presChan := make(chan interface{}, 1000)
	serialChan := make(chan interface{}, 1000)
	networkChan := make(chan interface{}, 1000)
	updateChan := make(chan interface{}, 100)
	powerChan := make(chan interface{}, 100)

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
		{"update", updateChan},
		{"power", powerChan},
	}

	// fire up subsystems
	go keypad.Run(keypadChan, appChan, state.HWVersion)
	go isui.Run(uiChan, appChan, config, state, dbData)
	go isio.Run(ioChan, appChan, config, state) // this is where io Run is called, w/ ioChan as in chan and appChan as out chan
	go iscontrol.Run(cntrlChan, appChan, config, state)
	if params.WebUI {
		go isapi.Server(webChan, appChan)
	}
	go issim.Run(simChan, appChan)
	go islcd.Run(lcdChan, appChan)
	go isflow.Run(flowChan, appChan, params.Sim, config)
	go islog.Run(logChan, appChan, state, config, dbData)
	go ispressure.Run(presChan, appChan, config)
	//go isserial.Run(serialChan, appChan, config)
	go isnetwork.Run(networkChan, appChan, config, state,
		params.PortalURL, params.DebugPortal)

	go isupdate.Run(updateChan, appChan)
	go ispower.Run(powerChan, appChan)

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
		err := dbConfig.WriteConfig(&config)
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
			logChan <- state
			if params.WebUI {
				webChan <- state
			}
			logChan <- state
			powerChan <- state

			lastStateSend = now
		}

		if now.Sub(lastStateSendSlow) > 5*time.Second {
			networkChan <- state
			lastStateSendSlow = now
		}
	}

	if config.Arm && state.GpioRelayInjectorEn {
		state.TimeArmedAndInjOn = time.Now()
	}

	var lastHistoryDialogDisplay time.Time
	var lastVisionUnknownStateDisplay time.Time
	var lastChannelDialogDisplay time.Time

	// Define dialog POINTERS for use in the logic below
	dlgShutdown := state.Dialogs[isdata.DialogShutdownS]
	dlgReboot := state.Dialogs[isdata.DialogRebootS]
	dlgFactoryReset := state.Dialogs[isdata.DialogFactoryResetS]
	dlgSetTimezone := state.Dialogs[isdata.DialogSetTimezoneS]
	dlgUnknownVisionState := state.Dialogs[isdata.DialogUnknownVisionStateS]
	dlgApp := state.Dialogs[isdata.DialogAppS]
	dlgStateMachine := state.Dialogs[isdata.DialogStateMachineS]
	dlgExport := state.Dialogs[isdata.DialogExportS]
	dlgResetTotalCurrent := state.Dialogs[isdata.DialogResetTotalCurrentS]
	dlgResetTotal1 := state.Dialogs[isdata.DialogResetTotal1S]
	dlgResetTotal2 := state.Dialogs[isdata.DialogResetTotal2S]
	dlgHistoryData := state.Dialogs[isdata.DialogHistoryDataS]

	updateUsageStats := func() {
		state.DbSampleCount, err = dbData.GetSampleCount()
		state.DataUsage, err = system.FsPercentUsed("/data")
		state.RootUsage, err = system.FsPercentUsed("/")
		if state.DbSampleCount > 7500 && time.Since(lastHistoryDialogDisplay) > time.Hour*24 {
			log.Println("activating history dialog")
			dlgHistoryData.Active = true
			lastHistoryDialogDisplay = time.Now()
		}
	}

	// Save the state so that the database version from migrations
	// and the TimeArmedAndInjOn are saved
	updateUsageStats()
	saveState()

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

		dlgUpdate := state.Dialogs[isdata.DialogUpdateS]
		dlgUpdate.Active = true
		dlgUpdate.Message = "System updated to v" + state.OSVersion.String()
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

	exportingDataMessage := "Exporting data to USB Disk\n\nPlease Wait ..."
	mainloopFile := "/run/is-mainloop"
	for {
		if runtime.GOARCH == "arm" {
			if !file.Exists(mainloopFile) {
				f, err := os.Create(mainloopFile)
				if err != nil {
					log.Println("Error creating: ", mainloopFile)
				} else {
					f.Close()
				}
			}
		}

		// max sure queues between subsystems are not full
		for _, c := range channels {
			if len(c.channel) >= cap(c.channel)-1 {
				log.Println("Warning channel full: ", c.name, len(c.channel))
				e := <-c.channel
				log.Printf("dropping entry of type: %T\n", e)

				// Fire a dialog to let user know about problem
				if time.Since(lastChannelDialogDisplay) > time.Hour {
					dlgApp.Active = true
					dlgApp.Message = "System overloaded: " +
						c.name +
						"\nchannel is full. Please\nrestart system."
					lastChannelDialogDisplay = time.Now()
				}

			} else if len(c.channel) > cap(c.channel)/2 &&
				time.Now().Sub(lastFillingWarning) > time.Minute {
				log.Println("Warning channel is filling: ", c.name, len(c.channel))
				lastFillingWarning = time.Now()
			}
		}
		select {
		case s := <-sigChan:

			log.Println("Received signal: ", s)
			img := image.NewRGBA(image.Rect(0, 0, 128, 64))
			isui.Clear(img)
			isui.DrawPng(img, "IS_logo_injector.png", 26, 0)
			lcdChan <- isui.ImageToBlt(0, 0, img, false)

			// Config cleanup
			config.ManualRelayInj = isdata.RelayControlStateType(isdata.RelayControlStateAuto)
			config.ManualRelayAux = isdata.RelayControlStateType(isdata.RelayControlStateAuto)
			config.ManualRelayShutdown = isdata.RelayControlStateType(isdata.RelayControlStateAuto)
			saveConfig()

			// State cleanup
			//fmt.Println("COLLIN, arm", config.Arm, "injrelay", state.GpioRelayInjectorEn)
			if config.Arm && state.GpioRelayInjectorEn {
				state.DurationArmedAndInjOn += time.Since(state.TimeArmedAndInjOn)
			}
			saveState()

			err := dbState.WriteState(&state)
			if err != nil {
				log.Println("Error writing state to state db:", err)
			}
			err = dbConfig.WriteConfig(&config)
			if err != nil {
				log.Println("Error writing config to config db:", err)
			}

			// save config and state in data db as well for backup
			dbData.WriteConfig(&config)
			dbData.WriteState(&state)

			// give time for splash screen to be displayed
			time.Sleep(100 * time.Millisecond)
			log.Println("state and config saved, SEE YA!")
			os.Exit(0)

		case <-saveStateTimer.C:
			updateUsageStats()
			saveState()

			if stateDirty {
				dbState.WriteState(&state)
				stateDirty = false
			}
		case m := <-appChan:

			if state.ViewMsg {
				fmt.Printf("Message: %T\n", m)
			}

			switch m := m.(type) {
			case isdata.LcdPixel:
				if params.WebUI {
					webChan <- m
				}

			case isdata.LcdBlt:
				if params.WebUI {
					webChan <- m
				}
				lcdChan <- m

			case isdata.LcdBltSolid:
				if params.WebUI {
					webChan <- m
				}

			case isdata.Flow:
				state.FlowPulseCount += int(m.Pulses)
				// log flow data (engineering purposes)
				if config.LogFlowData {
					logChan <- m
				}

				/*
					// update short moving average window used
					config.FlowAvgWindowShortUsed = m.ShortWin
				*/

				saveState()
				saveConfig()

			case data.Sample:
				switch m.Type {
				case isdata.SampleTypeFlowWindowAvg:

					if config.Arm && state.GpioRelayInjectorEn {
						state.FlowAverager.AddSample(m)
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
					isdata.SampleTypeFaultPresHigh,
					isdata.SampleTypeFaultShutdown,
					isdata.SampleTypeFaultNtFlowOff,
					isdata.SampleTypeFaultNtPresLow,
					isdata.SampleTypeFaultNtPresHigh:
					state.FaultsActive = append(state.FaultsActive, m)
					saveState()

					logChan <- m
					networkChan <- m

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
					dbData.WriteSample(data.Sample{
						Type:  isdata.SampleTypeInputInjector,
						Time:  time.Now(),
						Value: boolToSampleVal(m.Bool()),
					})

				case isdata.SampleTypeSimGpioDigIrg:
					state.GpioDigitalIrrigator = m.Bool()
					saveState()

					// save to database for system logs
					dbData.WriteSample(data.Sample{
						Type:  isdata.SampleTypeInputIrrigator,
						Time:  time.Now(),
						Value: boolToSampleVal(m.Bool()),
					})

				case isdata.SampleTypeSimGpioDigWaterOn:
					state.GpioDigitalWaterOn = m.Bool()
					saveState()

					// save to database for system logs
					dbData.WriteSample(data.Sample{
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
						state.FlowAverager.ResetAverage()
					}
					saveConfig()
					saveState()

					if config.Arm != oldArm {
						// save to database for system logs
						dbData.WriteSample(data.Sample{
							Type:  isdata.SampleTypeArm,
							Time:  time.Now(),
							Value: boolToSampleVal(config.Arm),
						})
					}

				case isdata.SampleTypeSimPump:
					uiChan <- isdata.KeyPump

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
				case isdata.KeyArm, isdata.KeyArmKp:
					oldArm := config.Arm
					toggleArmOrOpenDialog(&config, &state)
					if config.Arm {
						state.FlowAverager.ResetAverage()
					}
					saveConfig()
					saveState()

					// send to logging thread to be saved to database for system logs
					// send to network thread to be sent to portal
					if config.Arm != oldArm {
						s := data.Sample{
							Type:  isdata.SampleTypeArm,
							Time:  time.Now(),
							Value: boolToSampleVal(config.Arm),
						}
						logChan <- s
						//networkChan <- s
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

			case isdata.UpdateTimezone:
				config.Timezone = string(m)
				saveConfig()

			case isdata.UpdatePulsesPerGallon:
				config.PulsesPerGallon = int(m)
				saveConfig()

			case isdata.SetBacklight:
				lcdChan <- m

			/*
				case isdata.UpdateFlowAvgWindow:
					config.FlowAvgWindow = int(m)
					saveConfig()
			*/

			case isdata.UpdateFlowAvgWindowLong:
				config.FlowAvgWindowLong = int(m)
				saveConfig()

			/*
				case isdata.UpdateFlowAvgPercDiff:
					config.FlowAvgPercDiff = int(m)
					saveConfig()
			*/

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

			case isdata.UpdateHighPres:
				config.HighPres = int(m)
				saveConfig()

			case isdata.UpdateManualHighAlarmGPH:
				config.ManualHighAlarmGPH = float64(m)
				saveConfig()

			case isdata.UpdateStateMachineState:
				state.StateMachineState = int(m.ID)
				saveState()
				err = dbData.WriteSample(data.Sample{
					Time: time.Now(),
					Type: isdata.SampleTypeState,
					ID:   m.Name,
				})

			case isdata.UpdateDisarm:
				config.Arm = false
				if state.GpioRelayInjectorEn {
					state.DurationArmedAndInjOn += time.Since(state.TimeArmedAndInjOn)
				}
				saveConfig()
				saveState()

			case isdata.UpdateResetFlowPulseCount:
				state.FlowPulseCount = 0
				saveState()

			case isdata.UpdateResetTotal1:
				// Confirmation dialog
				dlgResetTotal1.Active = true
				saveState()

			case isdata.UpdateResetTotal2:
				// Confirmation dialog
				dlgResetTotal2.Active = true
				saveState()

			case isdata.UpdateResetCurrentProduct:
				// Confirmation dialog
				dlgResetTotalCurrent.Active = true
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

			case isdata.UpdateModemDisabled:
				config.ModemDisabled = bool(m)
				saveConfig()
				networkChan <- m

			case isdata.SetTimezone:
				dlgSetTimezone.Active = true

			case isdata.ExportData:

				// we only want one export process running at a time
				if dlgExport.Active {
					break
				}

				logChan <- isdata.ExportData{}
				dlgExport.Active = true
				dlgExport.Heading = "Notice"
				dlgExport.Message = exportingDataMessage

			case isdata.ExportConfig:

				// we only want one export process running at a time
				if dlgExport.Active {
					break
				}

				logChan <- isdata.ExportConfig{}
				dlgExport.Active = true
				dlgExport.Heading = "Notice"
				dlgExport.Message = exportingDataMessage

			case isdata.ExportSystemLogs:

				// we only want one export process running at a time
				if dlgExport.Active {
					break
				}

				logChan <- isdata.ExportSystemLogs{}
				dlgExport.Active = true
				dlgExport.Heading = "Notice"
				dlgExport.Message = exportingDataMessage

			case isdata.ExportFieldProductTotals:
				if dlgExport.Active {
					// we only want one export process running at a time
					break
				}
				logChan <- isdata.ExportFieldProductTotals{}
				dlgExport.Active = true
				dlgExport.Heading = "Notice"
				dlgExport.Message = exportingDataMessage

			case isdata.ExportAlreadyInProcess:
				dlgExport.Active = true
				dlgExport.Heading = "Error"
				dlgExport.Message = "Export already in process\n\nPlease Wait ..."

			case isdata.ExportDataFinished:
				dlgExport.Active = true
				dlgExport.Heading = "Notice"
				dlgExport.Message = "Done exporting data to USB\nUSB device ejected\nPlease remove USB disk"

			case isdata.ExportConfigFinished:
				dlgExport.Active = true
				dlgExport.Heading = "Notice"
				dlgExport.Message = "Done exporting config to USB\nUSB device ejected\nPlease remove USB disk"

			case isdata.NoDiskPresent:
				dlgExport.Active = true
				dlgExport.Heading = "Error"
				dlgExport.Message = "No USB disk present\nPlease insert or reinsert\nUSB drive and try again"

			case isdata.ErrWriteDisk:
				dlgExport.Active = true
				dlgExport.Heading = "Error"
				dlgExport.Message = "Error writing to USB disk"

			/*case isdata.UpdateTankAlertVolume:
			config.TankAlertVolume = int(m)
			saveConfig()*/

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

				// send to logging thread to be saved to database for system logs
				// send to network thread to be sent to portal
				s := data.Sample{
					Type:  isdata.SampleTypeInputInjector,
					Time:  time.Now(),
					Value: boolToSampleVal(bool(m)),
				}
				logChan <- s
				//networkChan <- s

			case isdata.UpdateGpioDigitalIrrigator:
				state.GpioDigitalIrrigator = bool(m)
				saveState()

				// send to logging thread to be saved to database for system logs
				// send to network thread to be sent to portal
				s := data.Sample{
					Type:  isdata.SampleTypeInputIrrigator,
					Time:  time.Now(),
					Value: boolToSampleVal(bool(m)),
				}
				logChan <- s
				//networkChan <- s

			case isdata.UpdateGpioDigitalWaterOn:
				state.GpioDigitalWaterOn = bool(m)
				saveState()

				// send to logging thread to be saved to database for system logs
				// send to network thread to be sent to portal
				s := data.Sample{
					Type:  isdata.SampleTypeInputWaterOn,
					Time:  time.Now(),
					Value: boolToSampleVal(bool(m)),
				}
				logChan <- s
				//networkChan <- s

			case isdata.UpdateGpioDigitalIn:
				state.GpioDigitalIn = bool(m)
				saveState()

			case isdata.UpdateGpioMainAuxPower:
				state.GpioMainAuxPwr = bool(m)
				saveState()
				// send to logging thread to be saved to database for system logs
				// send to network thread to be sent to portal
				s := data.Sample{
					Type:  isdata.SampleTypeMainAuxPwr,
					Time:  time.Now(),
					Value: boolToSampleVal(bool(m)),
				}
				logChan <- s

			case isdata.UpdateManualRelayInj:
				config.ManualRelayInj = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdateManualRelayAux:
				config.ManualRelayAux = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdateManualRelayShutdown:
				config.ManualRelayShutdown = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdateManualRegValve1:
				config.ManualRegValve1 = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdateManualRegValve2:
				config.ManualRegValve2 = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdateManualRelayAll:
				config.ManualRelayInj = isdata.RelayControlStateType(m)
				config.ManualRelayAux = isdata.RelayControlStateType(m)
				config.ManualRelayShutdown = isdata.RelayControlStateType(m)
				config.ManualRegValve1 = isdata.RelayControlStateType(m)
				config.ManualRegValve2 = isdata.RelayControlStateType(m)
				saveConfig()

			case isdata.UpdatePressureShutdownEnabled:
				config.PressureShutdownEnabled = !config.PressureShutdownEnabled
				if !config.PressureShutdownEnabled {
					dlgApp.Active = true
					dlgApp.Message = "You just disabled low-\npressure shutdown"
				}
				saveConfig()

			case isdata.UpdatePressureStartupLow:
				config.PressureStartupLow = int(m)
				saveConfig()

			case isdata.UpdateGpioRelayInjector:

				old := state.GpioRelayInjectorEn
				state.GpioRelayInjectorEn = bool(m)

				if config.Arm && old != state.GpioRelayInjectorEn {
					if state.GpioRelayInjectorEn {
						state.TimeArmedAndInjOn = time.Now()
					} else {
						state.DurationArmedAndInjOn += time.Since(state.TimeArmedAndInjOn)
					}
				}

				saveState()

			case isdata.UpdateGpioRelayShutdown:
				state.GpioRelayShutdownEn = bool(m)
				saveState()

			case isdata.UpdateGpioRelayAux:
				state.GpioRelayAuxEn = bool(m)
				saveState()

			case isdata.UpdateGpioRegValve1:
				state.GpioRegValve1 = bool(m)
				saveState()

			case isdata.UpdateGpioRegValve2:
				state.GpioRegValve2 = bool(m)
				saveState()

			case isdata.UpdateOperatingMode:
				config.OperatingMode = isdata.ISOperatingMode(m)
				if config.OperatingMode == isdata.ISOperatingModeMonitor {
					config.Arm = false // system can't be armed in monitor only mode
					if state.GpioRelayInjectorEn {
						state.DurationArmedAndInjOn += time.Since(state.TimeArmedAndInjOn)
					}
				}
				saveConfig()
				saveState()

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
				dlgReboot.Active = true
				dlgReboot.Message = "Reboot started, please wait"
				if runtime.GOARCH != "arm" {
					log.Println("on development platform, not rebooting")
				} else {
					log.Println("Menu reboot selected, rebooting ...")
					err := exec.Command("reboot").Run()
					if err != nil {
						log.Println("Error running reboot command")
					}
				}

			case isdata.FactoryReset:
				dlgFactoryReset.Active = true
				saveState()

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
					dlgUnknownVisionState.Message = "Vision panel state is unknown.\nOutputs shutting off."
					dlgUnknownVisionState.Active = true
					lastVisionUnknownStateDisplay = time.Now()
				}

				state.LindsayRegs = m
				state.LindsayLastUpdate = time.Now()
				saveState()

			case isdata.UpdateFaultActiveClearAll:
				state.FaultsActive = nil
				saveState()

			case isdata.DialogClose:

				if _, exists := state.Dialogs[m.Key]; !exists {
					log.Println("Error from app thread, No such entry exists in Dialogs map: ", m.Key)
					break
				}

				dialogPointer := state.Dialogs[m.Key]

				dialogPointer.Active = false
				saveState()

				switch dialogPointer.ID {
				case isdata.DialogFactoryReset:

					// Delete all the samples to reset the data in the main db
					err := dbData.DeleteSamples()
					if err != nil {
						log.Println("Error resetting main db: ", err)
					}

					// For state and config db's, just null out the state and config and
					// write fresh structs to db's after initializing
					state = isdata.State{}
					config = isdata.Config{}

					// Make sure config version in state is zero so that when the config is
					// initialized all of the migrations are run and the config values
					// are set to their factory defaults
					state.DBConfig.DBVersion = 0
					isdata.InitState(&state)
					config.Init(&state)
					state.SerialNumber = params.SerialNumber
					state.ViewMsg = params.ViewMsg
					state.HWVersion = isio.GetHwID()

					saveState()
					saveConfig()

					err = dbState.WriteState(&state)
					if err != nil {
						log.Println("Error writing state to new db:", err)
					}
					err = dbConfig.WriteConfig(&config)
					if err != nil {
						log.Println("Error writing config to new db:", err)
					}

					// save config and state in data db as well for backup
					err = dbData.WriteState(&state)
					if err != nil {
						log.Println("Error writing state to new backup db:", err)
					}
					err = dbData.WriteConfig(&config)
					if err != nil {
						log.Println("Error writing config to new backup db:", err)
					}

					// Redefine dialog POINTERS
					dlgShutdown = state.Dialogs[isdata.DialogShutdownS]
					dlgReboot = state.Dialogs[isdata.DialogRebootS]
					dlgFactoryReset = state.Dialogs[isdata.DialogFactoryResetS]
					dlgSetTimezone = state.Dialogs[isdata.DialogSetTimezoneS]
					dlgUnknownVisionState = state.Dialogs[isdata.DialogUnknownVisionStateS]
					dlgApp = state.Dialogs[isdata.DialogAppS]
					dlgStateMachine = state.Dialogs[isdata.DialogStateMachineS]
					dlgExport = state.Dialogs[isdata.DialogExportS]
					dlgResetTotalCurrent = state.Dialogs[isdata.DialogResetTotalCurrentS]
					dlgResetTotal1 = state.Dialogs[isdata.DialogResetTotal1S]
					dlgResetTotal2 = state.Dialogs[isdata.DialogResetTotal2S]
					dlgHistoryData = state.Dialogs[isdata.DialogHistoryDataS]

					// Remove all log messages
					messageFiles := []string{"messages", "messages.0", "messages.1",
						"messages.2", "messages.3"}

					for _, f := range messageFiles {
						err := os.Remove(path.Join("/data/log", f))
						if err != nil {
							log.Println("Error removing log message file:", err)
						}
					}

				case isdata.DialogSetTimezone:

					err := system.SetTimezone("US", config.Timezone)
					if err != nil {
						log.Println("Error setting timezone:", err)
					}

					if runtime.GOARCH != "arm" {
						log.Println("on development platform, not restarting")
						break
					}

					// Start a detached process versus using Run() and
					// Creating a child process
					log.Println("Timezone updated, restarting app ...")
					err = exec.Command("/etc/init.d/isapp", "restart").Start()
					if err != nil {
						log.Println("Error restarting the app")
					}

				case isdata.DialogResetTotalCurrent:
					state.FieldStates[config.CurrentFieldIndex][config.CurrentProductIndex].Total = 0

				case isdata.DialogResetTotal1:
					state.Total1 = 0

				case isdata.DialogResetTotal2:
					state.Total2 = 0

				case isdata.DialogStateMachine:
					dlgStateMachine.Ack = true

				}

				saveState()

			case isdata.DialogCancel:

				if _, exists := state.Dialogs[m.Key]; !exists {
					log.Println("Error from app thread, No such entry exists in Dialogs map: ", m.Key)
					break
				}

				dialogPointer := state.Dialogs[m.Key]

				switch dialogPointer.ID {

				// set the timezone value in config back to previous,
				// because it won't be set since user cancelled restart
				case isdata.DialogSetTimezone:
					_, zone, err := system.GetTimezone()
					if err != nil {
						log.Println("Error reading the timezone:", err)
					}

					config.Timezone = zone
				}

				dialogPointer.Active = false
				saveState()
				saveConfig()

			case isdata.UpdateDialogStateMachine:
				dlgStateMachine.Heading = string(m.Heading)
				dlgStateMachine.Message = string(m.Message)
				dlgStateMachine.Active = true
				dlgStateMachine.Ack = false
				saveState()

			case isdata.HelpScreenContent:
				config.HelpScreen.Name = m.Name
				config.HelpScreen.Text = m.Text
				config.HelpScreen.Active = true
				saveConfig()

			case isdata.HelpScreenClose:
				config.HelpScreen.Name = ""
				config.HelpScreen.Text = ""
				config.HelpScreen.Active = false
				saveConfig()

			case isdata.PanelDefinition:
				//newPanelType(m)

			case isdata.PanelType:
				config.PanelType = m
				saveConfig()

			case isdata.NetworkState:
				state.NetworkState = m
				saveState()

			case network.InterfaceConfig:
				state.NetworkInterfaceConfig = m
				saveState()

			case data.DeviceCmd:
				switch m.Cmd {
				case data.CmdUpdateApp:
					updateChan <- m
				case isdata.CmdFillTank:
					state.CurrentTankVolume = float64(config.TankCapacity)
					saveState()
				default:
					log.Println("Unknown command from portal: ", m)
				}

			case isdata.Shutdown:
				dlgShutdown.Active = true
				saveState()

			case data.GpsPos:
				state.Location = m
				saveState()

			default:
				// \r is required below to handle unknown keycode messages -- not sure why
				log.Printf("App Mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		}
	}
}

func toggleArmOrOpenDialog(config *isdata.Config, state *isdata.State) {

	dlgArm := state.Dialogs[isdata.DialogArmS]
	dlgArmReq := state.Dialogs[isdata.DialogArmReqS]

	if config.OperatingMode == isdata.ISOperatingModeMonitor {
		dlgArm.Active = true
		dlgArm.Message = "Cannot arm in Monitor \nOnly mode, please switch \nmodes"
		return
	}
	if config.UserPumpMode == isdata.UserPumpModeNotSet {
		dlgArm.Active = true
		dlgArm.Message = "Pump Command \nInput not selected, please \nselect before arming"
		return
	}

	if !config.Arm { // if the arm switch will be turned on
		if isdata.AllArmReqMet(config, state) {
			config.Arm = true                      // config.Arm = !config.Arm since Arm was false
			config.FlowRateTarget = state.FlowRate // set target flow rate to current
			config.PressureShutdownLow = state.PressureMin - state.PressureMin*config.LowPresPerc/100
			state.TimeArmedAndInjOn = time.Now()
			state.DurationArmedAndInjOn = 0
		} else {
			dlgArmReq.Active = true
		}
	} else {
		config.Arm = false
		if state.GpioRelayInjectorEn {
			state.DurationArmedAndInjOn += time.Since(state.TimeArmedAndInjOn)
		}
	}
}

func boolToSampleVal(b bool) float64 {
	if b {
		return float64(1)
	}
	return float64(0)
}
