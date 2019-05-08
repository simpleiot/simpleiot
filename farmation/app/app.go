package app

import (
	"fmt"
	"log"
	"os"
	"os/signal"
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

	config := isdata.Config{}
	state := isdata.State{}
	stateDirty := false

	err = db.ReadConfig(&config)

	if err != nil {
		log.Println("Error reading config: ", err)
	}

	err = db.ReadState(&state)

	if err != nil {
		log.Println("Error reading state: ", err)
	}

	stateDirty = isdata.InitState(&state)
	config.Init()

	// incoming channel to mux
	appChan := make(chan interface{}, 100)

	// outgoing channels to various other parts of the system
	keypadChan := make(chan interface{}, 100)
	uiChan := make(chan interface{}, 100)
	ioChan := make(chan interface{}, 100)
	webChan := make(chan interface{}, 100)
	simChan := make(chan interface{}, 100)
	lcdChan := make(chan interface{}, 100)
	flowChan := make(chan interface{}, 100)
	logChan := make(chan interface{}, 1000)

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
	}

	// fire up subsystems
	go keypad.Run(keypadChan, appChan)
	go isui.Run(uiChan, appChan, &config)
	go isio.Run(ioChan, appChan)
	go isapi.Server(webChan, appChan)
	go issim.Run(simChan, appChan)
	go islcd.Run(lcdChan, appChan)
	go isflow.Run(flowChan, appChan, sim)
	go islog.Run(logChan, appChan)

	lastFillingWarning := time.Time{}

	saveConfig := func() {
		if debugConfig {
			fmt.Printf("Config: %+v\n", config)
		}
		uiChan <- config
		flowChan <- config
		logChan <- config
		err := db.WriteConfig(&config)
		if err != nil {
			log.Println("Error saving config: ", err)
		}
	}

	saveState := func() {
		if debugState {
			fmt.Printf("State: %+v\n", state)
		}
		stateDirty = true
		uiChan <- state
	}

	saveStateTimer := time.NewTicker(time.Minute)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	for {
		// max sure queues between subsystems are not full
		for _, c := range channels {
			if len(c.channel) >= cap(c.channel)-1 {
				log.Println("Warning channel full: ", c.name, len(c.channel))
				log.Println("dropping entry: ", c.name, len(c.channel))
				<-c.channel
			} else if len(c.channel) > 30 &&
				time.Now().Sub(lastFillingWarning) > time.Minute {
				log.Println("Warning channel is filling: ", c.name, len(c.channel))
				lastFillingWarning = time.Now()
			}
		}
		select {
		case s := <-sigChan:
			fmt.Println("Received signal: ", s)
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
				case isdata.SampleTypeFlowRate:
					state.ProcessSample(m)
					uiChan <- state
				case isdata.SampleTypeKey:
					// convert from sample to key
					uiChan <- isdata.KeyFromString(m.ID)
				}
			case isdata.Key:
				uiChan <- m

			case isdata.UpdateFieldName:
				config.FieldConfigs[m.Index].Description = m.Name
				saveConfig()

			case isdata.Flow:
				state.FlowRate = m.Rate
				state.Total1 += m.Amount
				state.Total2 += m.Amount
				if config.LogFlowData {
					logChan <- m
				}
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

			case isdata.UpdateLogFlowEnable:
				config.LogFlowData = bool(m)
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

			case isdata.Pulse:
				logChan <- m

			default:
				// \r is required below to handle unknown keycode messages -- not sure why
				log.Printf("App Mux: unhandled message of type %T: %+v\r\n", m, m)

			}
		}
	}
}
